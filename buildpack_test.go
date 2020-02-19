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

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketoio/libpak"
	"github.com/sclevine/spec"
)

func testBuildpack(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	context("BuildpackVersion", func() {

		it("creates a new instance", func() {
			v := libpak.NewBuildpackVersion("1.0")
			Expect(v.Original()).To(Equal("1.0"))
		})

		it("panics when creating a new instance with an illegal version", func() {
			Expect(func() { libpak.NewBuildpackVersion("") }).Should(Panic())
		})

		it("marshals", func() {
			v := libpak.NewBuildpackVersion("1.0")
			Expect(v.MarshalText()).To(Equal([]byte("1.0")))
		})

		it("unmarshals", func() {
			v := libpak.BuildpackVersion{}
			Expect(v.UnmarshalText([]byte("1.0"))).To(Succeed())
			Expect(v.Original()).To(Equal("1.0"))
		})

	})

	context("BuildpackDependency", func() {
		var (
			dependency libpak.BuildpackDependency
		)

		it.Before(func() {
			dependency = libpak.BuildpackDependency{
				ID:      "test-id",
				Name:    "test-name",
				Version: libpak.NewBuildpackVersion("1.1.1"),
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
		})

		it("returns as metadata", func() {
			Expect(dependency.Metadata()).To(Equal(map[string]interface{}{
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
			}))
		})

		it("returns as BuildpackPlan", func() {
			Expect(dependency.PlanEntry()).To(Equal(libcnb.BuildpackPlanEntry{
				Name:    "test-id",
				Version: "1.1.1",
				Metadata: map[string]interface{}{
					"name":   "test-name",
					"uri":    "test-uri",
					"sha256": "test-sha256",
					"stacks": []string{"test-stack"},
					"licenses": []libpak.BuildpackDependencyLicense{
						{
							Type: "test-type",
							URI:  "test-uri",
						},
					},
				},
			}))
		})
	})

	context("NewBuildpackMetadata", func() {
		it("deserializes empty metadata", func() {
			Expect(libpak.NewBuildpackMetadata(map[string]interface{}{})).To(Equal(libpak.BuildpackMetadata{
				DefaultVersions: map[string]string{},
			}))
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
						Version: libpak.NewBuildpackVersion("1.0"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"}},
					{
						ID:      "test-id-2",
						Name:    "test-name",
						Version: libpak.NewBuildpackVersion("1.0"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"}},
				}

				constraint := libpak.DependencyConstraint{ID: "test-id-2", Version: "1.0", StackID: "test-stack-1"}

				Expect(resolver.Resolve(constraint)).To(Equal(libpak.BuildpackDependency{
					ID:      "test-id-2",
					Name:    "test-name",
					Version: libpak.NewBuildpackVersion("1.0"),
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"test-stack-1", "test-stack-2"}}))
			})

			it("filters by version constraint", func() {
				resolver.Dependencies = []libpak.BuildpackDependency{
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: libpak.NewBuildpackVersion("1.0"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"}},
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: libpak.NewBuildpackVersion("2.0"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"}},
				}

				constraint := libpak.DependencyConstraint{ID: "test-id", Version: "2.0", StackID: "test-stack-1"}

				Expect(resolver.Resolve(constraint)).To(Equal(libpak.BuildpackDependency{
					ID:      "test-id",
					Name:    "test-name",
					Version: libpak.NewBuildpackVersion("2.0"),
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"test-stack-1", "test-stack-2"}}))
			})

			it("filters by stack", func() {
				resolver.Dependencies = []libpak.BuildpackDependency{
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: libpak.NewBuildpackVersion("1.0"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"}},
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: libpak.NewBuildpackVersion("1.0"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-3"}},
				}

				constraint := libpak.DependencyConstraint{ID: "test-id", Version: "1.0", StackID: "test-stack-3"}

				Expect(resolver.Resolve(constraint)).To(Equal(libpak.BuildpackDependency{
					ID:      "test-id",
					Name:    "test-name",
					Version: libpak.NewBuildpackVersion("1.0"),
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"test-stack-1", "test-stack-3"}}))
			})

			it("returns the best dependency", func() {
				resolver.Dependencies = []libpak.BuildpackDependency{
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: libpak.NewBuildpackVersion("1.1"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"}},
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: libpak.NewBuildpackVersion("1.0"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-3"}},
				}

				constraint := libpak.DependencyConstraint{ID: "test-id", Version: "1.*", StackID: "test-stack-1"}

				Expect(resolver.Resolve(constraint)).To(Equal(libpak.BuildpackDependency{
					ID:      "test-id",
					Name:    "test-name",
					Version: libpak.NewBuildpackVersion("1.1"),
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"test-stack-1", "test-stack-2"}}))
			})

			it("returns the best dependency after filtering", func() {
				resolver.Dependencies = []libpak.BuildpackDependency{
					{
						ID:      "test-id-1",
						Name:    "test-name-1",
						Version: libpak.NewBuildpackVersion("1.9.1"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1"},
					},
					{
						ID:      "test-id-1",
						Name:    "test-name-1",
						Version: libpak.NewBuildpackVersion("1.9.1"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-2"},
					},
					{
						ID:      "test-id-2",
						Name:    "test-name-2",
						Version: libpak.NewBuildpackVersion("1.8.5"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-2"},
					},
					{
						ID:      "test-id-2",
						Name:    "test-name-2",
						Version: libpak.NewBuildpackVersion("1.8.6"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1"},
					},
					{
						ID:      "test-id-2",
						Name:    "test-name-2",
						Version: libpak.NewBuildpackVersion("1.8.6"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-2"},
					},
					{
						ID:      "test-id-2",
						Name:    "test-name-2",
						Version: libpak.NewBuildpackVersion("1.9.0"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1"},
					},
					{
						ID:      "test-id-2",
						Name:    "test-name-2",
						Version: libpak.NewBuildpackVersion("1.9.0"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-2"},
					},
				}

				constraint := libpak.DependencyConstraint{ID: "test-id-2", StackID: "test-stack-2"}

				Expect(resolver.Resolve(constraint)).To(Equal(libpak.BuildpackDependency{
					ID:      "test-id-2",
					Name:    "test-name-2",
					Version: libpak.NewBuildpackVersion("1.9.0"),
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
						Version: libpak.NewBuildpackVersion("1.0"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"}},
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: libpak.NewBuildpackVersion("1.0"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-3"}},
					{
						ID:      "test-id-2",
						Name:    "test-name",
						Version: libpak.NewBuildpackVersion("1.1"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-3"}},
				}

				constraint := libpak.DependencyConstraint{ID: "test-id-2", Version: "1.0", StackID: "test-stack-1"}

				_, err := resolver.Resolve(constraint)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("no valid dependencies for test-id-2, 1.0, and test-stack-1 in [(test-id, 1.0, [test-stack-1 test-stack-2]) (test-id, 1.0, [test-stack-1 test-stack-3]) (test-id-2, 1.1, [test-stack-1 test-stack-3])]"))
			})

			it("substitutes all wildcard for unspecified version constraint", func() {
				resolver.Dependencies = []libpak.BuildpackDependency{
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: libpak.NewBuildpackVersion("1.1"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"}},
				}

				constraint := libpak.DependencyConstraint{ID: "test-id", StackID: "test-stack-1"}

				Expect(resolver.Resolve(constraint)).To(Equal(libpak.BuildpackDependency{
					ID:      "test-id",
					Name:    "test-name",
					Version: libpak.NewBuildpackVersion("1.1"),
					URI:     "test-uri",
					SHA256:  "test-sha256",
					Stacks:  []string{"test-stack-1", "test-stack-2"}}))
			})
		})

		context("Any", func() {

			it("indicates that dependency exists", func() {
				resolver.Dependencies = []libpak.BuildpackDependency{
					{
						ID:      "test-id",
						Name:    "test-name",
						Version: libpak.NewBuildpackVersion("1.1"),
						URI:     "test-uri",
						SHA256:  "test-sha256",
						Stacks:  []string{"test-stack-1", "test-stack-2"}},
				}

				constraint := libpak.DependencyConstraint{ID: "test-id", StackID: "test-stack-1"}

				Expect(resolver.Any(constraint)).To(BeTrue())
			})

			it("indicates that dependency does not exist", func() {
				constraint := libpak.DependencyConstraint{ID: "test-id"}

				Expect(resolver.Any(constraint)).To(BeFalse())
			})
		})

	})
}
