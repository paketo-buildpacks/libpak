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
	"fmt"
	"os"
	"sort"

	"github.com/Masterminds/semver/v3"
	"github.com/buildpacks/libcnb"
	"github.com/heroku/color"
	"github.com/paketo-buildpacks/libpak/bard"
)

// BuildpackConfiguration represents a build or launch configuration parameter.
type BuildpackConfiguration struct {

	// Default is the default value of the configuration parameter.  Optional.
	Default string `toml:"default"`

	// Description is the description of the configuration parameter.
	Description string `toml:"description"`

	// Name is the environment variable name of the configuration parameter.
	Name string `toml:"name"`
}

// BuildpackDependencyLicense represents a license that a BuildpackDependency is distributed under.  At least one of
// Name or URI MUST be specified.
type BuildpackDependencyLicense struct {

	// Type is the type of the license.  This is typically the SPDX short identifier.
	Type string `toml:"type"`

	// URI is the location where the license can be found.
	URI string `toml:"uri"`
}

// BuildpackDependency describes a dependency known to the buildpack.
type BuildpackDependency struct {
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

	// Licenses are the stacks the dependency is distributed under.
	Licenses []BuildpackDependencyLicense `toml:"licenses"`
}

// AsBuildpackPlanEntry renders the dependency as a BuildpackPlanEntry.
func (b BuildpackDependency) AsBuildpackPlanEntry() libcnb.BuildpackPlanEntry {
	return libcnb.BuildpackPlanEntry{
		Name:    b.ID,
		Version: b.Version,
		Metadata: map[string]interface{}{
			"name":     b.Name,
			"uri":      b.URI,
			"sha256":   b.SHA256,
			"stacks":   b.Stacks,
			"licenses": b.Licenses,
		},
	}
}

// BuildpackMetadata is an extension to libcnb.Buildpack's metadata with opinions.
type BuildpackMetadata struct {

	// Configurations are environment variables that can be used at build time to configure the buildpack and launch
	// time to configure the application.
	Configurations []BuildpackConfiguration

	// Dependencies are the dependencies known to the buildpack.
	Dependencies []BuildpackDependency

	// IncludeFiles describes the files to include in the package.
	IncludeFiles []string

	// PrePackage describes a command to invoke before packaging.
	PrePackage string
}

// NewBuildpackMetadata creates a new instance of BuildpackMetadata from the contents of libcnb.Buildpack.Metadata
func NewBuildpackMetadata(metadata map[string]interface{}) (BuildpackMetadata, error) {
	m := BuildpackMetadata{}

	if v, ok := metadata["configurations"]; ok {
		for _, v := range v.([]map[string]interface{}) {
			var c BuildpackConfiguration

			if v, ok := v["default"].(string); ok {
				c.Default = v
			}

			if v, ok := v["description"].(string); ok {
				c.Description = v
			}

			if v, ok := v["name"].(string); ok {
				c.Name = v
			}

			m.Configurations = append(m.Configurations, c)
		}
	}

	if v, ok := metadata["dependencies"]; ok {
		for _, v := range v.([]map[string]interface{}) {
			var d BuildpackDependency

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
					var l BuildpackDependencyLicense

					if v, ok := v["type"].(string); ok {
						l.Type = v
					}

					if v, ok := v["uri"].(string); ok {
						l.URI = v
					}

					d.Licenses = append(d.Licenses, l)
				}
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
	Configurations []BuildpackConfiguration
}

// NewConfigurationResolver creates a new instance from buildpack metadata.  Logs configuration options to the body
// level int the form 'Set $Name to configure $Description[. Default <i>$Default</i>.]'.
func NewConfigurationResolver(buildpack libcnb.Buildpack, logger *bard.Logger) (ConfigurationResolver, error) {
	md, err := NewBuildpackMetadata(buildpack.Metadata)
	if err != nil {
		return ConfigurationResolver{}, fmt.Errorf("unable to unmarshal buildpack metadata\n%w", err)
	}

	if logger != nil {
		sort.Slice(md.Configurations, func(i, j int) bool {
			return md.Configurations[i].Name < md.Configurations[j].Name
		})

		for _, c := range md.Configurations {
			s := fmt.Sprintf("Set $%s to configure %s", c.Name, c.Description)

			if c.Default != "" {
				s += fmt.Sprintf(". Default %s.", color.New(color.Italic).Sprint(c.Default))
			}

			logger.Body(s)
		}
	}

	return ConfigurationResolver{Configurations: md.Configurations}, nil
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

// DependencyResolver provides functionality for resolving a dependency given a collection of constraints.
type DependencyResolver struct {

	// Dependencies are the dependencies to resolve against.
	Dependencies []BuildpackDependency

	// StackID is the stack id of the build.
	StackID string
}

// NewDependencyResolver creates a new instance from the buildpack metadata and stack id.
func NewDependencyResolver(context libcnb.BuildContext) (DependencyResolver, error) {
	md, err := NewBuildpackMetadata(context.Buildpack.Metadata)
	if err != nil {
		return DependencyResolver{}, fmt.Errorf("unable to unmarshal buildpack metadata\n%w", err)
	}

	return DependencyResolver{Dependencies: md.Dependencies, StackID: context.StackID}, nil
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
func (d *DependencyResolver) Resolve(id string, version string) (BuildpackDependency, error) {
	if version == "" {
		version = "*"
	}

	vc, err := semver.NewConstraint(version)
	if err != nil {
		return BuildpackDependency{}, fmt.Errorf("invalid constraint %s\n%w", vc, err)
	}

	var candidates []BuildpackDependency
	for _, c := range d.Dependencies {
		v, err := semver.NewVersion(c.Version)
		if err != nil {
			return BuildpackDependency{}, fmt.Errorf("unable to parse version %s\n%w", c.Version, err)
		}

		if c.ID == id && vc.Check(v) && d.contains(c.Stacks, d.StackID) {
			candidates = append(candidates, c)
		}
	}

	if len(candidates) == 0 {
		return BuildpackDependency{}, NoValidDependenciesError{
			Message: fmt.Sprintf("no valid dependencies for %s, %s, and %s in %s",
				id, version, d.StackID, DependenciesFormatter(d.Dependencies)),
		}
	}

	sort.Slice(candidates, func(i int, j int) bool {
		a, _ := semver.NewVersion(candidates[i].Version)
		b, _ := semver.NewVersion(candidates[j].Version)

		return a.GreaterThan(b)
	})

	return candidates[0], nil
}

func (DependencyResolver) contains(candidates []string, value string) bool {
	for _, c := range candidates {
		if c == value {
			return true
		}
	}

	return false
}
