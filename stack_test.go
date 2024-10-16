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

	context("bionic stacks", func() {
		it("matches standard bionic stack", func() {
			Expect(libpak.IsBionicStack("io.buildpacks.stacks.bionic")).To(BeTrue())
		})

		it("matches tiny bionic stack", func() {
			Expect(libpak.IsBionicStack("io.paketo.stacks.tiny")).To(BeTrue())
		})

		it("does not match non-bionic stack", func() {
			Expect(libpak.IsBionicStack("io.buildpacks.stacks.jammy")).To(BeFalse())
		})
	})

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
			Expect(libpak.IsJammyStack("io.buildpacks.stacks.bionic")).To(BeFalse())
		})
	})

	context("tiny stacks", func() {
		it("matches tiny bionic stack", func() {
			Expect(libpak.IsTinyStack("io.paketo.stacks.tiny")).To(BeTrue())
		})

		it("matches tiny jammy stack", func() {
			Expect(libpak.IsTinyStack("io.buildpacks.stacks.jammy.tiny")).To(BeTrue())
		})

		it("does not match full stack", func() {
			Expect(libpak.IsTinyStack("io.buildpacks.stacks.bionic")).To(BeFalse())
		})
	})

	context("static stack", func() {
		it("matches static jammy stack", func() {
			Expect(libpak.IsStaticStack("io.buildpacks.stacks.jammy.static")).To(BeTrue())
		})

		it("does not match full stack", func() {
			Expect(libpak.IsTinyStack("io.buildpacks.stacks.bionic")).To(BeFalse())
		})
	})

	context("shell", func() {
		it("matches a full stack", func() {
			Expect(libpak.IsShellPresentOnStack("io.buildpacks.stacks.bionic")).To(BeTrue())
		})

		it("does not match static jammy stack", func() {
			Expect(libpak.IsShellPresentOnStack("io.buildpacks.stacks.jammy.static")).To(BeFalse())
		})
	})
}
