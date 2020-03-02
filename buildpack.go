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
	"sort"

	"github.com/Masterminds/semver/v3"
)

// License represents a license that a BuildpackDependency is distributed under.  At least one of Name or URI MUST be
// specified.
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

// BuildpackMetadata is an extension to libcnb.Buildpack's metadata with opinions.
type BuildpackMetadata struct {

	// DefaultVersions represent the default versions for dependencies keyed by Dependency.Id.
	DefaultVersions map[string]string

	// Dependencies are the dependencies known to the buildpack.
	Dependencies []BuildpackDependency

	// IncludeFiles describes the files to include in the package.
	IncludeFiles []string

	// PrePackage describes a command to invoke before packaging.
	PrePackage string
}

// NewBuildpackMetadata creates a new instance of BuildpackMetadata from the contents of libcnb.Buildpack.Metadata
func NewBuildpackMetadata(metadata map[string]interface{}) (BuildpackMetadata, error) {
	m := BuildpackMetadata{
		DefaultVersions: map[string]string{},
	}

	if v, ok := metadata["default_versions"].(map[string]interface{}); ok {
		for k, v := range v {
			m.DefaultVersions[k] = v.(string)
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

	if v, ok := metadata["include_files"].([]interface{}); ok {
		for _, v := range v {
			m.IncludeFiles = append(m.IncludeFiles, v.(string))
		}
	}

	if v, ok := metadata["pre_package"].(string); ok {
		m.PrePackage = v
	}

	return m, nil
}

// DependencyResolver provides functionality for resolving a dependency fiven a collection of constraints.
type DependencyResolver struct {

	// Dependencies are the dependencies to resolve against.
	Dependencies []BuildpackDependency
}

// DependencyConstraint is the collection of constraints to use during resolution.
type DependencyConstraint struct {

	// ID is the id of dependency.
	ID string

	// Version is the version constraint of the dependency. Value can contain wildcards and defaults to "*" if not
	// specified.
	Version string

	// StackID is the stack ID of the dependency.
	StackID string
}

// Resolve returns the latest version of a dependency within the collection of Dependencies.  The candidate set is first
// filtered by the constraints, then the remaining candidates are sorted for the latest result by semver semantics.
// Version can contain wildcards and defaults to "*" if not specified.
func (d *DependencyResolver) Resolve(constraint DependencyConstraint) (BuildpackDependency, error) {
	if constraint.Version == "" {
		constraint.Version = "*"
	}

	vc, err := semver.NewConstraint(constraint.Version)
	if err != nil {
		return BuildpackDependency{}, fmt.Errorf("invalid constraint %s: %w", vc, err)
	}

	var candidates []BuildpackDependency
	for _, c := range d.Dependencies {
		v, err := semver.NewVersion(c.Version)
		if err != nil {
			return BuildpackDependency{}, fmt.Errorf("unable to parse version %s: %w", c.Version, err)
		}

		if c.ID == constraint.ID && vc.Check(v) && d.contains(c.Stacks, constraint.StackID) {
			candidates = append(candidates, c)
		}
	}

	if len(candidates) == 0 {
		return BuildpackDependency{}, fmt.Errorf("no valid dependencies for %s, %s, and %s in %s",
			constraint.ID, constraint.Version, constraint.StackID, DependenciesFormatter(d.Dependencies))
	}

	sort.Slice(candidates, func(i int, j int) bool {
		a, _ := semver.NewVersion(candidates[i].Version)
		b, _ := semver.NewVersion(candidates[j].Version)

		return a.GreaterThan(b)
	})

	return candidates[0], nil
}

// Any indicates whether the collection of dependencies has any dependency that satisfies the constraints.  This is
// used primarily to determine whether an optional dependency exists, before calling Resolve() which would throw an
// error if one did not.
func (d *DependencyResolver) Any(constraint DependencyConstraint) bool {
	_, err := d.Resolve(constraint)
	return err == nil
}

func (DependencyResolver) contains(candidates []string, value string) bool {
	for _, c := range candidates {
		if c == value {
			return true
		}
	}

	return false
}
