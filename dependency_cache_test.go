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
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
)

func testDependencyCache(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	context("NewDependencyCache", func() {
		var ctx libcnb.BuildContext

		it.Before(func() {
			ctx = libcnb.BuildContext{
				Buildpack: libcnb.Buildpack{
					Info: libcnb.BuildpackInfo{
						ID:      "some-buildpack-id",
						Version: "some-buildpack-version",
					},
					Path: "some/path",
				},
			}
		})

		it("set default CachePath and UserAgent", func() {
			dependencyCache, err := libpak.NewDependencyCache(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(dependencyCache.CachePath).To(Equal(filepath.Join("some/path/dependencies")))
			Expect(dependencyCache.UserAgent).To(Equal("some-buildpack-id/some-buildpack-version"))
			Expect(dependencyCache.Mappings).To(Equal(map[string]string{}))
		})

		it("uses default timeout values", func() {
			dependencyCache, err := libpak.NewDependencyCache(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(dependencyCache.HttpClientTimeouts.DialerTimeout).To(Equal(6 * time.Second))
			Expect(dependencyCache.HttpClientTimeouts.DialerKeepAlive).To(Equal(60 * time.Second))
			Expect(dependencyCache.HttpClientTimeouts.TLSHandshakeTimeout).To(Equal(5 * time.Second))
			Expect(dependencyCache.HttpClientTimeouts.ResponseHeaderTimeout).To(Equal(5 * time.Second))
			Expect(dependencyCache.HttpClientTimeouts.ExpectContinueTimeout).To(Equal(1 * time.Second))
		})

		context("custom timeout setttings", func() {
			it.Before(func() {
				t.Setenv("BP_DIALER_TIMEOUT", "7")
				t.Setenv("BP_DIALER_KEEP_ALIVE", "50")
				t.Setenv("BP_TLS_HANDSHAKE_TIMEOUT", "2")
				t.Setenv("BP_RESPONSE_HEADER_TIMEOUT", "3")
				t.Setenv("BP_EXPECT_CONTINUE_TIMEOUT", "2")
			})

			it("uses custom timeout values", func() {
				dependencyCache, err := libpak.NewDependencyCache(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(dependencyCache.HttpClientTimeouts.DialerTimeout).To(Equal(7 * time.Second))
				Expect(dependencyCache.HttpClientTimeouts.DialerKeepAlive).To(Equal(50 * time.Second))
				Expect(dependencyCache.HttpClientTimeouts.TLSHandshakeTimeout).To(Equal(2 * time.Second))
				Expect(dependencyCache.HttpClientTimeouts.ResponseHeaderTimeout).To(Equal(3 * time.Second))
				Expect(dependencyCache.HttpClientTimeouts.ExpectContinueTimeout).To(Equal(2 * time.Second))
			})
		})

		context("bindings with type dependencies exist", func() {
			it.Before(func() {
				ctx.Platform.Bindings = libcnb.Bindings{
					{
						Type: "dependency-mapping",
						Secret: map[string]string{
							"some-digest1": "some-uri1",
							"some-digest2": "some-uri2",
						},
					},
					{
						Type: "not-dependency-mapping",
						Secret: map[string]string{
							"some-thing": "other-thing",
						},
					},
					{
						Type: "dependency-mapping",
						Secret: map[string]string{
							"some-digest3": "some-uri3",
							"some-digest4": "some-uri4",
						},
					},
				}
			})

			it("sets Mappings", func() {
				dependencyCache, err := libpak.NewDependencyCache(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(dependencyCache.Mappings).To(Equal(
					map[string]string{
						"some-digest1": "some-uri1",
						"some-digest2": "some-uri2",
						"some-digest3": "some-uri3",
						"some-digest4": "some-uri4",
					},
				))
			})

			context("multiple bindings map the same digest", func() {
				it.Before(func() {
					ctx.Platform.Bindings = append(ctx.Platform.Bindings, libcnb.Binding{
						Type: "dependency-mapping",
						Secret: map[string]string{
							"some-digest1": "other-uri",
						},
					})
				})

				it("errors", func() {
					_, err := libpak.NewDependencyCache(ctx)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		context("dependency mirror from environment variable", func() {
			it.Before(func() {
				t.Setenv("BP_DEPENDENCY_MIRROR", "https://env-var-mirror.acme.com")
				t.Setenv("BP_DEPENDENCY_MIRROR_EXAMP__LE_COM", "https://examp-le.com")
			})

			it("uses BP_DEPENDENCY_MIRROR environment variable", func() {
				dependencyCache, err := libpak.NewDependencyCache(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(dependencyCache.DependencyMirrors["default"]).To(Equal("https://env-var-mirror.acme.com"))
				Expect(dependencyCache.DependencyMirrors["examp-le.com"]).To(Equal("https://examp-le.com"))
			})
		})

		context("dependency mirror from binding and environment variable", func() {
			it.Before(func() {
				t.Setenv("BP_DEPENDENCY_MIRROR_EXAMP__LE_COM", "https://examp-le.com")
				ctx.Platform.Bindings = append(ctx.Platform.Bindings, libcnb.Binding{
					Type: "dependency-mirror",
					Secret: map[string]string{
						"default":      "https://bindings-mirror.acme.com",
						"examp-le.com": "https://invalid.com",
					},
				})
			})

			it("uses dependency-mirror binding", func() {
				dependencyCache, err := libpak.NewDependencyCache(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(dependencyCache.DependencyMirrors["default"]).To(Equal("https://bindings-mirror.acme.com"))
			})

			it("environment variable overrides binding", func() {
				dependencyCache, err := libpak.NewDependencyCache(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(dependencyCache.DependencyMirrors["examp-le.com"]).To(Equal("https://examp-le.com"))
			})
		})
	})

	context("artifacts", func() {
		var (
			cachePath       string
			downloadPath    string
			dependency      libpak.BuildpackDependency
			dependencyCache libpak.DependencyCache
			server          *ghttp.Server
		)

		it.Before(func() {
			var err error

			cachePath = t.TempDir()
			Expect(err).NotTo(HaveOccurred())

			downloadPath = t.TempDir()
			Expect(err).NotTo(HaveOccurred())

			RegisterTestingT(t)
			server = ghttp.NewServer()

			dependency = libpak.BuildpackDependency{
				ID:              "test-id",
				Name:            "test-name",
				Version:         "1.1.1",
				URI:             fmt.Sprintf("%s/test-path", server.URL()),
				SHA256:          "576dd8416de5619ea001d9662291d62444d1292a38e96956bc4651c01f14bca1",
				Stacks:          []string{"test-stack"},
				DeprecationDate: time.Now(),
				Licenses: []libpak.BuildpackDependencyLicense{
					{
						Type: "test-type",
						URI:  "test-uri",
					},
				},
				CPEs: []string{"cpe:2.3:a:some:jre:11.0.2:*:*:*:*:*:*:*"},
				PURL: "pkg:generic/some-java11@11.0.2?arch=amd64",
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

			Expect(io.ReadAll(a)).To(Equal([]byte("test-fixture")))
		})

		it("returns from download path", func() {
			copyFile(filepath.Join("testdata", "test-file"), filepath.Join(downloadPath, dependency.SHA256, "test-path"))
			writeTOML(filepath.Join(downloadPath, fmt.Sprintf("%s.toml", dependency.SHA256)), dependency)

			a, err := dependencyCache.Artifact(dependency)
			Expect(err).NotTo(HaveOccurred())

			Expect(io.ReadAll(a)).To(Equal([]byte("test-fixture")))
		})

		it("downloads", func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest(http.MethodGet, "/test-path", ""),
				ghttp.RespondWith(http.StatusOK, "test-fixture"),
			))

			a, err := dependencyCache.Artifact(dependency)
			Expect(err).NotTo(HaveOccurred())

			Expect(io.ReadAll(a)).To(Equal([]byte("test-fixture")))
		})

		context("uri is overridden HTTP", func() {
			it.Before(func() {
				dependencyCache.Mappings = map[string]string{
					dependency.SHA256: fmt.Sprintf("%s/override-path", server.URL()),
				}
			})

			it("downloads from override uri", func() {
				server.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, "/override-path", ""),
					ghttp.RespondWith(http.StatusOK, "test-fixture"),
				))

				a, err := dependencyCache.Artifact(dependency)
				Expect(err).NotTo(HaveOccurred())

				Expect(io.ReadAll(a)).To(Equal([]byte("test-fixture")))
			})
		})

		context("uri is overridden FILE", func() {
			it.Before(func() {
				sourcePath := t.TempDir()
				sourceFile := filepath.Join(sourcePath, "source-file")
				Expect(os.WriteFile(sourceFile, []byte("test-fixture"), 0644)).ToNot(HaveOccurred())

				dependencyCache.Mappings = map[string]string{
					dependency.SHA256: fmt.Sprintf("file://%s", sourceFile),
				}
			})

			it("downloads from override filesystem", func() {
				a, err := dependencyCache.Artifact(dependency)
				Expect(err).NotTo(HaveOccurred())

				Expect(io.ReadAll(a)).To(Equal([]byte("test-fixture")))
			})
		})

		context("dependency mirror is used https", func() {
			var mirrorServer *ghttp.Server

			it.Before(func() {
				mirrorServer = ghttp.NewTLSServer()
				dependencyCache.DependencyMirrors = map[string]string{}
			})

			it.After(func() {
				mirrorServer.Close()
			})

			it("downloads from https mirror", func() {
				url, err := url.Parse(mirrorServer.URL())
				Expect(err).NotTo(HaveOccurred())
				mirrorServer.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyBasicAuth("username", "password"),
					ghttp.VerifyRequest(http.MethodGet, "/foo/bar/test-path", ""),
					ghttp.RespondWith(http.StatusOK, "test-fixture"),
				))

				dependencyCache.DependencyMirrors["default"] = url.Scheme + "://" + "username:password@" + url.Host + "/foo/bar"
				a, err := dependencyCache.Artifact(dependency)
				Expect(err).NotTo(HaveOccurred())

				Expect(io.ReadAll(a)).To(Equal([]byte("test-fixture")))
			})

			it("downloads from https mirror preserving hostname", func() {
				url, err := url.Parse(mirrorServer.URL())
				Expect(err).NotTo(HaveOccurred())
				mirrorServer.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, "/"+url.Hostname()+"/test-path", ""),
					ghttp.RespondWith(http.StatusOK, "test-fixture"),
				))

				dependencyCache.DependencyMirrors["default"] = url.Scheme + "://" + url.Host + "/{originalHost}"
				a, err := dependencyCache.Artifact(dependency)
				Expect(err).NotTo(HaveOccurred())

				Expect(io.ReadAll(a)).To(Equal([]byte("test-fixture")))
			})

			it("downloads from https mirror host specific", func() {
				url, err := url.Parse(mirrorServer.URL())
				Expect(err).NotTo(HaveOccurred())
				mirrorServer.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, "/host-specific/test-path", ""),
					ghttp.RespondWith(http.StatusOK, "test-fixture"),
				))

				dependencyCache.DependencyMirrors["127.0.0.1"] = url.Scheme + "://" + url.Host + "/host-specific"
				a, err := dependencyCache.Artifact(dependency)
				Expect(err).NotTo(HaveOccurred())

				Expect(io.ReadAll(a)).To(Equal([]byte("test-fixture")))
			})
		})

		context("dependency mirror is used file", func() {
			var (
				mirrorPath              string
				mirrorPathPreservedHost string
			)

			it.Before(func() {
				var err error
				mirrorPath, err = os.MkdirTemp("", "mirror-path")
				Expect(err).NotTo(HaveOccurred())
				originalUrl, err := url.Parse(dependency.URI)
				Expect(err).NotTo(HaveOccurred())
				mirrorPathPreservedHost = filepath.Join(mirrorPath, originalUrl.Hostname(), "prefix")
				Expect(os.MkdirAll(mirrorPathPreservedHost, os.ModePerm)).NotTo(HaveOccurred())
				dependencyCache.DependencyMirrors = map[string]string{}
			})

			it.After(func() {
				Expect(os.RemoveAll(mirrorPath)).To(Succeed())
			})

			it("downloads from file mirror", func() {
				mirrorFile := filepath.Join(mirrorPath, "test-path")
				Expect(os.WriteFile(mirrorFile, []byte("test-fixture"), 0644)).ToNot(HaveOccurred())

				dependencyCache.DependencyMirrors["default"] = "file://" + mirrorPath
				a, err := dependencyCache.Artifact(dependency)
				Expect(err).NotTo(HaveOccurred())

				Expect(io.ReadAll(a)).To(Equal([]byte("test-fixture")))
			})

			it("downloads from file mirror preserving hostname", func() {
				mirrorFilePreservedHost := filepath.Join(mirrorPathPreservedHost, "test-path")
				Expect(os.WriteFile(mirrorFilePreservedHost, []byte("test-fixture"), 0644)).ToNot(HaveOccurred())

				dependencyCache.DependencyMirrors["default"] = "file://" + mirrorPath + "/{originalHost}" + "/prefix"
				a, err := dependencyCache.Artifact(dependency)
				Expect(err).NotTo(HaveOccurred())

				Expect(io.ReadAll(a)).To(Equal([]byte("test-fixture")))
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

			Expect(io.ReadAll(a)).To(Equal([]byte("alternate-fixture")))
		})

		it("sets User-Agent", func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyHeaderKV("User-Agent", "test-user-agent"),
				ghttp.RespondWith(http.StatusOK, "test-fixture"),
			))

			a, err := dependencyCache.Artifact(dependency)
			Expect(err).NotTo(HaveOccurred())

			Expect(io.ReadAll(a)).To(Equal([]byte("test-fixture")))
		})

		it("modifies request", func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyHeaderKV("User-Agent", "test-user-agent"),
				ghttp.VerifyHeaderKV("Test-Key", "test-value"),
				ghttp.RespondWith(http.StatusOK, "test-fixture"),
			))

			a, err := dependencyCache.Artifact(dependency, func(request *http.Request) (*http.Request, error) {
				request.Header.Add("Test-Key", "test-value")
				return request, nil
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(io.ReadAll(a)).To(Equal([]byte("test-fixture")))
		})

		it("hide uri credentials from log", func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.RespondWith(http.StatusOK, "test-fixture"),
			))

			url, err := url.ParseRequestURI(dependency.URI)
			Expect(err).NotTo(HaveOccurred())
			credentials := "username:password"
			uriWithBasicCreds := url.Scheme + "://" + credentials + "@" + url.Hostname() + ":" + url.Port() + url.Path
			dependency.URI = uriWithBasicCreds

			var logBuffer bytes.Buffer
			dependencyCache.Logger = bard.NewLogger(&logBuffer)

			// Make sure the password is not part of the log output.
			a, errA := dependencyCache.Artifact(dependency)
			Expect(errA).NotTo(HaveOccurred())
			Expect(a).NotTo(BeNil())
			Expect(logBuffer.String()).NotTo(ContainSubstring("password"))
			logBuffer.Reset()

			// Make sure the password is not part of the log output when an error occurs.
			dependency.URI = "foo://username:password@acme.com"
			b, errB := dependencyCache.Artifact(dependency)
			Expect(errB).To(HaveOccurred())
			Expect(b).To(BeNil())
			Expect(logBuffer.String()).NotTo(ContainSubstring("password"))
		})
	})
}
