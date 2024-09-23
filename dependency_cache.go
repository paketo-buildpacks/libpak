/*
 * Copyright 2018-2020 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package libpak

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/libcnb"
	"github.com/heroku/color"

	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/sherpa"
)

type HttpClientTimeouts struct {
	DialerTimeout         time.Duration
	DialerKeepAlive       time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	ExpectContinueTimeout time.Duration
}

// DependencyCache allows a user to get an artifact either from a buildpack's cache, a previous download,
// a mirror registry, or to download directly.
type DependencyCache struct {

	// CachePath is the location where the buildpack has cached its dependencies.
	CachePath string

	// DownloadPath is the location of all downloads during this execution of the build.
	DownloadPath string

	// Logger is the logger used to write to the console.
	Logger bard.Logger

	// UserAgent is the User-Agent string to use with requests.
	UserAgent string

	// Mappings optionally provides URIs mapping for BuildpackDependencies
	Mappings map[string]string

	// httpClientTimeouts contains the timeout values used by HTTP client
	HttpClientTimeouts HttpClientTimeouts

	// Alternative sources used for downloading dependencies.
	DependencyMirrors map[string]string
}

// NewDependencyCache creates a new instance setting the default cache path (<BUILDPACK_PATH>/dependencies) and user
// agent (<BUILDPACK_ID>/<BUILDPACK_VERSION>).
// Mappings will be read from any libcnb.Binding in the context with type "dependency-mappings".
//
// In some environments, many dependencies might need to be downloaded from a (local) mirror registry or filesystem.
// Such alternative locations can be configured using bindings of type "dependency-mirror", avoiding too many "dependency-mapping" bindings.
// Environment variables named "BP_DEPENDENCY_MIRROR" (default) or "BP_DEPENDENCY_MIRROR_<HOSTNAME>" (hostname-specific mirror)
// can also be used for the same purpose.
func NewDependencyCache(context libcnb.BuildContext) (DependencyCache, error) {
	cache := DependencyCache{
		CachePath:         filepath.Join(context.Buildpack.Path, "dependencies"),
		DownloadPath:      os.TempDir(),
		UserAgent:         fmt.Sprintf("%s/%s", context.Buildpack.Info.ID, context.Buildpack.Info.Version),
		Mappings:          map[string]string{},
		DependencyMirrors: map[string]string{},
		// We create the logger here because the initialization process may log some warnings that should be visible to users.
		// This goes against the usual pattern, which has the user supply the Logger after initialization.
		// There's no choice though, if we want the warning messages to be visible to users. We should clean this up in v2.
		Logger: bard.NewLogger(os.Stdout),
	}
	mappings, err := filterBindingsByType(context.Platform.Bindings, "dependency-mapping")
	if err != nil {
		return DependencyCache{}, fmt.Errorf("unable to process dependency-mapping bindings\n%w", err)
	}
	cache.Mappings = mappings

	clientTimeouts, err := customizeHttpClientTimeouts()
	if err != nil {
		return DependencyCache{}, fmt.Errorf("unable to read custom timeout settings\n%w", err)
	}
	cache.HttpClientTimeouts = *clientTimeouts

	bindingMirrors, err := filterBindingsByType(context.Platform.Bindings, "dependency-mirror")
	if err != nil {
		return DependencyCache{}, fmt.Errorf("unable to process dependency-mirror bindings\n%w", err)
	}
	cache.setDependencyMirrors(bindingMirrors)

	return cache, nil
}

func customizeHttpClientTimeouts() (*HttpClientTimeouts, error) {
	rawStr := sherpa.GetEnvWithDefault("BP_DIALER_TIMEOUT", "6")
	dialerTimeout, err := strconv.Atoi(rawStr)
	if err != nil {
		return nil, fmt.Errorf("unable to convert BP_DIALER_TIMEOUT=%s to integer\n%w", rawStr, err)
	}

	rawStr = sherpa.GetEnvWithDefault("BP_DIALER_KEEP_ALIVE", "60")
	dialerKeepAlive, err := strconv.Atoi(rawStr)
	if err != nil {
		return nil, fmt.Errorf("unable to convert BP_DIALER_KEEP_ALIVE=%s to integer\n%w", rawStr, err)
	}

	rawStr = sherpa.GetEnvWithDefault("BP_TLS_HANDSHAKE_TIMEOUT", "5")
	tlsHandshakeTimeout, err := strconv.Atoi(rawStr)
	if err != nil {
		return nil, fmt.Errorf("unable to convert BP_TLS_HANDSHAKE_TIMEOUT=%s to integer\n%w", rawStr, err)
	}

	rawStr = sherpa.GetEnvWithDefault("BP_RESPONSE_HEADER_TIMEOUT", "5")
	responseHeaderTimeout, err := strconv.Atoi(rawStr)
	if err != nil {
		return nil, fmt.Errorf("unable to convert BP_RESPONSE_HEADER_TIMEOUT=%s to integer\n%w", rawStr, err)
	}

	rawStr = sherpa.GetEnvWithDefault("BP_EXPECT_CONTINUE_TIMEOUT", "1")
	expectContinueTimeout, err := strconv.Atoi(rawStr)
	if err != nil {
		return nil, fmt.Errorf("unable to convert BP_EXPECT_CONTINUE_TIMEOUT=%s to integer\n%w", rawStr, err)
	}

	return &HttpClientTimeouts{
		DialerTimeout:         time.Duration(dialerTimeout) * time.Second,
		DialerKeepAlive:       time.Duration(dialerKeepAlive) * time.Second,
		TLSHandshakeTimeout:   time.Duration(tlsHandshakeTimeout) * time.Second,
		ResponseHeaderTimeout: time.Duration(responseHeaderTimeout) * time.Second,
		ExpectContinueTimeout: time.Duration(expectContinueTimeout) * time.Second,
	}, nil
}

func (d *DependencyCache) setDependencyMirrors(bindingMirrors map[string]string) {
	// Initialize with mirrors from bindings.
	d.DependencyMirrors = bindingMirrors
	// Add mirrors from env variables and override duplicate hostnames set in bindings.
	envs := os.Environ()
	for _, env := range envs {
		envPair := strings.SplitN(env, "=", 2)
		if len(envPair) != 2 {
			continue
		}
		hostnameSuffix, isMirror := strings.CutPrefix(envPair[0], "BP_DEPENDENCY_MIRROR")
		if isMirror {
			hostnameEncoded, _ := strings.CutPrefix(hostnameSuffix, "_")
			if strings.ToLower(hostnameEncoded) == "default" {
				d.Logger.Bodyf("%s with illegal hostname 'default'. Please use BP_DEPENDENCY_MIRROR to set a default.",
					color.YellowString("Ignored dependency mirror"))
				continue
			}
			d.DependencyMirrors[decodeHostnameEnv(hostnameEncoded, d)] = envPair[1]
		}
	}
}

// Takes an encoded hostname (from env key) and returns the decoded version in lower case.
// Replaces double underscores (__) with one dash (-) and single underscores (_) with one period (.).
func decodeHostnameEnv(encodedHostname string, d *DependencyCache) string {
	if strings.ContainsAny(encodedHostname, "-.") || encodedHostname != strings.ToUpper(encodedHostname) {
		d.Logger.Bodyf("%s These will be allowed but for best results across different shells, you should replace . characters with _ characters "+
			"and - characters with __, and use all upper case letters. The buildpack will convert these back before using the mirror.",
			color.YellowString("You have invalid characters in your mirror host environment variable."))
	}
	var decodedHostname string
	if encodedHostname == "" {
		decodedHostname = "default"
	} else {
		decodedHostname = strings.ReplaceAll(strings.ReplaceAll(encodedHostname, "__", "-"), "_", ".")
	}
	return strings.ToLower(decodedHostname)
}

// Returns a key/value map with all entries for a given binding type.
// An error is returned if multiple entries are found using the same key (e.g. duplicate digests in dependency mappings).
func filterBindingsByType(bindings libcnb.Bindings, bindingType string) (map[string]string, error) {
	filteredBindings := map[string]string{}
	for _, binding := range bindings {
		if strings.ToLower(binding.Type) == bindingType {
			for key, value := range binding.Secret {
				if _, ok := filteredBindings[strings.ToLower(key)]; ok {
					return nil, fmt.Errorf("multiple %s bindings found with duplicate keys %s", binding.Type, key)
				}
				filteredBindings[strings.ToLower(key)] = value
			}
		}
	}
	return filteredBindings, nil
}

// RequestModifierFunc is a callback that enables modification of a download request before it is sent.  It is often
// used to set Authorization headers.
type RequestModifierFunc func(request *http.Request) (*http.Request, error)

// Artifact returns the path to the artifact.  Resolution of that path follows three tiers:
//
// 1. CachePath
// 2. DownloadPath
// 3. Download from URI
//
// If the BuildpackDependency's SHA256 is not set, the download can never be verified to be up to date and will always
// download, skipping all the caches.
func (d *DependencyCache) Artifact(dependency BuildpackDependency, mods ...RequestModifierFunc) (*os.File, error) {

	var (
		actual    BuildpackDependency
		artifact  string
		file      string
		isBinding bool
		uri       = dependency.URI
		urlP      *url.URL
	)

	for d, u := range d.Mappings {
		if d == dependency.SHA256 {
			isBinding = true
			uri = u
			break
		}
	}

	urlP, err := url.Parse(uri)
	if err != nil {
		d.Logger.Debugf("URI format invalid\n%w", err)
		return nil, fmt.Errorf("unable to parse URI. see DEBUG log level")
	}

	mirror := d.DependencyMirrors["default"]
	mirrorHostSpecific := d.DependencyMirrors[urlP.Hostname()]
	if mirrorHostSpecific != "" {
		mirror = mirrorHostSpecific
	}

	if isBinding && mirror != "" {
		d.Logger.Bodyf("Both dependency mirror and bindings are present. %s Please remove dependency map bindings if you wish to use the mirror.",
			color.YellowString("Mirror is being ignored."))
	} else {
		d.setDependencyMirror(urlP, mirror)
	}

	if dependency.SHA256 == "" {
		d.Logger.Headerf("%s Dependency has no SHA256. Skipping cache.",
			color.New(color.FgYellow, color.Bold).Sprint("Warning:"))

		d.Logger.Bodyf("%s from %s", color.YellowString("Downloading"), urlP.Redacted())
		artifact = filepath.Join(d.DownloadPath, filepath.Base(uri))
		if err := d.download(urlP, artifact, mods...); err != nil {
			return nil, fmt.Errorf("unable to download %s\n%w", urlP.Redacted(), err)
		}

		return os.Open(artifact)
	}

	file = filepath.Join(d.CachePath, fmt.Sprintf("%s.toml", dependency.SHA256))
	b, err := os.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to read %s\n%w", file, err)
	}
	if err := toml.Unmarshal(b, &actual); err != nil {
		return nil, fmt.Errorf("unable to decode download metadata %s\n%w", file, err)
	}

	if dependency.Equals(actual) {
		d.Logger.Bodyf("%s cached download from buildpack", color.GreenString("Reusing"))
		return os.Open(filepath.Join(d.CachePath, dependency.SHA256, filepath.Base(urlP.Path)))
	}

	file = filepath.Join(d.DownloadPath, fmt.Sprintf("%s.toml", dependency.SHA256))
	b, err = os.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to read %s\n%w", file, err)
	}
	if err := toml.Unmarshal(b, &actual); err != nil {
		return nil, fmt.Errorf("unable to decode download metadata %s\n%w", file, err)
	}

	if dependency.Equals(actual) {
		d.Logger.Bodyf("%s previously cached download", color.GreenString("Reusing"))
		return os.Open(filepath.Join(d.DownloadPath, dependency.SHA256, filepath.Base(urlP.Path)))
	}

	d.Logger.Bodyf("%s from %s", color.YellowString("Downloading"), urlP.Redacted())
	artifact = filepath.Join(d.DownloadPath, dependency.SHA256, filepath.Base(uri))
	if err := d.download(urlP, artifact, mods...); err != nil {
		return nil, fmt.Errorf("unable to download %s\n%w", urlP.Redacted(), err)
	}

	d.Logger.Body("Verifying checksum")
	if err := d.verify(artifact, dependency.SHA256); err != nil {
		return nil, err
	}

	file = filepath.Join(d.DownloadPath, fmt.Sprintf("%s.toml", dependency.SHA256))
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		return nil, fmt.Errorf("unable to make directory %s\n%w", filepath.Dir(file), err)
	}

	out, err := os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return nil, fmt.Errorf("unable to open file %s\n%w", file, err)
	}
	defer out.Close()

	if err := toml.NewEncoder(out).Encode(dependency); err != nil {
		return nil, fmt.Errorf("unable to write metadata %s\n%w", file, err)
	}

	return os.Open(artifact)
}

func (d DependencyCache) download(url *url.URL, destination string, mods ...RequestModifierFunc) error {
	if url.Scheme == "file" {
		return d.downloadFile(url.Path, destination, mods...)
	}

	return d.downloadHttp(url, destination, mods...)
}

func (d DependencyCache) downloadFile(source string, destination string, mods ...RequestModifierFunc) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return fmt.Errorf("unable to make directory %s\n%w", filepath.Dir(destination), err)
	}

	out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to open destination file %s\n%w", destination, err)
	}
	defer out.Close()

	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("unable to open source file %s\n%w", source, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, input); err != nil {
		return fmt.Errorf("unable to copy from %s to %s\n%w", source, destination, err)
	}

	return nil
}

func (d DependencyCache) downloadHttp(url *url.URL, destination string, mods ...RequestModifierFunc) error {
	var httpClient *http.Client
	if (strings.EqualFold(url.Hostname(), "localhost")) || (strings.EqualFold(url.Hostname(), "127.0.0.1")) {
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	} else {
		httpClient = &http.Client{
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout:   d.HttpClientTimeouts.DialerTimeout,
					KeepAlive: d.HttpClientTimeouts.DialerKeepAlive,
				}).Dial,
				TLSHandshakeTimeout:   d.HttpClientTimeouts.TLSHandshakeTimeout,
				ResponseHeaderTimeout: d.HttpClientTimeouts.ResponseHeaderTimeout,
				ExpectContinueTimeout: d.HttpClientTimeouts.ExpectContinueTimeout,
				Proxy:                 http.ProxyFromEnvironment,
			},
		}
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return fmt.Errorf("unable to create new GET request for %s\n%w", url.Redacted(), err)
	}

	if d.UserAgent != "" {
		req.Header.Set("User-Agent", d.UserAgent)
	}

	for _, m := range mods {
		req, err = m(req)
		if err != nil {
			return fmt.Errorf("unable to modify request\n%w", err)
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to request %s\n%w", url.Redacted(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("could not download %s: %d", url.Redacted(), resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return fmt.Errorf("unable to make directory %s\n%w", filepath.Dir(destination), err)
	}

	out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to open file %s\n%w", destination, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("unable to copy from %s to %s\n%w", url.Redacted(), destination, err)
	}

	return nil
}

func (DependencyCache) verify(path string, expected string) error {
	s := sha256.New()

	in, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("unable to verify %s\n%w", path, err)
	}
	defer in.Close()

	if _, err := io.Copy(s, in); err != nil {
		return fmt.Errorf("unable to read %s\n%w", path, err)
	}

	actual := hex.EncodeToString(s.Sum(nil))

	if expected != actual {
		return fmt.Errorf("sha256 for %s %s does not match expected %s", path, actual, expected)
	}

	return nil
}

func (d DependencyCache) setDependencyMirror(urlD *url.URL, mirror string) {
	if mirror != "" {
		d.Logger.Bodyf("%s Download URIs will be overridden.", color.GreenString("Dependency mirror found."))
		mirrorArgs := parseMirror(mirror)
		urlOverride, err := url.ParseRequestURI(mirrorArgs["mirror"])

		if strings.ToLower(urlOverride.Scheme) == "https" || strings.ToLower(urlOverride.Scheme) == "file" {
			urlD.Scheme = urlOverride.Scheme
			urlD.User = urlOverride.User
			urlD.Path = strings.Replace(urlOverride.Path, "{originalHost}", urlD.Hostname(), 1) + strings.Replace(urlD.Path, mirrorArgs["skip-path"], "", 1)
			urlD.Host = urlOverride.Host
		} else {
			d.Logger.Debugf("Dependency mirror URI is invalid: %s\n%w", mirror, err)
			d.Logger.Bodyf("%s is ignored. Have you used one of the supported schemes https:// or file://?", color.YellowString("Invalid dependency mirror"))
		}
	}
}

// Parses a raw mirror string into a map of arguments.
func parseMirror(mirror string) map[string]string {

	mirrorArgs := map[string]string{
		"mirror":    mirror,
		"skip-path": "",
	}

	// Split mirror string at commas and extract specified arguments.
	for _, arg := range strings.SplitN(mirror, ",", -1) {
		argPair := strings.SplitN(arg, "=", 2)
		// If a URI is provided without the key 'mirror=', still treat it as the 'mirror' argument.
		// This addresses backwards compatibility and user experience as most mirrors won't need any additional arguments.
		if len(argPair) == 1 && (strings.HasPrefix(argPair[0], "https") || strings.HasPrefix(argPair[0], "file")) {
			mirrorArgs["mirror"] = argPair[0]
		}
		// Add all provided arguments to key/value map.
		if len(argPair) == 2 {
			mirrorArgs[argPair[0]] = argPair[1]
		}
	}

	// Unescape mirror arguments to support URL-encoded strings.
	tmp, err := url.PathUnescape(mirrorArgs["mirror"])
	if err == nil {
		mirrorArgs["mirror"] = tmp
	}
	tmp, err = url.PathUnescape(mirrorArgs["skip-path"])
	if err == nil {
		mirrorArgs["skip-path"] = tmp
	}

	return mirrorArgs
}
