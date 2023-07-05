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

package libpak_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/internal"
	"github.com/paketo-buildpacks/libpak/sbom"
)

func testBuildpack(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	it("is equal after toml Marshal and Unmarshal", func() {
		dependency := libpak.BuildModuleDependency{
			ID:              "test-id",
			Name:            "test-name",
			Version:         "1.1.1",
			URI:             "test-uri",
			SHA256:          "test-sha256",
			DeprecationDate: time.Now(),
			Stacks:          []string{"test-stack"},
			Licenses: []libpak.BuildModuleDependencyLicense{
				{
					Type: "test-type",
					URI:  "test-uri",
				},
			},
		}

		bytes, err := internal.Marshal(dependency)
		Expect(err).NotTo(HaveOccurred())

		var newDependency libpak.BuildModuleDependency
		err = toml.Unmarshal(bytes, &newDependency)
		Expect(err).NotTo(HaveOccurred())

		Expect(dependency.Equals(newDependency)).To(BeTrue())
	})

	it("renders dependency as a SyftArtifact", func() {
		dependency := libpak.BuildModuleDependency{
			ID:      "test-id",
			Name:    "test-name",
			Version: "1.1.1",
			URI:     "test-uri",
			SHA256:  "test-sha256",
			Stacks:  []string{"test-stack"},
			Licenses: []libpak.BuildModuleDependencyLicense{
				{
					Type: "test-type",
					URI:  "test-uri",
				},
			},
			CPEs: []string{"test-cpe1", "test-cpe2"},
			PURL: "test-purl",
		}

		Expect(dependency.AsSyftArtifact("buildpack.toml")).To(Equal(sbom.SyftArtifact{
			ID:        "46713835f08d90b7",
			Name:      "test-name",
			Version:   "1.1.1",
			Type:      "UnknownPackage",
			FoundBy:   "libpak",
			Licenses:  []string{"test-type"},
			Locations: []sbom.SyftLocation{{Path: "buildpack.toml"}},
			CPEs:      []string{"test-cpe1", "test-cpe2"},
			PURL:      "test-purl",
		}))
	})

	it("renders extension dependency as a SyftArtifact", func() {
		dependency := libpak.BuildModuleDependency{
			ID:      "test-id",
			Name:    "test-name",
			Version: "1.1.1",
			URI:     "test-uri",
			SHA256:  "test-sha256",
			Stacks:  []string{"test-stack"},
			Licenses: []libpak.BuildModuleDependencyLicense{
				{
					Type: "test-type",
					URI:  "test-uri",
				},
			},
			CPEs: []string{"test-cpe1", "test-cpe2"},
			PURL: "test-purl",
		}

		Expect(dependency.AsSyftArtifact("extension.toml")).To(Equal(sbom.SyftArtifact{
			ID:        "9a52b9f58469d126",
			Name:      "test-name",
			Version:   "1.1.1",
			Type:      "UnknownPackage",
			FoundBy:   "libpak",
			Licenses:  []string{"test-type"},
			Locations: []sbom.SyftLocation{{Path: "extension.toml"}},
			CPEs:      []string{"test-cpe1", "test-cpe2"},
			PURL:      "test-purl",
		}))
	})

	it("calculates dependency deprecation", func() {
		deprecatedDependency := libpak.BuildModuleDependency{
			ID:              "test-id",
			DeprecationDate: time.Now().UTC(),
		}

		soonDeprecatedDependency := libpak.BuildModuleDependency{
			ID:              "test-id",
			DeprecationDate: time.Now().UTC().Add(30 * 24 * time.Hour),
		}

		Expect(deprecatedDependency.IsDeprecated()).To(BeTrue())
		Expect(deprecatedDependency.IsSoonDeprecated()).To(BeFalse())
		Expect(soonDeprecatedDependency.IsDeprecated()).To(BeFalse())
		Expect(soonDeprecatedDependency.IsSoonDeprecated()).To(BeTrue())
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
						"cpes":             []interface{}{"cpe:2.3:a:test-id:1.1.1"},
						"purl":             "pkg:generic:test-id@1.1.1",
						"deprecation_date": "2021-12-31T15:59:00-08:00",
					},
				},
				"include-files": []interface{}{"test-include-file"},
				"pre-package":   "test-pre-package",
			}

			deprecationDate, err := time.Parse(time.RFC3339, "2021-12-31T15:59:00-08:00")
			Expect(err).ToNot(HaveOccurred())

			expected := libpak.BuildModuleMetadata{
				Configurations: []libpak.BuildModuleConfiguration{
					{
						Name:        "test-name",
						Default:     "test-default",
						Description: "test-description",
					},
				},
				Dependencies: []libpak.BuildModuleDependency{
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: "1.1.1",
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack"},
						Licenses: []libpak.BuildModuleDependencyLicense{
							{
								Type: "test-type",
								URI:  "test-uri",
							},
						},
						CPEs:            []string{"cpe:2.3:a:test-id:1.1.1"},
						PURL:            "pkg:generic:test-id@1.1.1",
						DeprecationDate: deprecationDate,
					},
				},
				IncludeFiles: []string{"test-include-file"},
				PrePackage:   "test-pre-package",
			}

			Expect(libpak.NewBuildModuleMetadata(actual)).To(Equal(expected))
		})
	})

	context("ConfigurationResolver", func() {
		var (
			resolver = libpak.ConfigurationResolver{
				Configurations: []libpak.BuildModuleConfiguration{
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
				resolver.Dependencies = []libpak.BuildModuleDependency{
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

				Expect(resolver.Resolve("test-id-2", "1.0")).To(Equal(libpak.BuildModuleDependency{
					ID:      "test-id-2",
					Name:    "test-name",
					Version: "1.0",
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"test-stack-1", "test-stack-2"},
				}))
			})

			it("filters by version constraint", func() {
				resolver.Dependencies = []libpak.BuildModuleDependency{
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

				Expect(resolver.Resolve("test-id", "2.0")).To(Equal(libpak.BuildModuleDependency{
					ID:      "test-id",
					Name:    "test-name",
					Version: "2.0",
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"test-stack-1", "test-stack-2"},
				}))
			})

			it("filters by stack", func() {
				resolver.Dependencies = []libpak.BuildModuleDependency{
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

				Expect(resolver.Resolve("test-id", "1.0")).To(Equal(libpak.BuildModuleDependency{
					ID:      "test-id",
					Name:    "test-name",
					Version: "1.0",
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"test-stack-1", "test-stack-3"},
				}))
			})

			it("filters by stack and supports the wildcard stack", func() {
				resolver.Dependencies = []libpak.BuildModuleDependency{
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

				Expect(resolver.Resolve("test-id", "1.0")).To(Equal(libpak.BuildModuleDependency{
					ID:      "test-id",
					Name:    "test-name",
					Version: "1.0",
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"*"},
				}))
			})

			it("filters by stack and treats no stacks as the wildcard stack", func() {
				resolver.Dependencies = []libpak.BuildModuleDependency{
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

				Expect(resolver.Resolve("test-id", "1.0")).To(Equal(libpak.BuildModuleDependency{
					ID:      "test-id",
					Name:    "test-name",
					Version: "1.0",
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{},
				}))
			})

			it("returns the best dependency", func() {
				resolver.Dependencies = []libpak.BuildModuleDependency{
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

				Expect(resolver.Resolve("test-id", "1.*")).To(Equal(libpak.BuildModuleDependency{
					ID:      "test-id",
					Name:    "test-name",
					Version: "1.1",
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"test-stack-1", "test-stack-2"},
				}))
			})

			it("returns the best dependency after filtering", func() {
				resolver.Dependencies = []libpak.BuildModuleDependency{
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

				Expect(resolver.Resolve("test-id-2", "")).To(Equal(libpak.BuildModuleDependency{
					ID:      "test-id-2",
					Name:    "test-name-2",
					Version: "1.9.0",
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"test-stack-2"},
				}))
			})

			it("returns error if there are no matching dependencies", func() {
				resolver.Dependencies = []libpak.BuildModuleDependency{
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
				resolver.Dependencies = []libpak.BuildModuleDependency{
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

				Expect(resolver.Resolve("test-id", "")).To(Equal(libpak.BuildModuleDependency{
					ID:      "test-id",
					Name:    "test-name",
					Version: "1.1",
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"test-stack-1", "test-stack-2"},
				}))
			})

			it("prints outdated dependencies", func() {
				buff := bytes.NewBuffer(nil)
				logger := bard.NewLogger(buff)
				resolver.Logger = &logger
				soonDeprecated := time.Now().UTC().Add(30 * 24 * time.Hour)
				notSoSoonDeprecated := time.Now().UTC().Add(60 * 24 * time.Hour)
				resolver.Dependencies = []libpak.BuildModuleDependency{
					{
						ID:      "missing-deprecation-date",
						Name:    "missing-deprecation-date",
						Version: "1.1",
					},
					{
						ID:              "valid-dependency",
						Name:            "valid-dependency",
						Version:         "1.1",
						DeprecationDate: notSoSoonDeprecated,
					},
					{
						ID:              "soon-deprecated-dependency",
						Name:            "soon-deprecated-dependency",
						Version:         "1.1",
						DeprecationDate: soonDeprecated,
					},
					{
						ID:              "deprecated-dependency",
						Name:            "deprecated-dependency",
						Version:         "1.1",
						DeprecationDate: time.Now().UTC(),
					},
				}

				for _, dependency := range resolver.Dependencies {
					resolver.Resolve(dependency.ID, "")
				}

				Expect(buff.String()).To(Equal(fmt.Sprintf("  \x1b[33mDeprecation Notice:\x1b[0m\n\x1b[2m    \x1b[33mVersion 1.1 of soon-deprecated-dependency will be deprecated after %s.\x1b[0m\x1b[2m\x1b[0m\n\x1b[2m    \x1b[33mMigrate your application to a supported version of soon-deprecated-dependency before this time.\x1b[0m\x1b[2m\x1b[0m\n  \x1b[33mDeprecation Notice:\x1b[0m\n\x1b[2m    \x1b[33mVersion 1.1 of deprecated-dependency is deprecated.\x1b[0m\x1b[2m\x1b[0m\n\x1b[2m    \x1b[33mMigrate your application to a supported version of deprecated-dependency.\x1b[0m\x1b[2m\x1b[0m\n", soonDeprecated.Format("2006-01-02"))))
			})

		})

		it("indicates whether error is NoValidDependenciesError", func() {
			Expect(libpak.IsNoValidDependencies(nil)).To(BeFalse())
			Expect(libpak.IsNoValidDependencies(fmt.Errorf("test-error"))).To(BeFalse())
			Expect(libpak.IsNoValidDependencies(libpak.NoValidDependenciesError{})).To(BeTrue())
		})
	})
}
