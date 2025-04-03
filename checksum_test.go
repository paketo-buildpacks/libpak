package libpak_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak/v2"
)

func testChecksum(t *testing.T, _ spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	it("determines the correct algorithm", func() {
		tests := []struct {
			input    libpak.Checksum
			expected string
		}{
			{"sha256:abcdef", "sha256"},
			{"sha512:123456", "sha512"},
			{"abcdef", "sha256"}, // Default to sha256
		}

		for _, test := range tests {
			Expect(test.input.Algorithm()).To(Equal(test.expected))
		}
	})

	it("determines the correct checksum", func() {
		tests := []struct {
			input    libpak.Checksum
			expected string
		}{
			{"sha256:abcdef", "abcdef"},
			{"sha512:123456", "123456"},
			{"abcdef", "abcdef"}, // Default to sha256
		}

		for _, test := range tests {
			Expect(test.input.Hash()).To(Equal(test.expected))
		}
	})

	it("matches checksums", func() {
		tests := []struct {
			input1   libpak.Checksum
			input2   libpak.Checksum
			expected bool
		}{
			{"sha256:abcdef", "sha256:abcdef", true},
			{"abcdef", "abcdef", true},
			{"sha512:abcdef", "sha256:abcdef", false},
			{"sha256:abcdef", "sha256:abcdez", false},
		}

		for _, test := range tests {
			Expect(test.input1.Match(test.input2)).To(Equal(test.expected))
		}
	})
}
