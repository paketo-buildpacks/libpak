/*
 * Copyright 2018-2023 the original author or authors.
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
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/heroku/color"

	"github.com/paketo-buildpacks/libpak/v2/log"
	"github.com/paketo-buildpacks/libpak/v2/sbom"
)

// BuildpackConfiguration represents a build or launch configuration parameter.
type BuildModuleConfiguration struct {

	// Build indicates whether the configuration is for build-time.  Optional.
	Build bool `toml:"build"`

	// Default is the default value of the configuration parameter.  Optional.
	Default string `toml:"default"`

	// Description is the description of the configuration parameter.
	Description string `toml:"description"`

	// Launch indicates whether the configuration is for launch-time.  Optional.
	Launch bool `toml:"launch"`

	// Name is the environment variable name of the configuration parameter.
	Name string `toml:"name"`
}

// BuildModuleDependencyLicense represents a license that a BuildModuleDependency is distributed under.  At least one of
// Name or URI MUST be specified.
type BuildModuleDependencyLicense struct {

	// Type is the type of the license.  This is typically the SPDX short identifier.
	Type string `toml:"type"`

	// URI is the location where the license can be found.
	URI string `toml:"uri"`
}

// BuildModuleDependency describes a dependency known to the buildpack/extension
type BuildModuleDependency struct {
	// ID is the dependency ID.
	ID string `toml:"id"`

	// Name is the dependency name.
	Name string `toml:"name"`

	// Version is the dependency version.
	Version string `toml:"version"`

	// URI is the dependency URI.
	URI string `toml:"uri"`

	// SHA256 is the hash of the dependency.
	SHA256 string `toml:"sha256"`

	// Stacks are the stacks the dependency is compatible with.
	Stacks []string `toml:"stacks"`

	// Licenses are the licenses the dependency is distributed under.
	Licenses []BuildModuleDependencyLicense `toml:"licenses"`

	// CPEs are the Common Platform Enumeration identifiers for the dependency
	CPEs []string `toml:"cpes"`

	// PURL is the package URL that identifies the dependency
	PURL string `toml:"purl"`

	// DeprecationDate is the time when the dependency is deprecated
	DeprecationDate time.Time `toml:"deprecation_date"`
}

// Equals compares the 2 structs if they are equal. This is very simiar to reflect.DeepEqual
// except that properties that will not work (e.g. DeprecationDate) are ignored.
func (b1 BuildModuleDependency) Equals(b2 BuildModuleDependency) bool {
	b1.DeprecationDate = b1.DeprecationDate.Truncate(time.Second).In(time.UTC)
	b2.DeprecationDate = b2.DeprecationDate.Truncate(time.Second).In(time.UTC)

	if len(b1.CPEs) == 0 {
		b1.CPEs = nil
	}
	if len(b2.CPEs) == 0 {
		b2.CPEs = nil
	}

	return reflect.DeepEqual(b1, b2)
}

// AsSyftArtifact renders a bill of materials entry describing the dependency as Syft.
func (b BuildModuleDependency) AsSyftArtifact(source string) (sbom.SyftArtifact, error) {
	licenses := []string{}
	for _, license := range b.Licenses {
		licenses = append(licenses, license.Type)
	}

	sbomArtifact := sbom.SyftArtifact{
		Name:      b.Name,
		Version:   b.Version,
		Type:      "UnknownPackage",
		FoundBy:   "libpak",
		Licenses:  licenses,
		Locations: []sbom.SyftLocation{{Path: source}},
		CPEs:      b.CPEs,
		PURL:      b.PURL,
	}

	var err error
	sbomArtifact.ID, err = sbomArtifact.Hash()
	if err != nil {
		return sbom.SyftArtifact{}, fmt.Errorf("unable to generate hash\n%w", err)
	}

	return sbomArtifact, nil
}

func (b BuildModuleDependency) IsDeprecated() bool {
	deprecationDate := b.DeprecationDate.UTC()
	now := time.Now().UTC()
	return deprecationDate.Equal(now) || deprecationDate.Before(now)
}

func (b BuildModuleDependency) IsSoonDeprecated() bool {
	deprecationDate := b.DeprecationDate.UTC()
	now := time.Now().UTC()
	return deprecationDate.Add(-30*24*time.Hour).Before(now) && deprecationDate.After(now)
}

// BuildpackMetadata is an extension to libcnb.Buildpack / libcnb.Extension's metadata with opinions.
type BuildModuleMetadata struct {

	// Configurations are environment variables that can be used at build time to configure the buildpack and launch
	// time to configure the application.
	Configurations []BuildModuleConfiguration

	// Dependencies are the dependencies known to the buildpack.
	Dependencies []BuildModuleDependency

	// IncludeFiles describes the files to include in the package.
	IncludeFiles []string

	// PrePackage describes a command to invoke before packaging.
	PrePackage string
}

// NewBuildpackMetadata creates a new instance of BuildpackMetadata from the contents of libcnb.Buildpack.Metadata
func NewBuildModuleMetadata(metadata map[string]interface{}) (BuildModuleMetadata, error) {
	m := BuildModuleMetadata{}

	if v, ok := metadata["configurations"]; ok {
		for _, v := range v.([]map[string]interface{}) {
			var c BuildModuleConfiguration

			if v, ok := v["build"].(bool); ok {
				c.Build = v
			}

			if v, ok := v["default"].(string); ok {
				c.Default = v
			}

			if v, ok := v["description"].(string); ok {
				c.Description = v
			}

			if v, ok := v["launch"].(bool); ok {
				c.Launch = v
			}

			if v, ok := v["name"].(string); ok {
				c.Name = v
			}

			m.Configurations = append(m.Configurations, c)
		}
	}

	if v, ok := metadata["dependencies"]; ok {
		for _, v := range v.([]map[string]interface{}) {
			var d BuildModuleDependency

			if v, ok := v["id"].(string); ok {
				d.ID = v
			}

			if v, ok := v["name"].(string); ok {
				d.Name = v
			}

			if v, ok := v["version"].(string); ok {
				d.Version = v
			}

			if v, ok := v["uri"].(string); ok {
				d.URI = v
			}

			if v, ok := v["sha256"].(string); ok {
				d.SHA256 = v
			}

			if v, ok := v["stacks"].([]interface{}); ok {
				for _, v := range v {
					d.Stacks = append(d.Stacks, v.(string))
				}
			}

			if v, ok := v["licenses"].([]map[string]interface{}); ok {
				for _, v := range v {
					var l BuildModuleDependencyLicense

					if v, ok := v["type"].(string); ok {
						l.Type = v
					}

					if v, ok := v["uri"].(string); ok {
						l.URI = v
					}

					d.Licenses = append(d.Licenses, l)
				}
			}

			if v, ok := v["cpes"].([]interface{}); ok {
				for _, v := range v {
					d.CPEs = append(d.CPEs, v.(string))
				}
			}

			if v, ok := v["purl"].(string); ok {
				d.PURL = v
			}

			if v, ok := v["deprecation_date"].(string); ok {
				deprecationDate, err := time.Parse(time.RFC3339, v)

				if err != nil {
					return BuildModuleMetadata{}, fmt.Errorf("unable to parse deprecation date\n%w", err)
				}

				d.DeprecationDate = deprecationDate
			}

			m.Dependencies = append(m.Dependencies, d)
		}
	}

	if v, ok := metadata["include-files"].([]interface{}); ok {
		for _, v := range v {
			m.IncludeFiles = append(m.IncludeFiles, v.(string))
		}
	}

	if v, ok := metadata["pre-package"].(string); ok {
		m.PrePackage = v
	}

	return m, nil
}

// ConfigurationResolver provides functionality for resolving a configuration value.
type ConfigurationResolver struct {

	// Configurations are the configurations to resolve against
	Configurations []BuildModuleConfiguration
}

type configurationEntry struct {
	Name        string
	Description string
	Value       string
}

func (c configurationEntry) String(nameLength int, valueLength int) string {
	sb := strings.Builder{}

	sb.WriteString("$")
	sb.WriteString(c.Name)
	for i := 0; i < nameLength-len(c.Name); i++ {
		sb.WriteString(" ")
	}

	sb.WriteString("  ")
	sb.WriteString(c.Value)
	for i := 0; i < valueLength-len(c.Value); i++ {
		sb.WriteString(" ")
	}

	if valueLength > 0 {
		sb.WriteString("  ")
	}

	sb.WriteString(c.Description)

	return sb.String()
}

// NewConfigurationResolver creates a new instance from buildmodule metadata.  Logs configuration options to the body
// level int the form 'Set $Name to configure $Description[. Default <i>$Default</i>.]'.
func NewConfigurationResolver(md BuildModuleMetadata, logger *log.Logger) (ConfigurationResolver, error) {

	cr := ConfigurationResolver{Configurations: md.Configurations}

	if logger == nil {
		return cr, nil
	}

	var (
		build   []configurationEntry
		launch  []configurationEntry
		unknown []configurationEntry

		nameLength  int
		valueLength int
	)

	sort.Slice(md.Configurations, func(i, j int) bool {
		return md.Configurations[i].Name < md.Configurations[j].Name
	})

	for _, c := range md.Configurations {
		s, _ := cr.Resolve(c.Name)

		e := configurationEntry{
			Name:        c.Name,
			Description: c.Description,
			Value:       s,
		}

		if l := len(e.Name); l > nameLength {
			nameLength = l
		}

		if l := len(e.Value); l > valueLength {
			valueLength = l
		}

		if c.Build {
			build = append(build, e)
		}

		if c.Launch {
			launch = append(launch, e)
		}

		if !c.Build && !c.Launch {
			unknown = append(unknown, e)
		}
	}

	f := color.New(color.Faint)

	if len(build) > 0 {
		logger.Header(f.Sprint("Build Configuration:"))
		for _, e := range build {
			logger.Body(e.String(nameLength, valueLength))
		}
	}

	if len(launch) > 0 {
		logger.Header(f.Sprint("Launch Configuration:"))
		for _, e := range launch {
			logger.Body(e.String(nameLength, valueLength))
		}
	}

	if len(unknown) > 0 {
		logger.Header(f.Sprint("Unknown Configuration:"))
		for _, e := range unknown {
			logger.Body(e.String(nameLength, valueLength))
		}
	}

	return cr, nil
}

// Resolve resolves the value for a configuration option, returning the default value and false if it was not set.
func (c *ConfigurationResolver) Resolve(name string) (string, bool) {
	if v, ok := os.LookupEnv(name); ok {
		return v, ok
	}

	for _, c := range c.Configurations {
		if c.Name == name {
			return c.Default, false
		}
	}

	return "", false
}

// ResolveBool resolves a boolean value for a configuration option. Returns true for 1, t, T, TRUE, true, True. Returns
// false for all other values or unset.
func (c *ConfigurationResolver) ResolveBool(name string) bool {
	s, _ := c.Resolve(name)
	t, err := strconv.ParseBool(s)
	if err != nil {
		return false
	}

	return t
}

// DependencyResolver provides functionality for resolving a dependency given a collection of constraints.
type DependencyResolver struct {

	// Dependencies are the dependencies to resolve against.
	Dependencies []BuildModuleDependency

	// StackID is the stack id of the build.
	StackID string

	// Logger is the logger used to write to the console.
	Logger *log.Logger
}

// NewDependencyResolver creates a new instance from the build module metadata and stack id.
func NewDependencyResolver(md BuildModuleMetadata, stackId string) (DependencyResolver, error) {
	return DependencyResolver{Dependencies: md.Dependencies, StackID: stackId}, nil
}

// NoValidDependenciesError is returned when the resolver cannot find any valid dependencies given the constraints.
type NoValidDependenciesError struct {
	// Message is the error message
	Message string
}

func (n NoValidDependenciesError) Error() string {
	return n.Message
}

// IsNoValidDependencies indicates whether an error is a NoValidDependenciesError.
func IsNoValidDependencies(err error) bool {
	_, ok := err.(NoValidDependenciesError)
	return ok
}

// Resolve returns the latest version of a dependency within the collection of Dependencies.  The candidate set is first
// filtered by the constraints, then the remaining candidates are sorted for the latest result by semver semantics.
// Version can contain wildcards and defaults to "*" if not specified.
func (d *DependencyResolver) Resolve(id string, version string) (BuildModuleDependency, error) {
	if version == "" {
		version = "*"
	}

	vc, err := semver.NewConstraint(version)
	if err != nil {
		return BuildModuleDependency{}, fmt.Errorf("invalid constraint %s\n%w", vc, err)
	}

	var candidates []BuildModuleDependency
	for _, c := range d.Dependencies {
		v, err := semver.NewVersion(c.Version)
		if err != nil {
			return BuildModuleDependency{}, fmt.Errorf("unable to parse version %s\n%w", c.Version, err)
		}

		if c.ID == id && vc.Check(v) && d.contains(c.Stacks, d.StackID) {
			candidates = append(candidates, c)
		}
	}

	if len(candidates) == 0 {
		return BuildModuleDependency{}, NoValidDependenciesError{
			Message: fmt.Sprintf("no valid dependencies for %s, %s, and %s in %s",
				id, version, d.StackID, DependenciesFormatter(d.Dependencies)),
		}
	}

	sort.Slice(candidates, func(i int, j int) bool {
		a, _ := semver.NewVersion(candidates[i].Version)
		b, _ := semver.NewVersion(candidates[j].Version)

		return a.GreaterThan(b)
	})

	candidate := candidates[0]

	if (candidate.DeprecationDate != time.Time{}) {
		d.printDependencyDeprecation(candidate)
	}

	return candidate, nil
}

func (DependencyResolver) contains(candidates []string, value string) bool {
	if len(candidates) == 0 {
		return true
	}

	for _, c := range candidates {
		if c == value || c == "*" {
			return true
		}
	}

	return false
}

func (d *DependencyResolver) printDependencyDeprecation(dependency BuildModuleDependency) {
	if d.Logger == nil {
		return
	}

	f := color.New(color.FgYellow)

	if dependency.IsDeprecated() {
		d.Logger.Header(f.Sprint("Deprecation Notice:"))
		d.Logger.Body(f.Sprintf("Version %s of %s is deprecated.", dependency.Version, dependency.Name))
		d.Logger.Body(f.Sprintf("Migrate your application to a supported version of %s.", dependency.Name))
	} else if dependency.IsSoonDeprecated() {
		d.Logger.Header(f.Sprint("Deprecation Notice:"))
		d.Logger.Body(f.Sprintf("Version %s of %s will be deprecated after %s.", dependency.Version, dependency.Name, dependency.DeprecationDate.Format("2006-01-02")))
		d.Logger.Body(f.Sprintf("Migrate your application to a supported version of %s before this time.", dependency.Name))
	}
}
