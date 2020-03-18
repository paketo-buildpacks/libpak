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

	it("returns error if binding does not exist", func() {
		_, ok, err := resolver.Resolve("", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeFalse())
	})

	it("returns error if multiple bindings exist", func() {
		resolver.Bindings = libcnb.Bindings{
			libcnb.NewBinding("test-binding-1"),
			libcnb.NewBinding("test-binding-2"),
		}

		_, _, err := resolver.Resolve("", "")
		Expect(err).To(MatchError(fmt.Sprintf("multiple bindings found for , , and [] in %s", resolver.Bindings)))
	})

	it("filters on kind", func() {
		c := libcnb.NewBinding("test-binding-2")
		c.Metadata[libcnb.BindingKind] = "test-kind"
		c.Metadata["test-key"] = "test-value"

		resolver.Bindings = libcnb.Bindings{
			libcnb.NewBinding("test-binding-1"),
			c,
		}

		b, ok, err := resolver.Resolve("test-kind", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		Expect(b).To(Equal(c))
	})

	it("filters on provider", func() {
		c := libcnb.NewBinding("test-binding-2")
		c.Metadata[libcnb.BindingProvider] = "test-provider"
		c.Metadata["test-key"] = "test-value"

		resolver.Bindings = libcnb.Bindings{
			libcnb.NewBinding("test-binding-1"),
			c,
		}

		b, ok, err := resolver.Resolve("", "test-provider")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		Expect(b).To(Equal(c))
	})

	it("filters on all tags", func() {
		c := libcnb.NewBinding("test-binding-2")
		c.Metadata[libcnb.BindingTags] = "test-tag-1\ntest-tag-2\ntest-tag-3"
		c.Metadata["test-key"] = "test-value"

		resolver.Bindings = libcnb.Bindings{
			libcnb.NewBinding("test-binding-1"),
			c,
		}

		b, ok, err := resolver.Resolve("", "", "test-tag-1", "test-tag-2")
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
		Expect(b).To(Equal(c))
	})
}
