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
	"os"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak"
)

func testBuildpack(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	it("renders dependency as a BOMEntry", func() {
		dependency := libpak.BuildpackDependency{
			ID:      "test-id",
			Name:    "test-name",
			Version: "1.1.1",
			URI:     "test-uri",
			SHA256:  "test-sha256",
			Stacks:  []string{"test-stack"},
			Licenses: []libpak.BuildpackDependencyLicense{
				{
					Type: "test-type",
					URI:  "test-uri",
				},
			},
		}

		Expect(dependency.AsBOMEntry()).To(Equal(libcnb.BOMEntry{
			Name: dependency.ID,
			Metadata: map[string]interface{}{
				"name":     dependency.Name,
				"version":  dependency.Version,
				"uri":      dependency.URI,
				"sha256":   dependency.SHA256,
				"stacks":   dependency.Stacks,
				"licenses": dependency.Licenses,
			},
		}))
	})

	context("NewBuildpackMetadata", func() {
		it("deserializes metadata", func() {
			actual := map[string]interface{}{
				"configurations": []map[string]interface{}{
					{
						"name":        "test-name",
						"default":     "test-default",
						"description": "test-description",
					},
				},
				"dependencies": []map[string]interface{}{
					{
						"id":      "test-id",
						"name":    "test-name",
						"version": "1.1.1",
						"uri":     "test-uri",
						"sha256":  "test-sha256",
						"stacks":  []interface{}{"test-stack"},
						"licenses": []map[string]interface{}{
							{
								"type": "test-type",
								"uri":  "test-uri",
							},
						},
					},
				},
				"include-files": []interface{}{"test-include-file"},
				"pre-package":   "test-pre-package",
			}

			expected := libpak.BuildpackMetadata{
				Configurations: []libpak.BuildpackConfiguration{
					{
						Name:        "test-name",
						Default:     "test-default",
						Description: "test-description",
					},
				},
				Dependencies: []libpak.BuildpackDependency{
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: "1.1.1",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack"},
						Licenses: []libpak.BuildpackDependencyLicense{
							{
								Type: "test-type",
								URI:  "test-uri",
							},
						},
					},
				},
				IncludeFiles: []string{"test-include-file"},
				PrePackage:   "test-pre-package",
			}

			Expect(libpak.NewBuildpackMetadata(actual)).To(Equal(expected))
		})
	})

	context("ConfigurationResolver", func() {
		var (
			resolver = libpak.ConfigurationResolver{
				Configurations: []libpak.BuildpackConfiguration{
					{Name: "TEST_KEY_1", Default: "test-default-value-1"},
					{Name: "TEST_KEY_2", Default: "test-default-value-2"},
					{Name: "TEST_BOOL_3", Default: "true"},
					{Name: "TEST_BOOL_4", Default: "false"},
					{Name: "TEST_BOOL_6", Default: "test-value"},
				},
			}
		)

		it.Before(func() {
			Expect(os.Setenv("TEST_KEY_1", "test-value-1")).To(Succeed())
			Expect(os.Setenv("TEST_BOOL_1", "true")).To(Succeed())
			Expect(os.Setenv("TEST_BOOL_2", "false")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("TEST_KEY_1")).To(Succeed())
			Expect(os.Unsetenv("TEST_BOOL_1")).To(Succeed())
			Expect(os.Unsetenv("TEST_BOOL_2")).To(Succeed())
		})

		it("returns configured value", func() {
			v, ok := resolver.Resolve("TEST_KEY_1")
			Expect(v).To(Equal("test-value-1"))
			Expect(ok).To(BeTrue())
		})

		it("returns default value", func() {
			v, ok := resolver.Resolve("TEST_KEY_2")
			Expect(v).To(Equal("test-default-value-2"))
			Expect(ok).To(BeFalse())
		})

		it("returns unknown value", func() {
			v, ok := resolver.Resolve("TEST_KEY_3")
			Expect(v).To(Equal(""))
			Expect(ok).To(BeFalse())
		})

		it("returns configured bool", func() {
			Expect(resolver.ResolveBool("TEST_BOOL_1")).To(BeTrue())
			Expect(resolver.ResolveBool("TEST_BOOL_2")).To(BeFalse())
		})

		it("returns default bool", func() {
			Expect(resolver.ResolveBool("TEST_BOOL_3")).To(BeTrue())
			Expect(resolver.ResolveBool("TEST_BOOL_4")).To(BeFalse())
		})

		it("returns false for unset", func() {
			Expect(resolver.ResolveBool("TEST_BOOL_5")).To(BeFalse())
		})

		it("return false for invalid", func() {
			Expect(resolver.ResolveBool("TEST_BOOL_6")).To(BeFalse())
		})
	})

	context("DependencyResolver", func() {
		var (
			resolver libpak.DependencyResolver
		)

		context("Resolve", func() {

			it("filters by id", func() {
				resolver.Dependencies = []libpak.BuildpackDependency{
					{
						ID:      "test-id-1",
						Name:    "test-name",
						Version: "1.0",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"},
					},
					{
						ID:      "test-id-2",
						Name:    "test-name",
						Version: "1.0",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"},
					},
				}
				resolver.StackID = "test-stack-1"

				Expect(resolver.Resolve("test-id-2", "1.0")).To(Equal(libpak.BuildpackDependency{
					ID:      "test-id-2",
					Name:    "test-name",
					Version: "1.0",
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"test-stack-1", "test-stack-2"},
				}))
			})

			it("filters by version constraint", func() {
				resolver.Dependencies = []libpak.BuildpackDependency{
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: "1.0",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"},
					},
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: "2.0",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"},
					},
				}
				resolver.StackID = "test-stack-1"

				Expect(resolver.Resolve("test-id", "2.0")).To(Equal(libpak.BuildpackDependency{
					ID:      "test-id",
					Name:    "test-name",
					Version: "2.0",
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"test-stack-1", "test-stack-2"},
				}))
			})

			it("filters by stack", func() {
				resolver.Dependencies = []libpak.BuildpackDependency{
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: "1.0",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"},
					},
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: "1.0",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-3"},
					},
				}
				resolver.StackID = "test-stack-3"

				Expect(resolver.Resolve("test-id", "1.0")).To(Equal(libpak.BuildpackDependency{
					ID:      "test-id",
					Name:    "test-name",
					Version: "1.0",
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"test-stack-1", "test-stack-3"},
				}))
			})

			it("filters by stack and supports the wildcard stack", func() {
				resolver.Dependencies = []libpak.BuildpackDependency{
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: "1.0",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"},
					},
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: "1.0",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"*"},
					},
				}
				resolver.StackID = "test-stack-3"

				Expect(resolver.Resolve("test-id", "1.0")).To(Equal(libpak.BuildpackDependency{
					ID:      "test-id",
					Name:    "test-name",
					Version: "1.0",
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"*"},
				}))
			})

			it("filters by stack and treats no stacks as the wildcard stack", func() {
				resolver.Dependencies = []libpak.BuildpackDependency{
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: "1.0",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"},
					},
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: "1.0",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{},
					},
				}
				resolver.StackID = "test-stack-3"

				Expect(resolver.Resolve("test-id", "1.0")).To(Equal(libpak.BuildpackDependency{
					ID:      "test-id",
					Name:    "test-name",
					Version: "1.0",
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{},
				}))
			})

			it("returns the best dependency", func() {
				resolver.Dependencies = []libpak.BuildpackDependency{
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: "1.1",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"},
					},
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: "1.0",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-3"},
					},
				}
				resolver.StackID = "test-stack-1"

				Expect(resolver.Resolve("test-id", "1.*")).To(Equal(libpak.BuildpackDependency{
					ID:      "test-id",
					Name:    "test-name",
					Version: "1.1",
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"test-stack-1", "test-stack-2"},
				}))
			})

			it("returns the best dependency after filtering", func() {
				resolver.Dependencies = []libpak.BuildpackDependency{
					{
						ID:      "test-id-1",
						Name:    "test-name-1",
						Version: "1.9.1",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1"},
					},
					{
						ID:      "test-id-1",
						Name:    "test-name-1",
						Version: "1.9.1",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-2"},
					},
					{
						ID:      "test-id-2",
						Name:    "test-name-2",
						Version: "1.8.5",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-2"},
					},
					{
						ID:      "test-id-2",
						Name:    "test-name-2",
						Version: "1.8.6",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1"},
					},
					{
						ID:      "test-id-2",
						Name:    "test-name-2",
						Version: "1.8.6",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-2"},
					},
					{
						ID:      "test-id-2",
						Name:    "test-name-2",
						Version: "1.9.0",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1"},
					},
					{
						ID:      "test-id-2",
						Name:    "test-name-2",
						Version: "1.9.0",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-2"},
					},
				}
				resolver.StackID = "test-stack-2"

				Expect(resolver.Resolve("test-id-2", "")).To(Equal(libpak.BuildpackDependency{
					ID:      "test-id-2",
					Name:    "test-name-2",
					Version: "1.9.0",
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"test-stack-2"},
				}))
			})

			it("returns error if there are no matching dependencies", func() {
				resolver.Dependencies = []libpak.BuildpackDependency{
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: "1.0",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"},
					},
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: "1.0",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-3"},
					},
					{
						ID:      "test-id-2",
						Name:    "test-name",
						Version: "1.1",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-3"},
					},
				}
				resolver.StackID = "test-stack-1"

				_, err := resolver.Resolve("test-id-2", "1.0")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(libpak.NoValidDependenciesError{Message: "no valid dependencies for test-id-2, 1.0, and test-stack-1 in [(test-id, 1.0, [test-stack-1 test-stack-2]) (test-id, 1.0, [test-stack-1 test-stack-3]) (test-id-2, 1.1, [test-stack-1 test-stack-3])]"}))
			})

			it("substitutes all wildcard for unspecified version constraint", func() {
				resolver.Dependencies = []libpak.BuildpackDependency{
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: "1.1",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"},
					},
				}
				resolver.StackID = "test-stack-1"

				Expect(resolver.Resolve("test-id", "")).To(Equal(libpak.BuildpackDependency{
					ID:      "test-id",
					Name:    "test-name",
					Version: "1.1",
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"test-stack-1", "test-stack-2"},
				}))
			})
		})

		it("indicates whether error is NoValidDependenciesError", func() {
			Expect(libpak.IsNoValidDependencies(nil)).To(BeFalse())
			Expect(libpak.IsNoValidDependencies(fmt.Errorf("test-error"))).To(BeFalse())
			Expect(libpak.IsNoValidDependencies(libpak.NoValidDependenciesError{})).To(BeTrue())
		})
	})
}
