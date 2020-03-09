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
	"github.com/packeto-buildpacks/libpak"
	"github.com/sclevine/spec"
)

func testBuildpackPlan(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	context("ShallowMerge", func() {

		it("merges with empty", func() {
			a := libcnb.BuildpackPlanEntry{}
			b := libcnb.BuildpackPlanEntry{Name: "test-name"}

			expected := libcnb.BuildpackPlanEntry{Name: "test-name"}

			Expect(libpak.ShallowMerge(a, b)).To(Equal(expected))
		})

		context("Version", func() {
			it("chooses neither", func() {
				a := libcnb.BuildpackPlanEntry{Name: "test-name"}
				b := libcnb.BuildpackPlanEntry{Name: "test-name"}

				expected := libcnb.BuildpackPlanEntry{Name: "test-name"}

				Expect(libpak.ShallowMerge(a, b)).To(Equal(expected))
			})

			it("chooses a", func() {
				a := libcnb.BuildpackPlanEntry{Name: "test-name", Version: "test-version"}
				b := libcnb.BuildpackPlanEntry{Name: "test-name"}

				expected := libcnb.BuildpackPlanEntry{Name: "test-name", Version: "test-version"}

				Expect(libpak.ShallowMerge(a, b)).To(Equal(expected))
			})

			it("chooses b", func() {
				a := libcnb.BuildpackPlanEntry{Name: "test-name"}
				b := libcnb.BuildpackPlanEntry{Name: "test-name", Version: "test-version"}

				expected := libcnb.BuildpackPlanEntry{Name: "test-name", Version: "test-version"}

				Expect(libpak.ShallowMerge(a, b)).To(Equal(expected))
			})

			it("combines a and b with comma", func() {
				a := libcnb.BuildpackPlanEntry{Name: "test-name", Version: "test-version-1"}
				b := libcnb.BuildpackPlanEntry{Name: "test-name", Version: "test-version-2"}

				expected := libcnb.BuildpackPlanEntry{Name: "test-name", Version: "test-version-1,test-version-2"}

				Expect(libpak.ShallowMerge(a, b)).To(Equal(expected))
			})
		})

		context("metadata", func() {
			it("keeps a keys", func() {
				a := libcnb.BuildpackPlanEntry{
					Name:     "test-name",
					Metadata: map[string]interface{}{"test-key-1": "test-value-1"},
				}
				b := libcnb.BuildpackPlanEntry{Name: "test-name"}

				expected := libcnb.BuildpackPlanEntry{
					Name:     "test-name",
					Metadata: map[string]interface{}{"test-key-1": "test-value-1"},
				}

				Expect(libpak.ShallowMerge(a, b)).To(Equal(expected))
			})

			it("keeps b keys", func() {
				a := libcnb.BuildpackPlanEntry{Name: "test-name"}
				b := libcnb.BuildpackPlanEntry{
					Name:     "test-name",
					Metadata: map[string]interface{}{"test-key-1": "test-value-1"},
				}

				expected := libcnb.BuildpackPlanEntry{
					Name:     "test-name",
					Metadata: map[string]interface{}{"test-key-1": "test-value-1"},
				}

				Expect(libpak.ShallowMerge(a, b)).To(Equal(expected))
			})

			it("combines a and b keys", func() {
				a := libcnb.BuildpackPlanEntry{
					Name:     "test-name",
					Metadata: map[string]interface{}{"test-key-1": "test-value-1"},
				}
				b := libcnb.BuildpackPlanEntry{
					Name:     "test-name",
					Metadata: map[string]interface{}{"test-key-2": "test-value-2"},
				}

				expected := libcnb.BuildpackPlanEntry{
					Name:     "test-name",
					Metadata: map[string]interface{}{"test-key-1": "test-value-1", "test-key-2": "test-value-2"},
				}

				Expect(libpak.ShallowMerge(a, b)).To(Equal(expected))
			})

			it("overwrites a keys with b keys", func() {
				a := libcnb.BuildpackPlanEntry{
					Name:     "test-name",
					Metadata: map[string]interface{}{"test-key-1": "test-value-1"},
				}
				b := libcnb.BuildpackPlanEntry{
					Name:     "test-name",
					Metadata: map[string]interface{}{"test-key-1": "test-value-2"},
				}

				expected := libcnb.BuildpackPlanEntry{
					Name:     "test-name",
					Metadata: map[string]interface{}{"test-key-1": "test-value-2"},
				}

				Expect(libpak.ShallowMerge(a, b)).To(Equal(expected))
			})
		})

	})

	context("PlanEntryResolver", func() {

		context("ResolveWithMerge", func() {
			var (
				resolver = libpak.PlanEntryResolver{}
			)

			it.Before(func() {
				resolver.Plan = libcnb.BuildpackPlan{
					Entries: []libcnb.BuildpackPlanEntry{
						{
							Name: "test-name-1",
						},
						{
							Name:    "test-name-2",
							Version: "test-version-2a",
						},
						{
							Name:    "test-name-2",
							Version: "test-version-2b",
						},
					},
				}
			})

			var f = func(a, b libcnb.BuildpackPlanEntry) (libcnb.BuildpackPlanEntry, error) {
				return b, nil
			}

			it("returns error with no matches", func() {
				_, ok, err := resolver.ResolveWithMerge("test-name-0", f)
				Expect(err).NotTo(HaveOccurred())
				Expect(ok).To(BeFalse())
			})

			it("returns merged with single match", func() {
				e, ok, err := resolver.ResolveWithMerge("test-name-1", f)
				Expect(err).NotTo(HaveOccurred())
				Expect(ok).To(BeTrue())
				Expect(e).To(Equal(libcnb.BuildpackPlanEntry{
					Name: "test-name-1",
				}))
			})

			it("returns merged with multiple matches", func() {
				e, ok, err := resolver.ResolveWithMerge("test-name-2", f)
				Expect(err).NotTo(HaveOccurred())
				Expect(ok).To(BeTrue())
				Expect(e).To(Equal(libcnb.BuildpackPlanEntry{

					Name:    "test-name-2",
					Version: "test-version-2b",
				}))
			})
		})

		context("Resolve", func() {

			it("merges with empty", func() {
				a := libcnb.BuildpackPlanEntry{}
				b := libcnb.BuildpackPlanEntry{Name: "test-name"}

				resolver := libpak.PlanEntryResolver{
					Plan: libcnb.BuildpackPlan{Entries: []libcnb.BuildpackPlanEntry{a, b}},
				}
				expected := libcnb.BuildpackPlanEntry{Name: "test-name"}

				e, ok, err := resolver.Resolve("test-name")
				Expect(err).NotTo(HaveOccurred())
				Expect(ok).To(BeTrue())
				Expect(e).To(Equal(expected))
			})

			context("Version", func() {
				it("chooses neither", func() {
					a := libcnb.BuildpackPlanEntry{Name: "test-name"}
					b := libcnb.BuildpackPlanEntry{Name: "test-name"}

					resolver := libpak.PlanEntryResolver{
						Plan: libcnb.BuildpackPlan{Entries: []libcnb.BuildpackPlanEntry{a, b}},
					}
					expected := libcnb.BuildpackPlanEntry{Name: "test-name"}

					e, ok, err := resolver.Resolve("test-name")
					Expect(err).NotTo(HaveOccurred())
					Expect(ok).To(BeTrue())
					Expect(e).To(Equal(expected))
				})

				it("chooses a", func() {
					a := libcnb.BuildpackPlanEntry{Name: "test-name", Version: "test-version"}
					b := libcnb.BuildpackPlanEntry{Name: "test-name"}

					resolver := libpak.PlanEntryResolver{
						Plan: libcnb.BuildpackPlan{Entries: []libcnb.BuildpackPlanEntry{a, b}},
					}
					expected := libcnb.BuildpackPlanEntry{Name: "test-name", Version: "test-version"}

					e, ok, err := resolver.Resolve("test-name")
					Expect(err).NotTo(HaveOccurred())
					Expect(ok).To(BeTrue())
					Expect(e).To(Equal(expected))
				})

				it("chooses b", func() {
					a := libcnb.BuildpackPlanEntry{Name: "test-name"}
					b := libcnb.BuildpackPlanEntry{Name: "test-name", Version: "test-version"}

					resolver := libpak.PlanEntryResolver{
						Plan: libcnb.BuildpackPlan{Entries: []libcnb.BuildpackPlanEntry{a, b}},
					}
					expected := libcnb.BuildpackPlanEntry{Name: "test-name", Version: "test-version"}

					e, ok, err := resolver.Resolve("test-name")
					Expect(err).NotTo(HaveOccurred())
					Expect(ok).To(BeTrue())
					Expect(e).To(Equal(expected))
				})

				it("combines a and b with comma", func() {
					a := libcnb.BuildpackPlanEntry{Name: "test-name", Version: "test-version-1"}
					b := libcnb.BuildpackPlanEntry{Name: "test-name", Version: "test-version-2"}

					resolver := libpak.PlanEntryResolver{
						Plan: libcnb.BuildpackPlan{Entries: []libcnb.BuildpackPlanEntry{a, b}},
					}
					expected := libcnb.BuildpackPlanEntry{Name: "test-name", Version: "test-version-1,test-version-2"}

					e, ok, err := resolver.Resolve("test-name")
					Expect(err).NotTo(HaveOccurred())
					Expect(ok).To(BeTrue())
					Expect(e).To(Equal(expected))
				})
			})

			context("metadata", func() {
				it("keeps a keys", func() {
					a := libcnb.BuildpackPlanEntry{
						Name:     "test-name",
						Metadata: map[string]interface{}{"test-key-1": "test-value-1"},
					}
					b := libcnb.BuildpackPlanEntry{Name: "test-name"}

					resolver := libpak.PlanEntryResolver{
						Plan: libcnb.BuildpackPlan{Entries: []libcnb.BuildpackPlanEntry{a, b}},
					}
					expected := libcnb.BuildpackPlanEntry{
						Name:     "test-name",
						Metadata: map[string]interface{}{"test-key-1": "test-value-1"},
					}

					e, ok, err := resolver.Resolve("test-name")
					Expect(err).NotTo(HaveOccurred())
					Expect(ok).To(BeTrue())
					Expect(e).To(Equal(expected))
				})

				it("keeps b keys", func() {
					a := libcnb.BuildpackPlanEntry{Name: "test-name"}
					b := libcnb.BuildpackPlanEntry{
						Name:     "test-name",
						Metadata: map[string]interface{}{"test-key-1": "test-value-1"},
					}

					resolver := libpak.PlanEntryResolver{
						Plan: libcnb.BuildpackPlan{Entries: []libcnb.BuildpackPlanEntry{a, b}},
					}
					expected := libcnb.BuildpackPlanEntry{
						Name:     "test-name",
						Metadata: map[string]interface{}{"test-key-1": "test-value-1"},
					}

					e, ok, err := resolver.Resolve("test-name")
					Expect(err).NotTo(HaveOccurred())
					Expect(ok).To(BeTrue())
					Expect(e).To(Equal(expected))
				})

				it("combines a and b keys", func() {
					a := libcnb.BuildpackPlanEntry{
						Name:     "test-name",
						Metadata: map[string]interface{}{"test-key-1": "test-value-1"},
					}
					b := libcnb.BuildpackPlanEntry{
						Name:     "test-name",
						Metadata: map[string]interface{}{"test-key-2": "test-value-2"},
					}

					resolver := libpak.PlanEntryResolver{
						Plan: libcnb.BuildpackPlan{Entries: []libcnb.BuildpackPlanEntry{a, b}},
					}
					expected := libcnb.BuildpackPlanEntry{
						Name:     "test-name",
						Metadata: map[string]interface{}{"test-key-1": "test-value-1", "test-key-2": "test-value-2"},
					}

					e, ok, err := resolver.Resolve("test-name")
					Expect(err).NotTo(HaveOccurred())
					Expect(ok).To(BeTrue())
					Expect(e).To(Equal(expected))
				})

				it("overwrites a keys with b keys", func() {
					a := libcnb.BuildpackPlanEntry{
						Name:     "test-name",
						Metadata: map[string]interface{}{"test-key-1": "test-value-1"},
					}
					b := libcnb.BuildpackPlanEntry{
						Name:     "test-name",
						Metadata: map[string]interface{}{"test-key-1": "test-value-2"},
					}

					resolver := libpak.PlanEntryResolver{
						Plan: libcnb.BuildpackPlan{Entries: []libcnb.BuildpackPlanEntry{a, b}},
					}
					expected := libcnb.BuildpackPlanEntry{
						Name:     "test-name",
						Metadata: map[string]interface{}{"test-key-1": "test-value-2"},
					}

					e, ok, err := resolver.Resolve("test-name")
					Expect(err).NotTo(HaveOccurred())
					Expect(ok).To(BeTrue())
					Expect(e).To(Equal(expected))
				})
			})

		})

	})

}
