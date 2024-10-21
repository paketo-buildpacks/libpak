/*
 * Copyright 2018-2024 the original author or authors.
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
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak/v2"
)

func testStack(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	context("jammy stacks", func() {
		it("matches standard jammy stack", func() {
			Expect(libpak.IsJammyStack("io.buildpacks.stacks.jammy")).To(BeTrue())
		})

		it("matches tiny jammy stack", func() {
			Expect(libpak.IsJammyStack("io.buildpacks.stacks.jammy.tiny")).To(BeTrue())
		})

		it("matches static jammy stack", func() {
			Expect(libpak.IsJammyStack("io.buildpacks.stacks.jammy.static")).To(BeTrue())
		})

		it("does not match non-jammy stack", func() {
			Expect(libpak.IsJammyStack("io.buildpacks.stacks.noble")).To(BeFalse())
		})
	})

	context("noble stacks", func() {
		it("matches standard noble stack", func() {
			Expect(libpak.IsNobleStack("io.buildpacks.stacks.noble")).To(BeTrue())
		})

		it("matches tiny noble stack", func() {
			Expect(libpak.IsNobleStack("io.buildpacks.stacks.noble.tiny")).To(BeTrue())
		})

		it("matches static noble stack", func() {
			Expect(libpak.IsNobleStack("io.buildpacks.stacks.noble.static")).To(BeTrue())
		})

		it("does not match non-noble stack", func() {
			Expect(libpak.IsNobleStack("io.buildpacks.stacks.jammy")).To(BeFalse())
		})
	})

	context("tiny stacks", func() {
		it("matches tiny jammy stack", func() {
			Expect(libpak.IsTinyStack("io.buildpacks.stacks.jammy.tiny")).To(BeTrue())
		})

		it("matches tiny noble stack", func() {
			Expect(libpak.IsTinyStack("io.buildpacks.stacks.noble.tiny")).To(BeTrue())
		})

		it("does not match full stack", func() {
			Expect(libpak.IsTinyStack("io.buildpacks.stacks.jammy")).To(BeFalse())
		})
	})

	context("static stack", func() {
		it("matches static jammy stack", func() {
			Expect(libpak.IsStaticStack("io.buildpacks.stacks.jammy.static")).To(BeTrue())
		})

		it("matches static noble stack", func() {
			Expect(libpak.IsStaticStack("io.buildpacks.stacks.noble.static")).To(BeTrue())
		})

		it("does not match full stack", func() {
			Expect(libpak.IsTinyStack("io.buildpacks.stacks.jammy")).To(BeFalse())
		})
	})

	context("shell", func() {
		it("matches a full jammy stack", func() {
			Expect(libpak.IsShellPresentOnStack("io.buildpacks.stacks.jammy")).To(BeTrue())
		})

		it("matches a full noble stack", func() {
			Expect(libpak.IsShellPresentOnStack("io.buildpacks.stacks.noble")).To(BeTrue())
		})

		it("does not match static jammy stack", func() {
			Expect(libpak.IsShellPresentOnStack("io.buildpacks.stacks.jammy.static")).To(BeFalse())
		})
	})
}
