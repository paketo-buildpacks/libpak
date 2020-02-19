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
	"errors"
	"fmt"
	"reflect"
	"sort"

	"github.com/Masterminds/semver/v3"
	"github.com/buildpacks/libcnb"
	"github.com/mitchellh/mapstructure"
)

// BuildpackVersion is a semver complaint version.
type BuildpackVersion struct {
	*semver.Version
}

// NewBuildpackVersion creates a new instance of BuildpackVersion, panicing if the version is not semver compliant.
func NewBuildpackVersion(v string) BuildpackVersion {
	s, err := semver.NewVersion(v)
	if err != nil {
		panic(err)
	}

	return BuildpackVersion{s}
}

func (b BuildpackVersion) MarshalText() ([]byte, error) {
	return []byte(b.Original()), nil
}

func (b *BuildpackVersion) UnmarshalText(text []byte) error {
	var err error

	b.Version, err = semver.NewVersion(string(text))
	if err != nil && errors.Is(err, semver.ErrInvalidSemVer) {
		b.Version, _ = semver.NewVersion("0.0.0")
	} else if err != nil {
		return fmt.Errorf("unable to parse version %s: %w", string(text), err)
	}

	return nil
}

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
	Version BuildpackVersion `toml:"version"`

	// URI is the dependency URI.
	URI string `toml:"uri"`

	// SHA256 is the hash of the dependency.
	SHA256 string `toml:"sha256"`

	// Stacks are the stacks the dependency is compatible with.
	Stacks []string `toml:"stacks"`

	// Licenses are the stacks the dependency is distributed under.
	Licenses []BuildpackDependencyLicense `toml:"licenses"`
}

// PlanEntry returns the dependency as a BuildpackPlanEntry suitable for addition to a BuildpackPlan.
func (b BuildpackDependency) PlanEntry() libcnb.BuildpackPlanEntry {
	return libcnb.BuildpackPlanEntry{
		Name:    b.ID,
		Version: b.Version.String(),
		Metadata: map[string]interface{}{
			"name":     b.Name,
			"uri":      b.URI,
			"sha256":   b.SHA256,
			"stacks":   b.Stacks,
			"licenses": b.Licenses,
		},
	}
}

// Metadata returns a simplified representation of the dependency for use in metadata comparison.
func (b BuildpackDependency) Metadata() map[string]interface{} {
	stacks := make([]interface{}, len(b.Stacks))
	for i, s := range b.Stacks {
		stacks[i] = s
	}

	licenses := make([]map[string]interface{}, len(b.Licenses))
	for i, l := range b.Licenses {
		licenses[i] = map[string]interface{}{
			"type": l.Type,
			"uri":  l.URI,
		}
	}

	m := map[string]interface{}{
		"id":       b.ID,
		"name":     b.Name,
		"version":  b.Version.String(),
		"uri":      b.URI,
		"sha256":   b.SHA256,
		"stacks":   stacks,
		"licenses": licenses,
	}

	return m
}

// BuildpackMetadata is an extension to libcnb.Buildpack's metadata with opinions.
type BuildpackMetadata struct {

	// DefaultVersions represent the default versions for dependencies keyed by Dependency.Id.
	DefaultVersions map[string]string `mapstructure:"default-versions"`

	// Dependencies are the dependencies known to the buildpack.
	Dependencies []BuildpackDependency `mapstructure:"dependencies"`

	// IncludeFiles describes the files to include in the package.
	IncludeFiles []string `mapstructure:"include_files"`

	// PrePackage describes a command to invoke before packaging.
	PrePackage string `mapstructure:"pre_package"`
}

// NewBuildpackMetadata creates a new instance of BuildpackMetadata from the contents of libcnb.Buildpack.Metadata
func NewBuildpackMetadata(metadata map[string]interface{}) (BuildpackMetadata, error) {
	var m BuildpackMetadata

	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
			if f.Kind() != reflect.String {
				return data, nil
			}

			if t != reflect.TypeOf(BuildpackVersion{}) {
				return data, nil
			}

			v := BuildpackVersion{}
			if err := v.UnmarshalText([]byte(data.(string))); err != nil {
				return nil, err
			}

			return v, nil
		},
		Result: &m,
	})
	if err != nil {
		return BuildpackMetadata{}, fmt.Errorf("unable to create decoder")
	}

	if err := d.Decode(metadata); err != nil {
		return BuildpackMetadata{}, fmt.Errorf("unable to unmarshal metadata %+v: %w", metadata, err)
	}

	if m.DefaultVersions == nil {
		m.DefaultVersions = map[string]string{}
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
		if c.ID == constraint.ID && vc.Check(c.Version.Version) && d.contains(c.Stacks, constraint.StackID) {
			candidates = append(candidates, c)
		}
	}

	if len(candidates) == 0 {
		return BuildpackDependency{}, fmt.Errorf("no valid dependencies for %s, %s, and %s in %s",
			constraint.ID, constraint.Version, constraint.StackID, DependenciesFormatter(d.Dependencies))
	}

	sort.Slice(candidates, func(i int, j int) bool {
		return candidates[i].Version.GreaterThan(candidates[j].Version.Version)
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
