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

// DependencyCache allows a user to get an artifact either from a buildpack's cache, a previous download, or to download
// directly.
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
}

// NewDependencyCache creates a new instance setting the default cache path (<BUILDPACK_PATH>/dependencies) and user
// agent (<BUILDPACK_ID>/<BUILDPACK_VERSION>).
// Mappings will be read from any libcnb.Binding in the context with type "dependency-mappings"
func NewDependencyCache(context libcnb.BuildContext) (DependencyCache, error) {
	cache := DependencyCache{
		CachePath:    filepath.Join(context.Buildpack.Path, "dependencies"),
		DownloadPath: os.TempDir(),
		UserAgent:    fmt.Sprintf("%s/%s", context.Buildpack.Info.ID, context.Buildpack.Info.Version),
		Mappings:     map[string]string{},
	}
	mappings, err := mappingsFromBindings(context.Platform.Bindings)
	if err != nil {
		return DependencyCache{}, fmt.Errorf("unable to process dependency-mapping bindings\n%w", err)
	}
	cache.Mappings = mappings

	clientTimeouts, err := customizeHttpClientTimeouts()
	if err != nil {
		return DependencyCache{}, fmt.Errorf("unable to read custom timeout settings\n%w", err)
	}
	cache.HttpClientTimeouts = *clientTimeouts

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

func mappingsFromBindings(bindings libcnb.Bindings) (map[string]string, error) {
	mappings := map[string]string{}
	for _, binding := range bindings {
		if strings.ToLower(binding.Type) == "dependency-mapping" {
			for digest, uri := range binding.Secret {
				if _, ok := mappings[digest]; ok {
					return nil, fmt.Errorf("multiple mappings for digest %q", digest)
				}
				mappings[digest] = uri
			}
		}
	}
	return mappings, nil
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
		actual   BuildpackDependency
		artifact string
		file     string
		uri      = dependency.URI
	)

	for d, u := range d.Mappings {
		if d == dependency.SHA256 {
			uri = u
			break
		}
	}

	if dependency.SHA256 == "" {
		d.Logger.Headerf("%s Dependency has no SHA256. Skipping cache.",
			color.New(color.FgYellow, color.Bold).Sprint("Warning:"))

		d.Logger.Bodyf("%s from %s", color.YellowString("Downloading"), uri)

		// ensure query parameters are not included in the downloaded file name if the uri is http type
		downloadUri, err := url.Parse(uri)
		if err != nil {
			return nil, fmt.Errorf("unable to parse the download uri %s\n%w", uri, err)
		}

		if(downloadUri.Scheme == "http" || downloadUri.Scheme == "https") {
			artifact = filepath.Join(d.DownloadPath, filepath.Base(downloadUri.Path))
		} else {
			artifact = filepath.Join(d.DownloadPath, filepath.Base(uri))
		}

		if err := d.download(uri, artifact, mods...); err != nil {
			return nil, fmt.Errorf("unable to download %s\n%w", uri, err)
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
		return os.Open(filepath.Join(d.CachePath, dependency.SHA256, filepath.Base(uri)))
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
		return os.Open(filepath.Join(d.DownloadPath, dependency.SHA256, filepath.Base(uri)))
	}

	d.Logger.Bodyf("%s from %s", color.YellowString("Downloading"), uri)
	artifact = filepath.Join(d.DownloadPath, dependency.SHA256, filepath.Base(uri))
	if err := d.download(uri, artifact, mods...); err != nil {
		return nil, fmt.Errorf("unable to download %s\n%w", uri, err)
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

func (d DependencyCache) download(uri string, destination string, mods ...RequestModifierFunc) error {
	url, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("unable to parse URI %s\n%w", uri, err)
	}

	if url.Scheme == "file" {
		return d.downloadFile(url.Path, destination, mods...)
	}

	return d.downloadHttp(uri, destination, mods...)
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

func (d DependencyCache) downloadHttp(uri string, destination string, mods ...RequestModifierFunc) error {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return fmt.Errorf("unable to create new GET request for %s\n%w", uri, err)
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

	client := http.Client{
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
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("unable to request %s\n%w", uri, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("could not download %s: %d", uri, resp.StatusCode)
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
		return fmt.Errorf("unable to copy from %s to %s\n%w", uri, destination, err)
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
