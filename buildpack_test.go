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
	"testing"

	. "github.com/onsi/gomega"
	"github.com/paketoio/libpak"
	"github.com/sclevine/spec"
)

func testBuildpack(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	context("NewBuildpackMetadata", func() {
		it("deserializes metadata", func() {
			actual := map[string]interface{}{
				"default-versions": map[string]interface{}{
					"test-key": "test-value",
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
				DefaultVersions: map[string]string{
					"test-key": "test-value",
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

		context("Any", func() {

			it("indicates that dependency exists", func() {
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

				Expect(resolver.Any("test-id", "")).To(BeTrue())
			})

			it("indicates that dependency does not exist", func() {
				Expect(resolver.Any("test-id", "")).To(BeFalse())
			})
		})

	})
}
