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

package bindings_test

import (
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak/bindings"
)

func testResolve(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		binds libcnb.Bindings
	)

	it.Before(func() {
		binds = []libcnb.Binding{
			{
				Name:     "name1",
				Type:     "some-type",
				Provider: "some-provider",
			},
			{
				Name:     "name2",
				Type:     "some-type",
				Provider: "other-provider",
			},
			{
				Name:     "name3",
				Type:     "other-type",
				Provider: "some-provider",
			},
			{
				Name:     "name1",
				Type:     "unknown",
				Provider: "unknown",
			},
		}
	})

	context("Resolve", func() {
		context("no predicate", func() {
			it("returns all bindings", func() {
				resolved := bindings.Resolve(binds)
				Expect(resolved).To(Equal(binds))
			})
		})

		context("ByType", func() {
			it("returns all with matching type", func() {
				resolved := bindings.Resolve(binds,
					bindings.OfType("some-type"),
				)
				Expect(resolved).To(Equal(libcnb.Bindings{
					{
						Name:     "name1",
						Type:     "some-type",
						Provider: "some-provider",
					},
					{
						Name:     "name2",
						Type:     "some-type",
						Provider: "other-provider",
					},
				}))
			})
		})

		context("ByKeyWithValueFromSecret", func() {
			it("returns all with matching type", func() {
				b := []libcnb.Binding{
					{
						Name:     "name1",
						Type:     "some-type",
						Provider: "some-provider",
						Secret: map[string]string{
							"type": "other-type",
							"user": "some-user",
							"pass": "some-pass",
						},
					},
					{
						Name:     "name2",
						Type:     "some-type",
						Provider: "other-provider",
						Secret: map[string]string{
							"type": "other-type",
							"user": "some-user",
							"pass": "some-pass",
						},
					},
					{
						Name:     "name3",
						Type:     "other-type",
						Provider: "some-provider",
						Secret: map[string]string{
							"type": "foo",
							"user": "some-user",
							"pass": "some-pass",
						},
					},
					{
						Name:     "name1",
						Type:     "unknown",
						Provider: "unknown",
						Secret: map[string]string{
							"type": "some-type",
							"user": "some-user",
							"pass": "some-pass",
						},
					},
				}

				resolved := bindings.Resolve(b,
					bindings.OfKeyWithValueFromSecret("type", "other-type"),
				)
				Expect(resolved).To(Equal(libcnb.Bindings{
					{
						Name:     "name1",
						Type:     "some-type",
						Provider: "some-provider",
						Secret: map[string]string{
							"type": "other-type",
							"user": "some-user",
							"pass": "some-pass",
						},
					},
					{
						Name:     "name2",
						Type:     "some-type",
						Provider: "other-provider",
						Secret: map[string]string{
							"type": "other-type",
							"user": "some-user",
							"pass": "some-pass",
						},
					},
				}))
			})
		})

		context("ByProvider", func() {
			it("returns all with matching type", func() {
				resolved := bindings.Resolve(binds,
					bindings.OfProvider("some-provider"),
				)
				Expect(resolved).To(Equal(libcnb.Bindings{
					{
						Name:     "name1",
						Type:     "some-type",
						Provider: "some-provider",
					},
					{
						Name:     "name3",
						Type:     "other-type",
						Provider: "some-provider",
					},
				}))
			})
		})

		context("WithName", func() {
			it("returns all with matching name", func() {
				resolved := bindings.Resolve(binds,
					bindings.WithName("Name1"),
				)
				Expect(resolved).To(Equal(libcnb.Bindings{
					{
						Name:     "name1",
						Type:     "some-type",
						Provider: "some-provider",
					},
					{
						Name:     "name1",
						Type:     "unknown",
						Provider: "unknown",
					},
				}))
			})
		})

		context("multiple predicates", func() {
			it("returns the intersection", func() {
				resolved := bindings.Resolve(binds,
					bindings.OfType("some-type"),
					bindings.OfProvider("some-provider"),
				)
				Expect(resolved).To(Equal(libcnb.Bindings{
					{
						Name:     "name1",
						Type:     "some-type",
						Provider: "some-provider",
					},
				}))
			})
		})

		context("zero bindings match", func() {
			it("returns nil", func() {
				resolved := bindings.Resolve(binds,
					bindings.OfType("missing-type"),
				)
				Expect(resolved).To(BeNil())
			})
		})
	})

	context("ResolveOne", func() {
		context("one binding matches", func() {
			it("returns the binding and true", func() {
				bind, ok, err := bindings.ResolveOne(binds,
					bindings.OfType("some-type"),
					bindings.OfProvider("some-provider"),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(ok).To(BeTrue())
				Expect(bind).To(Equal(libcnb.Binding{
					Name:     "name1",
					Type:     "some-type",
					Provider: "some-provider",
				}))
			})
		})

		context("multiples match", func() {
			it("returns an error", func() {
				_, _, err := bindings.ResolveOne(binds,
					bindings.OfType("some-type"),
				)
				Expect(err).To(MatchError(`multiple bindings matched the given predicates [name1 name2]`))
			})
		})

		context("zero bindings match", func() {
			it("returns the binding and false", func() {
				_, ok, err := bindings.ResolveOne(binds,
					bindings.OfType("missing-type"),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
	})
}
