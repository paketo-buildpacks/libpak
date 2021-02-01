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
	"path/filepath"
	"reflect"

	"github.com/heroku/color"
	"github.com/pelletier/go-toml"

	"github.com/buildpacks/libcnb"

	"github.com/paketo-buildpacks/libpak/sherpa"

	"github.com/paketo-buildpacks/libpak/bard"
)

// LayerContributor is a helper for implementing a libcnb.LayerContributor in order to get consistent logging and
// avoidance.
type LayerContributor struct {

	// ExpectedMetadata is the metadata to compare against any existing layer metadata.
	ExpectedMetadata interface{}

	// Logger is the logger to use.
	Logger bard.Logger

	// Name is the user readable name of the contribution.
	Name string

	// ExpectedTypes indicates the types that should be set on the layer.
	ExpectedTypes libcnb.LayerTypes
}

// NewLayerContributor creates a new instance.
func NewLayerContributor(name string, expectedMetadata interface{}, expectedTypes libcnb.LayerTypes) LayerContributor {
	return LayerContributor{
		ExpectedMetadata: expectedMetadata,
		Name:             name,
		ExpectedTypes:    expectedTypes,
	}
}

// LayerFunc is a callback function that is invoked when a layer needs to be contributed.
type LayerFunc func() (libcnb.Layer, error)


// Contribute is the function to call when implementing your libcnb.LayerContributor.
func (l *LayerContributor) Contribute(layer libcnb.Layer, f LayerFunc) (libcnb.Layer, error) {
	raw, err := toml.Marshal(l.ExpectedMetadata)
	if err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to encode metadata\n%w", err)
	}

	expected := map[string]interface{}{}
	if err := toml.Unmarshal(raw, &expected); err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to decode metadata\n%w", err)
	}

	l.Logger.Debugf("Expected metadata: %+v", expected)
	l.Logger.Debugf("Actual metadata: %+v", layer.Metadata)

	// TODO: compare entire layer not just metadata (in case build, launch, or cache have changed)
	if reflect.DeepEqual(expected, layer.Metadata) {
		l.Logger.Headerf("%s: %s cached layer", color.BlueString(l.Name), color.GreenString("Reusing"))
		return layer, nil
	}

	l.Logger.Headerf("%s: %s to layer", color.BlueString(l.Name), color.YellowString("Contributing"))

	if err := os.RemoveAll(layer.Path); err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to remove existing layer directory %s\n%w", layer.Path, err)
	}

	if err := os.MkdirAll(layer.Path, 0755); err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to create layer directory %s\n%w", layer.Path, err)
	}

	layer, err = f()
	if err != nil {
		return libcnb.Layer{}, err
	}

	layer.LayerTypes = l.ExpectedTypes
	layer.Metadata = expected

	return layer, nil
}

// DependencyLayerContributor is a helper for implementing a libcnb.LayerContributor for a BuildpackDependency in order
// to get consistent logging and avoidance.
type DependencyLayerContributor struct {

	// Dependency is the dependency being contributed.
	Dependency BuildpackDependency

	// DependencyCache is the cache to use to get the dependency.
	DependencyCache DependencyCache

	// ExpectedTypes indicates the types that should be set on the layer.
	ExpectedTypes libcnb.LayerTypes

	// ExpectedMetadata contains metadata describing the expected layer
	ExpectedMetadata interface{}

	// Logger is the logger to use.
	Logger bard.Logger

	// RequestModifierFuncs is an optional Request Modifier to use when downloading the dependency.
	RequestModifierFuncs []RequestModifierFunc
}

// NewDependencyLayer returns a new DependencyLayerContributor for the given BuildpackDependency and a BOMEntry describing the layer contents.
func NewDependencyLayer(dependency BuildpackDependency, cache DependencyCache, types libcnb.LayerTypes) (DependencyLayerContributor, libcnb.BOMEntry) {
	c := DependencyLayerContributor{
		Dependency:       dependency,
		ExpectedMetadata: dependency,
		DependencyCache:  cache,
		ExpectedTypes:    types,
	}

	entry := dependency.AsBOMEntry()
	entry.Metadata["layer"] = c.LayerName()

	if types.Launch {
		entry.Launch = true
	}
	if !(types.Launch && !types.Cache && !types.Build) {
		// launch-only layers are the only layers NOT guaranteed to be present in the build environment
		entry.Build = true
	}

	return c, entry
}

// DependencyLayerFunc is a callback function that is invoked when a dependency needs to be contributed.
type DependencyLayerFunc func(artifact *os.File) (libcnb.Layer, error)

// Contribute is the function to call whe implementing your libcnb.LayerContributor.
func (d *DependencyLayerContributor) Contribute(layer libcnb.Layer, f DependencyLayerFunc) (libcnb.Layer, error) {
	lc := NewLayerContributor(d.Name(), d.ExpectedMetadata, d.ExpectedTypes)
	lc.Logger = d.Logger

	return lc.Contribute(layer, func() (libcnb.Layer, error) {
		artifact, err := d.DependencyCache.Artifact(d.Dependency, d.RequestModifierFuncs...)
		if err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to get dependency %s\n%w", d.Dependency.ID, err)
		}
		defer artifact.Close()

		return f(artifact)
	})
}

// LayerName returns the conventional name of the layer for this contributor
func (d *DependencyLayerContributor) LayerName() string {
	return d.Dependency.ID
}

// Name returns the human readable name of the layer
func (d *DependencyLayerContributor) Name() string {
	return fmt.Sprintf("%s %s", d.Dependency.Name, d.Dependency.Version)
}

// HelperLayerContributor is a helper for implementing a libcnb.LayerContributor for a buildpack helper application in
// order to get consistent logging and avoidance.
type HelperLayerContributor struct {

	// Path is the path to the helper application.
	Path string

	// BuildpackInfo describes the buildpack that provides the helper
	BuildpackInfo libcnb.BuildpackInfo

	// Logger is the logger to use.
	Logger bard.Logger

	// Names are the names of the helpers to create
	Names []string
}

// NewHelperLayer returns a new HelperLayerContributor and a BOMEntry describing the layer contents.
func NewHelperLayer(buildpack libcnb.Buildpack, names ...string) (HelperLayerContributor, libcnb.BOMEntry) {
	c := HelperLayerContributor{
		Path:          filepath.Join(buildpack.Path, "bin", "helper"),
		Names:         names,
		BuildpackInfo: buildpack.Info,
	}

	entry := libcnb.BOMEntry{
		Name: "helper",
		Metadata: map[string]interface{}{
			"layer":   c.Name(),
			"names":   names,
			"version": buildpack.Info.Version,
		},
		Launch: true,
	}

	return c, entry
}

// Name returns the conventional name of the layer for this contributor
func (h HelperLayerContributor) Name() string {
	return filepath.Base(h.Path)
}

// Contribute is the function to call whe implementing your libcnb.LayerContributor.
func (h HelperLayerContributor) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	lc := NewLayerContributor("Launch Helper", h.BuildpackInfo, libcnb.LayerTypes{
		Launch: true,
	})
	lc.Logger = h.Logger

	return lc.Contribute(layer, func() (libcnb.Layer, error) {
		in, err := os.Open(h.Path)
		if err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to open %s\n%w", h.Path, err)
		}
		defer in.Close()

		out := filepath.Join(layer.Path, "helper")
		if err := sherpa.CopyFile(in, out); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to copy %s to %s", h.Path, out)
		}

		for _, n := range h.Names {
			link := layer.Exec.FilePath(n)
			h.Logger.Bodyf("Creating %s", link)

			f := filepath.Dir(link)
			if err := os.MkdirAll(f, 0755); err != nil {
				return libcnb.Layer{}, fmt.Errorf("unable to create %s\n%w", f, err)
			}

			if err := os.Symlink(out, link); err != nil {
				return libcnb.Layer{}, fmt.Errorf("unable to link %s to %s\n%w", out, link, err)
			}
		}

		return layer, nil
	})
}
