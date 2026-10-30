package libpak

import (
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// compareVersions orders two semver versions, returning a negative number when
// a is older than b, a positive number when a is newer than b, and zero when
// they are equivalent.
//
// Semver ignores build metadata when comparing versions, so 21.0.13,
// 21.0.13+0.10, and 21.0.13+0.11 all compare equal. Vendors republish the same
// version with a new build, and the newer build has to win, so fall back to
// comparing the build metadata when the precedence is equal.
func compareVersions(a, b string) int {
	aVersion := semver.MustParse(a)
	bVersion := semver.MustParse(b)

	if result := aVersion.Compare(bVersion); result != 0 {
		return result
	}

	return compareBuildMetadata(aVersion.Metadata(), bVersion.Metadata())
}

// compareBuildMetadata orders two semver build metadata strings. Numeric
// identifiers are compared numerically so that +0.10 sorts after +0.9, and all
// other identifiers are compared lexically. A version that carries build
// metadata is considered newer than one that does not, and a longer metadata
// string is considered newer when it extends the other.
func compareBuildMetadata(a, b string) int {
	if a == b {
		return 0
	}

	if a == "" {
		return -1
	}

	if b == "" {
		return 1
	}

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		aPart, bPart := aParts[i], bParts[i]

		aNumber, aErr := strconv.Atoi(aPart)
		bNumber, bErr := strconv.Atoi(bPart)

		if aErr == nil && bErr == nil {
			if aNumber != bNumber {
				if aNumber < bNumber {
					return -1
				}
				return 1
			}
			continue
		}

		if result := strings.Compare(aPart, bPart); result != 0 {
			return result
		}
	}

	switch {
	case len(aParts) < len(bParts):
		return -1
	case len(aParts) > len(bParts):
		return 1
	default:
		return 0
	}
}
