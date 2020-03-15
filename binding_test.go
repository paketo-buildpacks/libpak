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
	"github.com/paketo-buildpacks/libpak"
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

	context("Resolve", func() {

		it("returns error if binding does not exist", func() {
			_, err := resolver.Resolve("", "")
			Expect(err).To(MatchError(libpak.NoValidBindingError{Message: "no valid binding for , , and [] in map[]"}))
		})

		it("returns error if multiple bindings exist", func() {
			resolver.Bindings["test-binding-1"] = libcnb.NewBinding()
			resolver.Bindings["test-binding-2"] = libcnb.NewBinding()

			_, err := resolver.Resolve("", "")
			Expect(err).To(MatchError(libpak.NoValidBindingError{Message: fmt.Sprintf("no valid binding for , , and [] in %s", resolver.Bindings)}))
		})

		it("filters on kind", func() {
			resolver.Bindings["test-binding-1"] = libcnb.NewBinding()
			resolver.Bindings["test-binding-2"] = libcnb.NewBinding()
			resolver.Bindings["test-binding-2"].Metadata[libcnb.BindingKind] = "test-kind"
			resolver.Bindings["test-binding-2"].Metadata["test-key"] = "test-value"

			expected := libcnb.NewBinding()
			expected.Metadata[libcnb.BindingKind] = "test-kind"
			expected.Metadata["test-key"] = "test-value"

			b, err := resolver.Resolve("test-kind", "")
			Expect(err).NotTo(HaveOccurred())
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

			b, err := resolver.Resolve("", "test-provider")
			Expect(err).NotTo(HaveOccurred())
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

			b, err := resolver.Resolve("", "", "test-tag-1", "test-tag-2")
			Expect(err).NotTo(HaveOccurred())
			Expect(b).To(Equal(expected))
		})
	})

	context("Any", func() {

		it("indicates that binding exists", func() {
			resolver.Bindings["test-binding-1"] = libcnb.NewBinding()
			resolver.Bindings["test-binding-2"] = libcnb.NewBinding()
			resolver.Bindings["test-binding-2"].Metadata[libcnb.BindingKind] = "test-kind"
			resolver.Bindings["test-binding-2"].Metadata["test-key"] = "test-value"

			Expect(resolver.Any("test-kind", "")).To(BeTrue())
		})

		it("indicates that binding does not exist", func() {
			Expect(resolver.Any("test-kind", "")).To(BeFalse())
		})
	})

	it("indicates whether error is NoValidBindingError", func() {
		Expect(libpak.IsNoValidBinding(nil)).To(BeFalse())
		Expect(libpak.IsNoValidBinding(fmt.Errorf("test-error"))).To(BeFalse())
		Expect(libpak.IsNoValidBinding(libpak.NoValidBindingError{})).To(BeTrue())
	})
}
