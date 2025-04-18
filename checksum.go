package libpak

import (
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"strings"
)

// Checksum represents a checksum algorithm and hash pair formatted as
// algorithm:hash.
// Source: https://github.com/paketo-buildpacks/packit/blob/9f6c6ec9e475e38b3e741d7157ca993c2ad7fbf3/cargo/checksum.go
type Checksum string

// Algorithm returns the algorithm portion of the checksum string. If that
// portion is missing, it defaults to "sha256".
func (c Checksum) Algorithm() string {
	algorithm, _, found := strings.Cut(string(c), ":")
	if !found {
		return "sha256"
	}

	return algorithm
}

// AlgorithmHash return the corresponding hash.Hash interface for the
// algorithm portion of the checksum string
func (c Checksum) AlgorithmHash() (hash.Hash, error) {
	switch c.Algorithm() {
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported checksum algorithm: %s", c.Algorithm())
	}
}

// Hash returns the hexadecimal encoded hash portion of the checksum string.
func (c Checksum) Hash() string {
	_, hash, found := strings.Cut(string(c), ":")
	if !found {
		hash = string(c)
	}

	return hash
}

// Equal returns true only when the given checksum algorithms and hashes
// match.
func (c Checksum) Equal(o Checksum) bool {
	return strings.EqualFold(c.Algorithm(), o.Algorithm()) && c.Hash() == o.Hash()
}

// MatchString returns true only when the given checksum formatted string
// algorithms and hashes match.
func (c Checksum) MatchString(o string) bool {
	return c.Equal(Checksum(o))
}
