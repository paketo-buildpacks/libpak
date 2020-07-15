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
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak"
)

func testDependencyCache(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		dependencyCache libpak.DependencyCache
	)

	context("NewDependencyCache", func() {
		it.Before(func() {
			var err error
			dependencyCache, err = libpak.NewDependencyCache(libcnb.BuildContext{
				Buildpack: libcnb.Buildpack{
					Info: libcnb.BuildpackInfo{
						ID:      "some-buildpack-id",
						Version: "some-buildpack-version",
					},
					Path: "some/path",
				},
				Platform: libcnb.Platform{
					Path: filepath.Join("testdata", "platform"),
				},
			})
			Expect(err).NotTo(HaveOccurred())
		})

		it("defaults CachePath to <BUILDPACK_PATH>/dependencies", func() {
			Expect(dependencyCache.CachePath).To(Equal(filepath.Join("some/path/dependencies")))
		})
		it("sets the user agent to <BUILDPACK_ID>/<BUILDPACK_VERSION>", func() {
			Expect(dependencyCache.UserAgent).To(Equal("some-buildpack-id/some-buildpack-version"))
		})
		it("reads mappings for buildpack from the platform dir", func() {
			Expect(dependencyCache.Mappings).To(ConsistOf(
				libpak.DependencyMapping{
					ID:      "some-dependency-id",
					Version: "some-dependency-version",
					URI:     "some-uri",
				},
				libpak.DependencyMapping{
					ID:      "other-dependency-id",
					Version: "other-dependency-version",
					URI:     "other-uri",
				},
			))
		})
	})

	context("artifacts", func() {
		var (
			cachePath    string
			downloadPath string
			dependency   libpak.BuildpackDependency
			server       *ghttp.Server
		)

		it.Before(func() {
			var err error

			cachePath, err = ioutil.TempDir("", "dependency-cache-cache-path")
			Expect(err).NotTo(HaveOccurred())

			downloadPath, err = ioutil.TempDir("", "dependency-cache-download-path")
			Expect(err).NotTo(HaveOccurred())

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

			dependencyCache = libpak.DependencyCache{
				CachePath:    cachePath,
				DownloadPath: downloadPath,
				UserAgent:    "test-user-agent",
			}
		})

		it.After(func() {
			Expect(os.RemoveAll(cachePath)).To(Succeed())
			Expect(os.RemoveAll(downloadPath)).To(Succeed())
			server.Close()
		})

		copyFile := func(source string, destination string) {
			in, err := os.Open(source)
			Expect(err).NotTo(HaveOccurred())
			defer in.Close()

			Expect(os.MkdirAll(filepath.Dir(destination), 0755)).To(Succeed())
			out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
			Expect(err).NotTo(HaveOccurred())
			defer out.Close()

			_, err = io.Copy(out, in)
			Expect(err).NotTo(HaveOccurred())
		}

		writeTOML := func(destination string, v interface{}) {
			Expect(os.MkdirAll(filepath.Dir(destination), 0755)).To(Succeed())
			out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
			Expect(err).NotTo(HaveOccurred())
			defer out.Close()

			Expect(toml.NewEncoder(out).Encode(v)).To(Succeed())
		}

		it("returns from cache path", func() {
			copyFile(filepath.Join("testdata", "test-file"), filepath.Join(cachePath, dependency.SHA256, "test-path"))
			writeTOML(filepath.Join(cachePath, fmt.Sprintf("%s.toml", dependency.SHA256)), dependency)

			a, err := dependencyCache.Artifact(dependency)
			Expect(err).NotTo(HaveOccurred())

			Expect(ioutil.ReadAll(a)).To(Equal([]byte("test-fixture")))
		})

		it("returns from download path", func() {
			copyFile(filepath.Join("testdata", "test-file"), filepath.Join(downloadPath, dependency.SHA256, "test-path"))
			writeTOML(filepath.Join(downloadPath, fmt.Sprintf("%s.toml", dependency.SHA256)), dependency)

			a, err := dependencyCache.Artifact(dependency)
			Expect(err).NotTo(HaveOccurred())

			Expect(ioutil.ReadAll(a)).To(Equal([]byte("test-fixture")))
		})

		it("downloads", func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/test-path", ""),
				ghttp.RespondWith(http.StatusOK, "test-fixture"),
			))

			a, err := dependencyCache.Artifact(dependency)
			Expect(err).NotTo(HaveOccurred())

			Expect(ioutil.ReadAll(a)).To(Equal([]byte("test-fixture")))
		})

		context("uri is overridden", func() {
			it.Before(func() {
				dependencyCache.Mappings = []libpak.DependencyMapping{
					{
						ID:      dependency.ID,
						Version: dependency.Version,
						URI:     fmt.Sprintf("%s/override-path", server.URL()),
					},
				}
			})

			it("downloads from override uri", func() {
				server.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, "/override-path", ""),
					ghttp.RespondWith(http.StatusOK, "test-fixture"),
				))

				a, err := dependencyCache.Artifact(dependency)
				Expect(err).NotTo(HaveOccurred())

				Expect(ioutil.ReadAll(a)).To(Equal([]byte("test-fixture")))
			})
		})

		it("fails with invalid SHA256", func() {
			server.AppendHandlers(ghttp.RespondWith(http.StatusOK, "invalid-fixture"))

			_, err := dependencyCache.Artifact(dependency)
			Expect(err).To(HaveOccurred())
		})

		it("skips cache with empty SHA256", func() {
			copyFile(filepath.Join("testdata", "test-file"), filepath.Join(cachePath, dependency.SHA256, "test-path"))
			writeTOML(filepath.Join(cachePath, fmt.Sprintf("%s.toml", dependency.SHA256)), dependency)
			copyFile(filepath.Join("testdata", "test-file"), filepath.Join(downloadPath, dependency.SHA256, "test-path"))
			writeTOML(filepath.Join(downloadPath, fmt.Sprintf("%s.toml", dependency.SHA256)), dependency)

			dependency.SHA256 = ""
			server.AppendHandlers(ghttp.RespondWith(http.StatusOK, "alternate-fixture"))

			a, err := dependencyCache.Artifact(dependency)
			Expect(err).NotTo(HaveOccurred())

			Expect(ioutil.ReadAll(a)).To(Equal([]byte("alternate-fixture")))
		})

		it("sets User-Agent", func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyHeaderKV("User-Agent", "test-user-agent"),
				ghttp.RespondWith(http.StatusOK, "test-fixture"),
			))

			a, err := dependencyCache.Artifact(dependency)
			Expect(err).NotTo(HaveOccurred())

			Expect(ioutil.ReadAll(a)).To(Equal([]byte("test-fixture")))
		})

		it("modifies request", func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyHeaderKV("User-Agent", "test-user-agent"),
				ghttp.VerifyHeaderKV("Test-Key", "test-value"),
				ghttp.RespondWith(http.StatusOK, "test-fixture"),
			))

			a, err := dependencyCache.ArtifactWithRequestModification(dependency, func(request *http.Request) (*http.Request, error) {
				request.Header.Add("Test-Key", "test-value")
				return request, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(ioutil.ReadAll(a)).To(Equal([]byte("test-fixture")))
		})
	})
}
