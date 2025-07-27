/*
 * Copyright 2018-2025 the original author or authors.
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
	"bytes"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver/v3"
	"github.com/heroku/color"

	"github.com/paketo-buildpacks/libpak/v2/log"
)

// BuildModuleConfiguration represents a build or launch configuration parameter.
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

// DependencyLayerContributorMetadata returns the subset of data from BuildpackDependency that is use as expected metadata for the DependencyLayerContributor.
type DependencyLayerContributorMetadata struct {
	// ID is the dependency ID.
	ID string `toml:"id"`

	// Name is the dependency name.
	Name string `toml:"name"`

	// Version is the dependency version.
	Version string `toml:"version"`

	// Checksum is the hash of the dependency.
	Checksum string `toml:"checksum"`
}

// BuildModuleMetadata is an extension to libcnb.Buildpack / libcnb.Extension's metadata with opinions.
type BuildModuleMetadata struct {
	// Configurations are environment variables that can be used at build time to configure the buildpack and launch
	// time to configure the application.
	Configurations []BuildModuleConfiguration `toml:"configurations"`

	// Dependencies are the dependencies known to the buildpack.
	Dependencies []BuildModuleDependency `toml:"dependencies"`

	// IncludeFiles describes the files to include in the package.
	IncludeFiles []string `toml:"include-files"`

	// PrePackage describes a command to invoke before packaging.
	PrePackage string `toml:"pre-package"`
}

// NewBuildModuleMetadata creates a new instance of BuildpackMetadata from the contents of libcnb.Buildpack.Metadata
func NewBuildModuleMetadata(metadata map[string]interface{}) (BuildModuleMetadata, error) {
	m := BuildModuleMetadata{}

	buf := bytes.NewBuffer(nil)
	if err := toml.NewEncoder(buf).Encode(metadata); err != nil {
		return BuildModuleMetadata{}, fmt.Errorf("unable to encode metadata\n%w", err)
	}

	if _, err := toml.NewDecoder(buf).Decode(&m); err != nil {
		return BuildModuleMetadata{}, fmt.Errorf("unable to decode metadata\n%w", err)
	}

	for i := range m.Dependencies {
		if len(m.Dependencies[i].GetPURLS()) != 1 {
			return BuildModuleMetadata{}, fmt.Errorf("dependency %s has %d PURLs, expected 1", m.Dependencies[i].ID, len(m.Dependencies[i].GetPURLS()))
		}
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

// NewConfigurationResolver creates a new instance from buildmodule metadata.
func NewConfigurationResolver(md BuildModuleMetadata) (ConfigurationResolver, error) {
	cr := ConfigurationResolver{Configurations: md.Configurations}

	sort.Slice(md.Configurations, func(i, j int) bool {
		return md.Configurations[i].Name < md.Configurations[j].Name
	})

	return cr, nil
}

// LogConfiguration will write the configuration options to the body level in
// the form 'Set $Name to configure $Description[. Default <i>$Default</i>.]'.
func (c *ConfigurationResolver) LogConfiguration(logger log.Logger) {
	var (
		build   []configurationEntry
		launch  []configurationEntry
		unknown []configurationEntry

		nameLength  int
		valueLength int
	)

	for _, config := range c.Configurations {
		s, _ := c.Resolve(config.Name)

		e := configurationEntry{
			Name:        config.Name,
			Description: config.Description,
			Value:       s,
		}

		if l := len(e.Name); l > nameLength {
			nameLength = l
		}

		if l := len(e.Value); l > valueLength {
			valueLength = l
		}

		if config.Build {
			build = append(build, e)
		}

		if config.Launch {
			launch = append(launch, e)
		}

		if !config.Build && !config.Launch {
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
	Logger log.Logger
}

// NewDependencyResolver creates a new instance from the build module metadata and stack id.
func NewDependencyResolver(md BuildModuleMetadata, stackID string) (DependencyResolver, error) {
	return DependencyResolver{Dependencies: md.Dependencies, StackID: stackID}, nil
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

		// filter out deps that do not match the current running architecture
		arch, err := c.GetArch()
		if err != nil {
			return BuildModuleDependency{}, fmt.Errorf("unable to compare arch\n%w", err)
		}
		if c.Arch != "" && arch != archFromSystem() {
			continue
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

	if (candidate.GetEOLDate() != time.Time{}) {
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
		d.Logger.Body(f.Sprintf("Version %s of %s will be deprecated after %s.", dependency.Version, dependency.Name, dependency.GetEOLDate().Format("2006-01-02")))
		d.Logger.Body(f.Sprintf("Migrate your application to a supported version of %s before this time.", dependency.Name))
	}
}
