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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/heroku/color"

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
}

// NewLayerContributor creates a new instance.
func NewLayerContributor(name string, expectedMetadata interface{}) LayerContributor {
	return LayerContributor{
		ExpectedMetadata: expectedMetadata,
		Name:             name,
	}
}

// LayerFunc is a callback function that is invoked when a layer needs to be contributed.
type LayerFunc func() (libcnb.Layer, error)

// Contribute is the function to call when implementing your libcnb.LayerContributor.
func (l *LayerContributor) Contribute(layer libcnb.Layer, f LayerFunc) (libcnb.Layer, error) {
	raw := &bytes.Buffer{}
	if err := toml.NewEncoder(raw).Encode(l.ExpectedMetadata); err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to encode metadata\n%w", err)
	}

	expected := map[string]interface{}{}
	if _, err := toml.Decode(raw.String(), &expected); err != nil {
		return libcnb.Layer{}, fmt.Errorf("unable to decode metadata\n%w", err)
	}

	l.Logger.Debugf("Expected metadata: %+v", expected)
	l.Logger.Debugf("Actual metadata: %+v", layer.Metadata)

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

	layer, err := f()
	if err != nil {
		return libcnb.Layer{}, err
	}

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

	// LayerContributor is the contained LayerContributor used for the actual contribution.
	LayerContributor LayerContributor

	// Logger is the logger to use.
	Logger bard.Logger

	// RequestModifierFunc is an optional Request Modifier to use when downloading the dependency.
	RequestModifierFunc RequestModifierFunc
}

// NewDependencyLayerContributor creates a new instance and adds the dependency to the Buildpack Plan.
func NewDependencyLayerContributor(dependency BuildpackDependency, cache DependencyCache, plan *libcnb.BuildpackPlan) DependencyLayerContributor {
	c := DependencyLayerContributor{
		Dependency:       dependency,
		DependencyCache:  cache,
		LayerContributor: NewLayerContributor(fmt.Sprintf("%s %s", dependency.Name, dependency.Version), dependency),
	}

	entry := dependency.AsBuildpackPlanEntry()
	entry.Metadata["layer"] = c.LayerName()
	plan.Entries = append(plan.Entries, entry)

	return c
}

// DependencyLayerFunc is a callback function that is invoked when a dependency needs to be contributed.
type DependencyLayerFunc func(artifact *os.File) (libcnb.Layer, error)

// Contribute is the function to call whe implementing your libcnb.LayerContributor.
func (d *DependencyLayerContributor) Contribute(layer libcnb.Layer, f DependencyLayerFunc) (libcnb.Layer, error) {
	d.LayerContributor.Logger = d.Logger

	return d.LayerContributor.Contribute(layer, func() (libcnb.Layer, error) {
		artifact, err := d.DependencyCache.ArtifactWithRequestModification(d.Dependency, d.RequestModifierFunc)
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

// HelperLayerContributor is a helper for implementing a libcnb.LayerContributor for a buildpack helper application in
// order to get consistent logging and avoidance.
type HelperLayerContributor struct {

	// Path is the path to the helper application.
	Path string

	// LayerContributor is the contained LayerContributor used for the actual contribution.
	LayerContributor LayerContributor

	// Logger is the logger to use.
	Logger bard.Logger

	// Names are the names of the helpers to create
	Names []string
}

// NewHelperLayerContributor creates a new instance and adds the helper to the Buildpack Plan.
func NewHelperLayerContributor(buildpack libcnb.Buildpack, plan *libcnb.BuildpackPlan, names ...string) HelperLayerContributor {
	c := HelperLayerContributor{
		Path:             filepath.Join(buildpack.Path, "bin", "helper"),
		LayerContributor: NewLayerContributor("Launch Helper", buildpack.Info),
		Names:            names,
	}

	plan.Entries = append(plan.Entries, libcnb.BuildpackPlanEntry{
		Name: "helper",
		Metadata: map[string]interface{}{
			"layer":   c.Name(),
			"names":   names,
			"version": buildpack.Info.Version,
		},
	})

	return c
}

// Contribute is the function to call whe implementing your libcnb.LayerContributor.
func (h HelperLayerContributor) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	h.LayerContributor.Logger = h.Logger

	return h.LayerContributor.Contribute(layer, func() (libcnb.Layer, error) {
		in, err := os.Open(h.Path)
		if err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to open %s\n%w", h.Path, err)
		}
		defer in.Close()

		out := filepath.Join(layer.Path, "helper")
		if err := sherpa.CopyFile(in, out); err != nil {
			return libcnb.Layer{}, fmt.Errorf("unable to copy %s to %s", h.Path, out)
		}

		var p []string
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

			p = append(p,
				"exec 4<&1",
				fmt.Sprintf(`for_export=$(%s 3>&1 >&4) || exit $?`, link),
				"exec 4<&-",
				"set -a",
				`eval "$for_export"`,
				"set +a",
			)
		}

		layer.Profile.Add("helper", strings.Join(p, "\n"))

		layer.Launch = true
		return layer, nil
	})
}

// Name returns the conventional name of the layer for this contributor
func (h HelperLayerContributor) Name() string {
	return filepath.Base(h.Path)
}
