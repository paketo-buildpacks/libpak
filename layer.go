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
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/heroku/color"

	"github.com/buildpacks/libcnb/v2"

	"github.com/paketo-buildpacks/libpak/v2/log"
	"github.com/paketo-buildpacks/libpak/v2/sbom"
	"github.com/paketo-buildpacks/libpak/v2/sherpa"
	"github.com/paketo-buildpacks/libpak/v2/utils"
)

// ContributeLayersFunc takes a context and result pointer returning a list of Contributables, the list of Contributables will be turned into layers automatically
type ContributeLayersFunc func(context libcnb.BuildContext, result *libcnb.BuildResult) ([]Contributable, error)

// ContributableBuildFunc is a standard libcnb.BuildFunc implementation that delegates to a list of Contributables
func ContributableBuildFunc(contributeLayersFunc ContributeLayersFunc) libcnb.BuildFunc {
	return func(context libcnb.BuildContext) (libcnb.BuildResult, error) {
		result := libcnb.NewBuildResult()

		layerContributors, err := contributeLayersFunc(context, &result)
		if err != nil {
			return libcnb.BuildResult{}, fmt.Errorf("unable to fetch layer contributors\n%w", err)
		}

		for _, creator := range layerContributors {
			name := creator.Name()
			layer, err := context.Layers.Layer(name)
			if err != nil {
				return libcnb.BuildResult{}, fmt.Errorf("unable to create layer %s\n%w", name, err)
			}

			err = creator.Contribute(&layer)
			if err != nil {
				return libcnb.BuildResult{}, fmt.Errorf("unable to invoke layer creator\n%w", err)
			}

			result.Layers = append(result.Layers, layer)
		}

		return result, nil
	}
}

// Contributable is an interface that is implemented when you want to contribute new layers
type Contributable interface {
	// Contribute accepts a new empty layer and transforms it
	Contribute(layer *libcnb.Layer) error

	// Name is the name of the layer.
	Name() string
}

// LayerContributor is a helper for implementing Contributable in order to get consistent logging and avoidance.
type LayerContributor struct {
	// ExpectedMetadata is the metadata to compare against any existing layer metadata.
	ExpectedMetadata interface{}

	// Logger is the logger to use.
	Logger log.Logger

	// Name is the user readable name of the contribution.
	Name string

	// ExpectedTypes indicates the types that should be set on the layer.
	ExpectedTypes libcnb.LayerTypes
}

// NewLayerContributor creates a new instance.
func NewLayerContributor(name string, expectedMetadata interface{}, expectedTypes libcnb.LayerTypes, logger log.Logger) LayerContributor {
	return LayerContributor{
		ExpectedMetadata: expectedMetadata,
		ExpectedTypes:    expectedTypes,
		Logger:           logger,
		Name:             name,
	}
}

// LayerFunc is a callback function that is invoked when a layer needs to be contributed.
type LayerFunc func(*libcnb.Layer) error

// Contribute is the function to call when implementing Contributable.
func (l *LayerContributor) Contribute(layer *libcnb.Layer, f LayerFunc) error {
	layerRestored, err := l.checkIfLayerRestored(*layer)
	if err != nil {
		return fmt.Errorf("unable to check metadata\n%w", err)
	}

	expected, cached, err := l.checkIfMetadataMatches(*layer)
	if err != nil {
		return fmt.Errorf("unable to check metadata\n%w", err)
	}

	if cached && layerRestored {
		l.Logger.Headerf("%s: %s cached layer", color.BlueString(l.Name), color.GreenString("Reusing"))
		layer.LayerTypes = l.ExpectedTypes
		return nil
	}

	if !layerRestored {
		l.Logger.Headerf("%s: %s cached layer", color.BlueString(l.Name), color.RedString("Reloading"))
	} else {
		l.Logger.Headerf("%s: %s to layer", color.BlueString(l.Name), color.YellowString("Contributing"))
	}

	err = l.reset(layer)
	if err != nil {
		return fmt.Errorf("unable to reset\n%w", err)
	}

	err = f(layer)
	if err != nil {
		return err
	}

	layer.LayerTypes = l.ExpectedTypes
	layer.Metadata = expected

	return nil
}

func (l *LayerContributor) checkIfMetadataMatches(layer libcnb.Layer) (map[string]interface{}, bool, error) {
	raw, err := utils.Marshal(l.ExpectedMetadata)
	if err != nil {
		return map[string]interface{}{}, false, fmt.Errorf("unable to encode metadata\n%w", err)
	}

	expected := map[string]interface{}{}
	if err := toml.Unmarshal(raw, &expected); err != nil {
		return map[string]interface{}{}, false, fmt.Errorf("unable to decode metadata\n%w", err)
	}

	l.Logger.Debugf("Expected metadata: %+v", expected)
	l.Logger.Debugf("Actual metadata: %+v", layer.Metadata)

	match, err := l.Equals(expected, layer.Metadata)
	if err != nil {
		return map[string]interface{}{}, false, fmt.Errorf("unable to compare metadata\n%w", err)
	}
	return expected, match, nil
}

func (l *LayerContributor) Equals(expectedM map[string]interface{}, layerM map[string]interface{}) (bool, error) {
	if dep, ok := expectedM["dependency"].(map[string]interface{}); ok {
		for k, v := range dep {
			if k == "deprecation_date" {
				deprecationDate := v.(time.Time).Truncate(time.Second).In(time.UTC)
				dep["deprecation_date"] = deprecationDate
				break
			}
		}
	}
	if dep, ok := layerM["dependency"].(map[string]interface{}); ok {
		for k, v := range dep {
			if k == "deprecation_date" {
				deprecationDate, err := time.Parse(time.RFC3339, v.(string))
				if err != nil {
					return false, fmt.Errorf("unable to parse deprecation_date %s", v.(string))
				}
				deprecationDate = deprecationDate.Truncate(time.Second).In(time.UTC)
				dep["deprecation_date"] = deprecationDate
				break
			}
		}
	}
	return reflect.DeepEqual(expectedM, layerM), nil
}

func (l *LayerContributor) checkIfLayerRestored(layer libcnb.Layer) (bool, error) {
	layerTOML := fmt.Sprintf("%s.toml", layer.Path)
	tomlExists, err := sherpa.FileExists(layerTOML)
	if err != nil {
		return false, fmt.Errorf("unable to check if layer toml exists %s\n%w", layerTOML, err)
	}

	layerDirExists, err := sherpa.DirExists(layer.Path)
	if err != nil {
		return false, fmt.Errorf("unable to check if layer directory exists %s\n%w", layer.Path, err)
	}

	var dirContents []fs.DirEntry
	if layerDirExists {
		dirContents, err = os.ReadDir(layer.Path)
		if err != nil {
			return false, fmt.Errorf("unable to read directory %s\n%w", layer.Path, err)
		}
	}

	l.Logger.Debugf("Check If Layer Restored -> tomlExists: %s, layerDirExists: %s, dirContents: %s, cache: %s, build: %s",
		tomlExists, layerDirExists, dirContents, l.ExpectedTypes.Cache, l.ExpectedTypes.Build)
	return !(tomlExists && (!layerDirExists || len(dirContents) == 0) && (l.ExpectedTypes.Cache || l.ExpectedTypes.Build)), nil
}

func (l *LayerContributor) reset(layer *libcnb.Layer) error {
	if err := os.RemoveAll(layer.Path); err != nil {
		return fmt.Errorf("unable to remove existing layer directory %s\n%w", layer.Path, err)
	}

	if err := os.MkdirAll(layer.Path, 0755); err != nil {
		return fmt.Errorf("unable to create layer directory %s\n%w", layer.Path, err)
	}

	return nil
}

// DependencyLayerContributor is a helper for implementing a Contributable for a BuildpackDependency in order
// to get consistent logging and avoidance.
type DependencyLayerContributor struct {
	// Dependency is the dependency being contributed.
	Dependency BuildModuleDependency

	// DependencyCache is the cache to use to get the dependency.
	DependencyCache DependencyCache

	// ExpectedTypes indicates the types that should be set on the layer.
	ExpectedTypes libcnb.LayerTypes

	// ExpectedMetadata contains metadata describing the expected layer
	ExpectedMetadata interface{}

	// Logger is the logger to use.
	Logger log.Logger

	// RequestModifierFuncs is an optional Request Modifier to use when downloading the dependency.
	RequestModifierFuncs []RequestModifierFunc
}

// NewDependencyLayerContributor returns a new DependencyLayerContributor for the given BuildpackDependency
func NewDependencyLayerContributor(dependency BuildModuleDependency, cache DependencyCache, types libcnb.LayerTypes, logger log.Logger) DependencyLayerContributor {
	return DependencyLayerContributor{
		Dependency:       dependency,
		DependencyCache:  cache,
		ExpectedMetadata: dependency.GetMetadata(),
		ExpectedTypes:    types,
		Logger:           logger,
	}
}

// DependencyLayerFunc is a callback function that is invoked when a dependency needs to be contributed.
type DependencyLayerFunc func(layer *libcnb.Layer, artifact *os.File) error

// Contribute is the function to call whe implementing your Contributable.
func (d *DependencyLayerContributor) Contribute(layer *libcnb.Layer, f DependencyLayerFunc) error {
	lc := NewLayerContributor(d.Name(), d.ExpectedMetadata, d.ExpectedTypes, d.Logger)

	return lc.Contribute(layer, func(_ *libcnb.Layer) error {
		artifact, err := d.DependencyCache.Artifact(d.Dependency, d.RequestModifierFuncs...)
		if err != nil {
			return fmt.Errorf("unable to get dependency %s\n%w", d.Dependency.ID, err)
		}
		defer artifact.Close()

		// Only buildpacks can create layers, so source must be buildpack.toml
		sbomArtifact, err := d.Dependency.AsSyftArtifact("buildpack.toml")
		if err != nil {
			return fmt.Errorf("unable to get SBOM artifact %s\n%w", d.Dependency.ID, err)
		}

		sbomPath := layer.SBOMPath(libcnb.SyftJSON)
		dep := sbom.NewSyftDependency(layer.Path, []sbom.SyftArtifact{sbomArtifact})
		d.Logger.Debugf("Writing Syft SBOM at %s: %+v", sbomPath, dep)
		if err := dep.WriteTo(sbomPath); err != nil {
			return fmt.Errorf("unable to write SBOM\n%w", err)
		}

		return f(layer, artifact)
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

// HelperLayerContributor is a helper for implementing a Contributable for a buildpack helper application in
// order to get consistent logging and avoidance.
type HelperLayerContributor struct {

	// Path is the path to the helper application.
	Path string

	// BuildpackInfo describes the buildpack that provides the helper
	BuildpackInfo libcnb.BuildpackInfo

	// Logger is the logger to use.
	Logger log.Logger

	// Names are the names of the helpers to create
	Names []string
}

// NewHelperLayerContributor returns a new HelperLayerContributor
func NewHelperLayerContributor(buildpack libcnb.Buildpack, logger log.Logger, names ...string) HelperLayerContributor {
	return HelperLayerContributor{
		BuildpackInfo: buildpack.Info,
		Logger:        logger,
		Names:         names,
		Path:          filepath.Join(buildpack.Path, "bin", "helper"),
	}
}

// Name returns the conventional name of the layer for this contributor
func (h HelperLayerContributor) Name() string {
	return filepath.Base(h.Path)
}

// Contribute is the function to call whe implementing your Contributable.
func (h HelperLayerContributor) Contribute(layer *libcnb.Layer) error {
	expected := map[string]interface{}{"buildpackInfo": h.BuildpackInfo, "helperNames": h.Names}
	lc := NewLayerContributor("Launch Helper", expected, libcnb.LayerTypes{
		Launch: true,
	}, h.Logger)

	return lc.Contribute(layer, func(_ *libcnb.Layer) error {
		in, err := os.Open(h.Path)
		if err != nil {
			return fmt.Errorf("unable to open %s\n%w", h.Path, err)
		}
		defer in.Close()

		out := filepath.Join(layer.Path, "helper")
		if err := sherpa.CopyFile(in, out); err != nil {
			return fmt.Errorf("unable to copy %s to %s", h.Path, out)
		}

		for _, n := range h.Names {
			link := layer.Exec.FilePath(n)
			h.Logger.Bodyf("Creating %s", link)

			f := filepath.Dir(link)
			if err := os.MkdirAll(f, 0755); err != nil {
				return fmt.Errorf("unable to create %s\n%w", f, err)
			}

			if err := os.Symlink(out, link); err != nil {
				return fmt.Errorf("unable to link %s to %s\n%w", out, link, err)
			}
		}

		sbomArtifact, err := h.AsSyftArtifact()
		if err != nil {
			return fmt.Errorf("unable to get SBOM artifact for helper\n%w", err)
		}

		sbomPath := layer.SBOMPath(libcnb.SyftJSON)
		dep := sbom.NewSyftDependency(layer.Path, []sbom.SyftArtifact{sbomArtifact})
		h.Logger.Debugf("Writing Syft SBOM at %s: %+v", sbomPath, dep)
		if err := dep.WriteTo(sbomPath); err != nil {
			return fmt.Errorf("unable to write SBOM\n%w", err)
		}

		return nil
	})
}

func (h HelperLayerContributor) AsSyftArtifact() (sbom.SyftArtifact, error) {
	licenses := []string{}
	for _, license := range h.BuildpackInfo.Licenses {
		licenses = append(licenses, license.Type)
	}

	locations := []sbom.SyftLocation{}
	cpes := []string{}
	for _, name := range h.Names {
		locations = append(locations, sbom.SyftLocation{Path: name})
		cpes = append(cpes, fmt.Sprintf("cpe:2.3:a:%s:%s:%s:*:*:*:*:*:*:*",
			h.BuildpackInfo.ID, name, h.BuildpackInfo.Version))
	}

	artifact := sbom.SyftArtifact{
		Name:      "helper",
		Version:   h.BuildpackInfo.Version,
		Type:      "UnknownPackage",
		FoundBy:   "libpak",
		Licenses:  licenses,
		Locations: locations,
		CPEs:      cpes,
		PURL:      fmt.Sprintf("pkg:generic/%s@%s", h.BuildpackInfo.ID, h.BuildpackInfo.Version),
	}
	var err error
	artifact.ID, err = artifact.Hash()
	if err != nil {
		return sbom.SyftArtifact{}, fmt.Errorf("unable to generate hash\n%w", err)
	}

	return artifact, nil
}
