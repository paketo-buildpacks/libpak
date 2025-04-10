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

package libpak_test

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/buildpacks/libcnb/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"

	"github.com/paketo-buildpacks/libpak/v2"
	"github.com/paketo-buildpacks/libpak/v2/log"
	"github.com/paketo-buildpacks/libpak/v2/mocks"
)

func testLayer(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir string
		layer     *libcnb.Layer
	)

	it.Before(func() {
		layersDir = t.TempDir()
		layer = &libcnb.Layer{}
		layer.Path = filepath.Join(layersDir, "test-layer")

		layer.Exec.Path = layer.Path
		layer.Metadata = map[string]interface{}{}
	})

	context("LayerContributor", func() {
		var (
			lc libpak.LayerContributor
		)

		it.Before(func() {
			lc.Logger = log.NewPaketoLogger(bytes.NewBuffer(nil))
			lc.ExpectedMetadata = map[string]interface{}{
				"alpha": "test-alpha",
				"bravo": map[string]interface{}{
					"bravo-1": "test-bravo-1",
					"bravo-2": "test-bravo-2",
				},
			}
		})

		it("calls function with no existing metadata", func() {
			var called bool

			err := lc.Contribute(layer, func(_ *libcnb.Layer) error {
				called = true
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeTrue())
		})

		it("calls function with non-matching metadata", func() {
			layer.Metadata["alpha"] = "test-alpha"

			var called bool

			err := lc.Contribute(layer, func(_ *libcnb.Layer) error {
				called = true
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeTrue())
		})

		context("reloads layers not restored", func() {
			var called bool

			it.Before(func() {
				layer.Metadata = map[string]interface{}{
					"alpha": "test-alpha",
					"bravo": map[string]interface{}{
						"bravo-1": "test-bravo-1",
						"bravo-2": "test-bravo-2",
					},
				}
				called = false
			})

			it("calls function with matching metadata but no layer directory on cache layer", func() {
				Expect(os.WriteFile(fmt.Sprintf("%s.toml", layer.Path), []byte{}, 0600)).To(Succeed())
				Expect(os.RemoveAll(layer.Path)).To(Succeed())
				lc.ExpectedTypes.Cache = true

				err := lc.Contribute(layer, func(_ *libcnb.Layer) error {
					called = true
					return nil
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(called).To(BeTrue())
			})

			it("calls function with matching metadata but no layer directory on build layer", func() {
				Expect(os.WriteFile(fmt.Sprintf("%s.toml", layer.Path), []byte{}, 0600)).To(Succeed())
				Expect(os.RemoveAll(layer.Path)).To(Succeed())
				lc.ExpectedTypes.Build = true

				err := lc.Contribute(layer, func(_ *libcnb.Layer) error {
					called = true
					return nil
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(called).To(BeTrue())
			})

			it("calls function with matching metadata but an empty layer directory on build layer", func() {
				Expect(os.WriteFile(fmt.Sprintf("%s.toml", layer.Path), []byte{}, 0600)).To(Succeed())
				Expect(os.MkdirAll(layer.Path, 0755)).To(Succeed())
				lc.ExpectedTypes.Build = true

				err := lc.Contribute(layer, func(_ *libcnb.Layer) error {
					called = true
					return nil
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(called).To(BeTrue())
			})

			it("does not call function with matching metadata when layer directory exists and has a file in it", func() {
				Expect(os.WriteFile(fmt.Sprintf("%s.toml", layer.Path), []byte{}, 0600)).To(Succeed())
				Expect(os.MkdirAll(layer.Path, 0755)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(layer.Path, "foo"), []byte{}, 0600)).To(Succeed())
				lc.ExpectedTypes.Build = true

				err := lc.Contribute(layer, func(_ *libcnb.Layer) error {
					called = true
					return nil
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(called).To(BeFalse())
			})

			it("does not call function with matching metadata when layer TOML missing", func() {
				Expect(os.MkdirAll(layer.Path, 0755)).To(Succeed())
				layer.Build = true

				err := lc.Contribute(layer, func(_ *libcnb.Layer) error {
					called = true
					return nil
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(called).To(BeFalse())
			})
		})

		it("does not call function with matching metadata", func() {
			layer.Metadata = map[string]interface{}{
				"alpha": "test-alpha",
				"bravo": map[string]interface{}{
					"bravo-1": "test-bravo-1",
					"bravo-2": "test-bravo-2",
				},
			}

			var called bool

			err := lc.Contribute(layer, func(_ *libcnb.Layer) error {
				called = true
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeFalse())
		})

		it("returns function error", func() {
			err := lc.Contribute(layer, func(_ *libcnb.Layer) error {
				return fmt.Errorf("test-error")
			})
			Expect(err).To(MatchError("test-error"))
		})

		it("adds expected metadata to layer", func() {
			err := lc.Contribute(layer, func(_ *libcnb.Layer) error {
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.Metadata).To(Equal(map[string]interface{}{
				"alpha": "test-alpha",
				"bravo": map[string]interface{}{
					"bravo-1": "test-bravo-1",
					"bravo-2": "test-bravo-2",
				},
			}))
		})

		it("sets build layer flag", func() {
			lc.ExpectedTypes.Build = true
			err := lc.Contribute(layer, func(_ *libcnb.Layer) error {
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.LayerTypes.Build).To(BeTrue())
		})

		it("sets cache layer flag", func() {
			lc.ExpectedTypes.Cache = true
			err := lc.Contribute(layer, func(_ *libcnb.Layer) error {
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.LayerTypes.Cache).To(BeTrue())
		})

		it("sets launch layer flag", func() {
			lc.ExpectedTypes.Launch = true
			err := lc.Contribute(layer, func(_ *libcnb.Layer) error {
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.LayerTypes.Launch).To(BeTrue())
		})
	})

	context("DependencyLayerContributor", func() {
		var (
			dependency libpak.BuildModuleDependency
			dlc        libpak.DependencyLayerContributor
			server     *ghttp.Server
		)

		it.Before(func() {
			RegisterTestingT(t)
			server = ghttp.NewServer()

			deprecationDate, err := time.Parse(time.RFC3339, "2021-04-01T00:00:00Z")
			Expect(err).ToNot(HaveOccurred())

			dependency = libpak.BuildModuleDependency{
				ID:       "test-id",
				Name:     "test-name",
				Version:  "1.1.1",
				URI:      fmt.Sprintf("%s/test-path", server.URL()),
				Checksum: "sha256:576dd8416de5619ea001d9662291d62444d1292a38e96956bc4651c01f14bca1",
				Stacks:   []string{"test-stack"},
				Licenses: []libpak.BuildModuleDependencyLicense{
					{
						Type: "test-type",
						URI:  "test-uri",
					},
				},
				CPEs:    []string{"cpe:2.3:a:some:jre:11.0.2:*:*:*:*:*:*:*"},
				PURLS:   []string{"pkg:generic/some-java11@11.0.2?arch=amd64"},
				EOLDate: deprecationDate,
			}

			layer.Metadata = map[string]interface{}{}

			dlc.Logger = log.NewDiscardLogger()
			dlc.ExpectedMetadata = dependency.GetMetadata()
			dlc.Dependency = dependency
			dlc.DependencyCache.CachePath = layer.Path
			dlc.DependencyCache.DownloadPath = layer.Path
			dlc.DependencyCache.Logger = log.NewDiscardLogger()
		})

		it.After(func() {
			server.Close()
		})

		it("calls function with no existing metadata", func() {
			server.AppendHandlers(ghttp.RespondWith(http.StatusOK, "test-fixture"))

			var called bool

			err := dlc.Contribute(layer, func(_ *libcnb.Layer, artifact *os.File) error {
				defer artifact.Close()

				called = true
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeTrue())
		})

		it("modifies request", func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyHeaderKV("Test-Key", "test-value"),
				ghttp.RespondWith(http.StatusOK, "test-fixture"),
			))

			dlc.RequestModifierFuncs = append(dlc.RequestModifierFuncs, func(request *http.Request) (*http.Request, error) {
				request.Header.Add("Test-Key", "test-value")
				return request, nil
			})

			err := dlc.Contribute(layer, func(_ *libcnb.Layer, artifact *os.File) error {
				defer artifact.Close()
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
		})

		it("calls function with non-matching metadata", func() {
			layer.Metadata["alpha"] = "test-alpha"

			server.AppendHandlers(ghttp.RespondWith(http.StatusOK, "test-fixture"))

			var called bool

			err := dlc.Contribute(layer, func(_ *libcnb.Layer, artifact *os.File) error {
				defer artifact.Close()

				called = true
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeTrue())
		})

		it("does not call function with matching metadata", func() {
			layer.Metadata = map[string]interface{}{
				"id":       dependency.ID,
				"name":     dependency.Name,
				"version":  dependency.Version,
				"checksum": string(dependency.GetChecksum()),
			}

			var called bool

			err := dlc.Contribute(layer, func(_ *libcnb.Layer, artifact *os.File) error {
				defer artifact.Close()

				called = true
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeFalse())
		})

		it("returns function error", func() {
			server.AppendHandlers(ghttp.RespondWith(http.StatusOK, "test-fixture"))

			err := dlc.Contribute(layer, func(_ *libcnb.Layer, artifact *os.File) error {
				defer artifact.Close()

				return fmt.Errorf("test-error")
			})
			Expect(err).To(MatchError("test-error"))
		})

		it("adds expected metadata to layer", func() {
			server.AppendHandlers(ghttp.RespondWith(http.StatusOK, "test-fixture"))

			err := dlc.Contribute(layer, func(_ *libcnb.Layer, artifact *os.File) error {
				defer artifact.Close()
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.Metadata).To(Equal(map[string]interface{}{
				"id":       dependency.ID,
				"name":     dependency.Name,
				"version":  dependency.Version,
				"checksum": string(dependency.GetChecksum()),
			}))
		})

		it("adds expected Syft SBOM file", func() {
			server.AppendHandlers(ghttp.RespondWith(http.StatusOK, "test-fixture"))

			err := dlc.Contribute(layer, func(_ *libcnb.Layer, artifact *os.File) error {
				defer artifact.Close()
				return nil
			})
			Expect(err).NotTo(HaveOccurred())

			outputFile := layer.SBOMPath(libcnb.SyftJSON)
			Expect(outputFile).To(BeARegularFile())

			data, err := os.ReadFile(outputFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(`"Artifacts":[`))
			Expect(string(data)).To(ContainSubstring(`"FoundBy":"libpak",`))
			Expect(string(data)).To(ContainSubstring(`"PURL":"pkg:generic/some-java11@11.0.2?arch=amd64"`))
			Expect(string(data)).To(ContainSubstring(`"Schema":{`))
			Expect(string(data)).To(ContainSubstring(`"Descriptor":{`))
			Expect(string(data)).To(ContainSubstring(`"Source":{`))
		})
	})

	context("HelperLayerContributor", func() {
		var (
			buildpack libcnb.Buildpack
			hlc       libpak.HelperLayerContributor
		)

		it.Before(func() {
			buildpack.Info = libcnb.BuildpackInfo{
				ID:       "test-id",
				Name:     "test-name",
				Version:  "test-version",
				Homepage: "test-homepage",
			}

			buildpack.Path = t.TempDir()
			file := filepath.Join(buildpack.Path, "bin")
			Expect(os.MkdirAll(file, 0755)).To(Succeed())

			file = filepath.Join(file, "helper")
			// #nosec G306 - permissions need to be 755 on the helper as it should be executable
			Expect(os.WriteFile(file, []byte{}, 0755)).To(Succeed())

			hlc = libpak.HelperLayerContributor{
				Path:          file,
				BuildpackInfo: buildpack.Info,
				Logger:        log.NewPaketoLogger(bytes.NewBuffer(nil)),
				Names:         []string{"test-name-1", "test-name-2"},
			}
		})

		it.After(func() {
			Expect(os.RemoveAll(buildpack.Path)).To(Succeed())
		})

		it("calls function with no existing metadata", func() {
			err := hlc.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.Exec.FilePath("test-name-1")).To(BeAnExistingFile())
		})

		it("calls function with non-matching metadata", func() {
			layer.Metadata["alpha"] = "other-alpha"

			err := hlc.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			file := layer.Exec.FilePath("test-name-1")
			Expect(file).To(BeAnExistingFile())
			Expect(os.Readlink(file)).To(Equal(filepath.Join(layer.Path, "helper")))

			file = layer.Exec.FilePath("test-name-2")
			Expect(file).To(BeAnExistingFile())
			Expect(os.Readlink(file)).To(Equal(filepath.Join(layer.Path, "helper")))
		})

		it("does not call function with matching metadata", func() {
			buildpackInfo := map[string]interface{}{
				"id":          buildpack.Info.ID,
				"name":        buildpack.Info.Name,
				"version":     buildpack.Info.Version,
				"homepage":    buildpack.Info.Homepage,
				"clear-env":   buildpack.Info.ClearEnvironment,
				"description": "",
			}
			layer.Metadata["buildpackInfo"] = buildpackInfo
			layer.Metadata["helperNames"] = []interface{}{hlc.Names[0], hlc.Names[1]}

			err := hlc.Contribute(layer)

			Expect(err).NotTo(HaveOccurred())

			Expect(layer.Exec.FilePath("test-name-1")).NotTo(BeAnExistingFile())
			Expect(layer.Exec.FilePath("test-name-2")).NotTo(BeAnExistingFile())
		})

		it("adds expected metadata to layer", func() {
			err := hlc.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			buildpackInfo := map[string]interface{}{
				"id":          buildpack.Info.ID,
				"name":        buildpack.Info.Name,
				"version":     buildpack.Info.Version,
				"homepage":    buildpack.Info.Homepage,
				"clear-env":   buildpack.Info.ClearEnvironment,
				"description": "",
			}
			Expect(layer.Metadata).To(Equal(map[string]interface{}{"buildpackInfo": buildpackInfo, "helperNames": []interface{}{hlc.Names[0], hlc.Names[1]}}))
		})

		it("adds expected Syft SBOM file", func() {
			layer.Metadata = map[string]interface{}{}

			err := hlc.Contribute(layer)
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.Exec.FilePath("test-name-1")).To(BeAnExistingFile())
			Expect(layer.Exec.FilePath("test-name-2")).To(BeAnExistingFile())

			outputFile := layer.SBOMPath(libcnb.SyftJSON)
			Expect(outputFile).To(BeARegularFile())

			data, err := os.ReadFile(outputFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(`"Artifacts":[`))
			Expect(string(data)).To(ContainSubstring(`"FoundBy":"libpak",`))
			Expect(string(data)).To(ContainSubstring(`"PURL":"pkg:generic/test-id@test-version"`))
			Expect(string(data)).To(ContainSubstring(`"CPEs":["cpe:2.3:a:test-id:test-name-1:test-version:*:*:*:*:*:*:*","cpe:2.3:a:test-id:test-name-2:test-version:*:*:*:*:*:*:*"]`))
			Expect(string(data)).To(ContainSubstring(`"Schema":{`))
			Expect(string(data)).To(ContainSubstring(`"Descriptor":{`))
			Expect(string(data)).To(ContainSubstring(`"Source":{`))
		})
	})

	context("ContributableBuildFunc", func() {
		it("calls contributeLayersFunc and processes each layer contributor", func() {
			mockContributor1 := &mocks.Contributable{}
			mockContributor2 := &mocks.Contributable{}

			mockContributor1.On("Name").Return("layer-1")
			mockContributor2.On("Name").Return("layer-2")

			mockContributor1.On("Contribute", mock.Anything).Return(nil)
			mockContributor2.On("Contribute", mock.Anything).Return(nil)

			buildFunc := libpak.ContributableBuildFunc(func(context libcnb.BuildContext, result *libcnb.BuildResult) ([]libpak.Contributable, error) {
				return []libpak.Contributable{
					mockContributor1,
					mockContributor2,
				}, nil
			})

			buildResult, err := buildFunc(libcnb.BuildContext{})

			Expect(err).NotTo(HaveOccurred())
			Expect(buildResult.Layers[0].Name).To(Equal("layer-1"))
			Expect(buildResult.Layers[1].Name).To(Equal("layer-2"))
			Expect(mockContributor1.Calls).To(HaveLen(2))
			Expect(mockContributor2.Calls).To(HaveLen(2))
		})

		it("calls contributeLayersFunc and processes each layer contributor and process type contributor", func() {
			mockContributor1 := &mocks.ProcessContributable{}
			mockContributor2 := &mocks.ProcessContributable{}

			mockContributor1.On("Name").Return("layer-1")
			mockContributor2.On("Name").Return("layer-2")

			mockContributor1.On("Contribute", mock.Anything).Return(nil)
			mockContributor2.On("Contribute", mock.Anything).Return(nil)

			proc1 := []libcnb.Process{{Type: "foo", Command: []string{"bar"}}}
			proc2 := []libcnb.Process{{Type: "baz", Command: []string{"qux"}}}
			mockContributor1.On("ProcessTypes", mock.Anything).Return(proc1, nil)
			mockContributor2.On("ProcessTypes", mock.Anything).Return(proc2, nil)

			buildFunc := libpak.ContributableBuildFunc(func(context libcnb.BuildContext, result *libcnb.BuildResult) ([]libpak.Contributable, error) {
				return []libpak.Contributable{
					mockContributor1,
					mockContributor2,
				}, nil
			})

			buildResult, err := buildFunc(libcnb.BuildContext{})

			Expect(err).NotTo(HaveOccurred())
			Expect(buildResult.Layers[0].Name).To(Equal("layer-1"))
			Expect(buildResult.Layers[1].Name).To(Equal("layer-2"))
			Expect(buildResult.Processes).To(Equal(append(proc1, proc2...)))
			Expect(mockContributor1.Calls).To(HaveLen(3))
			Expect(mockContributor2.Calls).To(HaveLen(3))
		})
	})
}
