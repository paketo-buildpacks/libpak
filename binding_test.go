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
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketoio/libpak"
	"github.com/sclevine/spec"
)

func testBinding(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		resolver libpak.BindingResolver
	)

	it.Before(func() {
		resolver.Bindings = libcnb.Bindings{}
	})

	it("returns error if binding does not exist", func() {
		_, ok, err := resolver.Resolve(libpak.BindingConstraint{})
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
	})

	it("returns error if multiple bindings exist", func() {
		resolver.Bindings["test-binding-1"] = libcnb.NewBinding()
		resolver.Bindings["test-binding-2"] = libcnb.NewBinding()

		_, _, err := resolver.Resolve(libpak.BindingConstraint{})
		Expect(err).To(MatchError(fmt.Errorf("multiple bindings found for %+v in %+v",
			libpak.BindingConstraint{}, resolver.Bindings)))
	})

	it("filters on name", func() {
		resolver.Bindings["test-binding-1"] = libcnb.NewBinding()
		resolver.Bindings["test-binding-2"] = libcnb.NewBinding()
		resolver.Bindings["test-binding-2"].Metadata["test-key"] = "test-value"

		expected := libcnb.NewBinding()
		expected.Metadata["test-key"] = "test-value"

		b, ok, err := resolver.Resolve(libpak.BindingConstraint{Name: "test-binding-2"})
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		Expect(b).To(Equal(expected))
	})

	it("filters on kind", func() {
		resolver.Bindings["test-binding-1"] = libcnb.NewBinding()
		resolver.Bindings["test-binding-2"] = libcnb.NewBinding()
		resolver.Bindings["test-binding-2"].Metadata[libcnb.BindingKind] = "test-kind"
		resolver.Bindings["test-binding-2"].Metadata["test-key"] = "test-value"

		expected := libcnb.NewBinding()
		expected.Metadata[libcnb.BindingKind] = "test-kind"
		expected.Metadata["test-key"] = "test-value"

		b, ok, err := resolver.Resolve(libpak.BindingConstraint{Kind: "test-kind"})
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		Expect(b).To(Equal(expected))

	})

	it("filters on provider", func() {
		resolver.Bindings["test-binding-1"] = libcnb.NewBinding()
		resolver.Bindings["test-binding-2"] = libcnb.NewBinding()
		resolver.Bindings["test-binding-2"].Metadata[libcnb.BindingProvider] = "test-provider"
		resolver.Bindings["test-binding-2"].Metadata["test-key"] = "test-value"

		expected := libcnb.NewBinding()
		expected.Metadata[libcnb.BindingProvider] = "test-provider"
		expected.Metadata["test-key"] = "test-value"

		b, ok, err := resolver.Resolve(libpak.BindingConstraint{Provider: "test-provider"})
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		Expect(b).To(Equal(expected))

	})

	it("filters on all tags", func() {
		resolver.Bindings["test-binding-1"] = libcnb.NewBinding()
		resolver.Bindings["test-binding-2"] = libcnb.NewBinding()
		resolver.Bindings["test-binding-2"].Metadata[libcnb.BindingTags] = "test-tag-1\ntest-tag-2\ntest-tag-3"
		resolver.Bindings["test-binding-2"].Metadata["test-key"] = "test-value"

		expected := libcnb.NewBinding()
		expected.Metadata[libcnb.BindingTags] = "test-tag-1\ntest-tag-2\ntest-tag-3"
		expected.Metadata["test-key"] = "test-value"

		b, ok, err := resolver.Resolve(libpak.BindingConstraint{Tags: []string{"test-tag-1", "test-tag-2"}})
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		Expect(b).To(Equal(expected))
	})
}
