package libpak_test

import (
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"hash"
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

	it("returns the correct hash algorithm", func() {
		tests := []struct {
			input    libpak.Checksum
			expected hash.Hash
			err      error
		}{
			{"sha512:abcdef", sha512.New(), nil},
			{"sha256:abcdef", sha256.New(), nil},
			{"abcdef", sha256.New(), nil},
			{"md5:abcdef", nil, errors.New("unsupported checksum algorithm: md5")},
		}

		for _, test := range tests {
			algorithm, err := test.input.AlgorithmHash()
			if test.err != nil {
				Expect(err).To(MatchError(test.err))
			} else {
				Expect(err).To(BeNil())
				Expect(algorithm).To(Equal(test.expected))
			}
		}
	})
}
