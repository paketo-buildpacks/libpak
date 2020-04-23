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

package libpak_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/paketo-buildpacks/libpak"
	"github.com/sclevine/spec"
)

func testLayer(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layer libcnb.Layer
		path  string
	)

	it.Before(func() {
		var err error

		path, err = ioutil.TempDir("", "layer")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	context("LayerContributor", func() {
		var (
			lc libpak.LayerContributor
		)

		it.Before(func() {
			layer.Metadata = map[string]interface{}{}
			layer.Path = path
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

			_, err := lc.Contribute(layer, func() (libcnb.Layer, error) {
				called = true
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeTrue())
		})

		it("calls function with non-matching metadata", func() {
			layer.Metadata["alpha"] = "test-alpha"

			var called bool

			_, err := lc.Contribute(layer, func() (libcnb.Layer, error) {
				called = true
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeTrue())
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

			_, err := lc.Contribute(layer, func() (libcnb.Layer, error) {
				called = true
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeFalse())
		})

		it("returns function error", func() {
			_, err := lc.Contribute(layer, func() (libcnb.Layer, error) {
				return libcnb.Layer{}, fmt.Errorf("test-error")
			})
			Expect(err).To(MatchError("test-error"))
		})

		it("adds expected metadata to layer", func() {
			layer, err := lc.Contribute(layer, func() (libcnb.Layer, error) {
				return layer, nil
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
	})

	context("DependencyLayerContributor", func() {
		var (
			dependency libpak.BuildpackDependency
			dlc        libpak.DependencyLayerContributor
			server     *ghttp.Server
		)

		it.Before(func() {
			RegisterTestingT(t)
			server = ghttp.NewServer()

			dependency = libpak.BuildpackDependency{
				ID:      "test-id",
				Name:    "test-name",
				Version: "1.1.1",
				URI:     fmt.Sprintf("%s/test-path", server.URL()),
				SHA256:  "576dd8416de5619ea001d9662291d62444d1292a38e96956bc4651c01f14bca1",
				Stacks:  []string{"test-stack"},
				Licenses: []libpak.BuildpackDependencyLicense{
					{
						Type: "test-type",
						URI:  "test-uri",
					},
				},
			}

			layer.Metadata = map[string]interface{}{}

			dlc.LayerContributor.ExpectedMetadata = dependency

			layer.Path = path
			dlc.Dependency = dependency
			dlc.DependencyCache.CachePath = path
			dlc.DependencyCache.DownloadPath = path
		})

		it.After(func() {
			server.Close()
		})

		it("calls function with no existing metadata", func() {
			server.AppendHandlers(ghttp.RespondWith(http.StatusOK, "test-fixture"))

			var called bool

			_, err := dlc.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
				defer artifact.Close()

				called = true
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeTrue())
		})

		it("modifies request", func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyHeaderKV("Test-Key", "test-value"),
				ghttp.RespondWith(http.StatusOK, "test-fixture"),
			))

			dlc.RequestModifierFunc = func(request *http.Request) (*http.Request, error) {
				request.Header.Add("Test-Key", "test-value")
				return request, nil
			}

			_, err := dlc.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
				defer artifact.Close()
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())
		})

		it("calls function with non-matching metadata", func() {
			layer.Metadata["alpha"] = "test-alpha"

			server.AppendHandlers(ghttp.RespondWith(http.StatusOK, "test-fixture"))

			var called bool

			_, err := dlc.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
				defer artifact.Close()

				called = true
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeTrue())
		})

		it("does not call function with matching metadata", func() {
			layer.Metadata = map[string]interface{}{
				"id":      dependency.ID,
				"name":    dependency.Name,
				"version": dependency.Version,
				"uri":     dependency.URI,
				"sha256":  dependency.SHA256,
				"stacks":  dependency.Stacks,
				"licenses": []map[string]interface{}{
					{
						"type": dependency.Licenses[0].Type,
						"uri":  dependency.Licenses[0].URI,
					},
				},
			}

			var called bool

			_, err := dlc.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
				defer artifact.Close()

				called = true
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeFalse())
		})

		it("returns function error", func() {
			server.AppendHandlers(ghttp.RespondWith(http.StatusOK, "test-fixture"))

			_, err := dlc.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
				defer artifact.Close()

				return libcnb.Layer{}, fmt.Errorf("test-error")
			})
			Expect(err).To(MatchError("test-error"))
		})

		it("adds expected metadata to layer", func() {
			server.AppendHandlers(ghttp.RespondWith(http.StatusOK, "test-fixture"))

			layer, err := dlc.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
				defer artifact.Close()
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.Metadata).To(Equal(map[string]interface{}{
				"id":      dependency.ID,
				"name":    dependency.Name,
				"version": dependency.Version,
				"uri":     dependency.URI,
				"sha256":  dependency.SHA256,
				"stacks":  dependency.Stacks,
				"licenses": []libpak.BuildpackDependencyLicense{
					{
						Type: dependency.Licenses[0].Type,
						URI:  dependency.Licenses[0].URI,
					},
				},
			}))
		})

		it("contributes to buildpack plan", func() {
			plan := libcnb.BuildpackPlan{}

			_ = libpak.NewDependencyLayerContributor(dependency, libpak.DependencyCache{}, &plan)

			Expect(plan.Entries).To(ContainElement(libcnb.BuildpackPlanEntry{
				Name:    dependency.ID,
				Version: dependency.Version,
				Metadata: map[string]interface{}{
					"name":     dependency.Name,
					"uri":      dependency.URI,
					"sha256":   dependency.SHA256,
					"stacks":   dependency.Stacks,
					"licenses": dependency.Licenses,
				},
			}))
		})
	})

	context("HelperLayerContributor", func() {
		var (
			helper *os.File
			hlc    libpak.HelperLayerContributor
			info   libcnb.BuildpackInfo
		)

		it.Before(func() {
			var err error
			helper, err = ioutil.TempFile("", "layer")
			Expect(err).NotTo(HaveOccurred())
			Expect(helper.Close()).To(Succeed())

			info = libcnb.BuildpackInfo{
				ID:       "test-id",
				Name:     "test-name",
				Version:  "test-version",
				Homepage: "test-homepage",
			}

			layer.Metadata = map[string]interface{}{}
			layer.Path = path

			hlc.LayerContributor.ExpectedMetadata = info
			hlc.Path = helper.Name()
		})

		it.After(func() {
			Expect(os.RemoveAll(hlc.Path)).To(Succeed())
		})

		it("calls function with no existing metadata", func() {
			var called bool

			_, err := hlc.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
				defer artifact.Close()

				called = true
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeTrue())
		})

		it("calls function with non-matching metadata", func() {
			layer.Metadata["alpha"] = "other-alpha"

			var called bool

			_, err := hlc.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
				defer artifact.Close()

				called = true
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeTrue())
		})

		it("does not call function with matching metadata", func() {
			layer.Metadata = map[string]interface{}{
				"id":        info.ID,
				"name":      info.Name,
				"version":   info.Version,
				"homepage":  info.Homepage,
				"clear-env": info.ClearEnvironment,
			}

			var called bool

			_, err := hlc.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
				defer artifact.Close()

				called = true
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(called).To(BeFalse())
		})

		it("returns function error", func() {
			_, err := hlc.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
				defer artifact.Close()

				return libcnb.Layer{}, fmt.Errorf("test-error")
			})
			Expect(err).To(MatchError("test-error"))
		})

		it("adds expected metadata to layer", func() {
			layer, err := hlc.Contribute(layer, func(artifact *os.File) (libcnb.Layer, error) {
				defer artifact.Close()
				return layer, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(layer.Metadata).To(Equal(map[string]interface{}{
				"id":        info.ID,
				"name":      info.Name,
				"version":   info.Version,
				"homepage":  info.Homepage,
				"clear-env": info.ClearEnvironment,
			}))
		})

		it("contributes to buildpack plan", func() {
			plan := libcnb.BuildpackPlan{}

			_ = libpak.NewHelperLayerContributor(helper.Name(), "test-name", info, &plan)

			Expect(plan.Entries).To(ContainElement(libcnb.BuildpackPlanEntry{
				Name:    filepath.Base(helper.Name()),
				Version: info.Version,
			}))
		})
	})
}
