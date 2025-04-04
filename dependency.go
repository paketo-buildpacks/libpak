package libpak

import (
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/paketo-buildpacks/libpak/v2/sbom"
)

// BuildModuleDependencyLicense represents a license that a BuildModuleDependency is distributed under.
// At least one of Name or URI MUST be specified.
type BuildModuleDependencyLicense struct {
	// Type is the type of the license.  This is typically the SPDX short identifier.
	Type string `toml:"type"`

	// URI is the location where the license can be found.
	URI string `toml:"uri"`
}

// BuildModuleDependencyDistro represents a supported distribution of a BuildModuleDependency
type BuildModuleDependencyDistro struct {
	// Name is the name of the distribution.
	Name string `toml:"name"`
	// Version is the version of the distribution.
	Version string `toml:"version"`
}

// BuildModuleDependency describes a dependency known to the buildpack/extension
type BuildModuleDependency struct {
	// ID is the dependency ID.
	ID string `toml:"id"`

	// Name is the dependency name.
	Name string `toml:"name"`

	// Version is the dependency version.
	Version string `toml:"version"`

	// URI is the dependency URI.
	URI string `toml:"uri"`

	// Deprecated: use GetChecksum()
	SHA256 Checksum `toml:"sha256"`

	// Checksum is a checksum of the dependency in the format "<algorithm>:<checksum>"
	// e.g. "sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	Checksum Checksum `toml:"checksum"`

	// Stacks are the stacks the dependency is compatible with.
	// Deprecated: use Distros fields instead
	Stacks []string `toml:"stacks"`

	// Licenses are the licenses the dependency is distributed under.
	Licenses []BuildModuleDependencyLicense `toml:"licenses"`

	// CPEs are the Common Platform Enumeration identifiers for the dependency
	CPEs []string `toml:"cpes"`

	// Deprecated: Use GetPURLS()
	PURL string `toml:"purl"`

	// PURLS are the Package URLs for the dependency
	PURLS []string `toml:"purls"`

	// Deprecated: Use GetEOLDate()
	DeprecationDate time.Time `toml:"deprecation_date"`

	// EOLDate is the time when the dependency is end of life
	EOLDate time.Time `toml:"eol-date"`

	// Arch is the architecture of the dependency
	Arch string `toml:"arch"`

	// Distros is the distribution of the dependency
	Distros []BuildModuleDependencyDistro `toml:"distros"`

	// StripComponents is the number of components to strip from the path
	StripComponents uint `toml:"strip-components"`

	// Source is the source of the dependency
	Source string `toml:"source"`

	// SourceChecksum is the checksum of the source
	SourceChecksum Checksum `toml:"source-checksum"`
}

// GetMetadata return the relevant metadata of this dependency
func (b BuildModuleDependency) GetMetadata() DependencyLayerContributorMetadata {
	return DependencyLayerContributorMetadata{
		ID:      b.ID,
		Name:    b.Name,
		Version: b.Version,
		SHA256:  b.GetChecksum().Hash(),
	}
}

// Equals compares the 2 structs if they are equal. This is very simiar to reflect.DeepEqual
// except that properties that will not work (e.g. DeprecationDate) are ignored.
func (b BuildModuleDependency) Equals(other BuildModuleDependency) bool {
	return cmp.Equal(b, other, cmpopts.EquateEmpty())
}

// AsSyftArtifact renders a bill of materials entry describing the dependency as Syft.
func (b BuildModuleDependency) AsSyftArtifact(source string) (sbom.SyftArtifact, error) {
	licenses := []string{}
	for _, license := range b.Licenses {
		licenses = append(licenses, license.Type)
	}

	sbomArtifact := sbom.SyftArtifact{
		Name:      b.Name,
		Version:   b.Version,
		Type:      "UnknownPackage",
		FoundBy:   "libpak",
		Licenses:  licenses,
		Locations: []sbom.SyftLocation{{Path: source}},
		CPEs:      b.CPEs,
		PURL:      b.GetPURLS()[0],
	}

	var err error
	sbomArtifact.ID, err = sbomArtifact.Hash()
	if err != nil {
		return sbom.SyftArtifact{}, fmt.Errorf("unable to generate hash\n%w", err)
	}

	return sbomArtifact, nil
}

func (b BuildModuleDependency) IsDeprecated() bool {
	deprecationDate := b.GetEOLDate().UTC()
	now := time.Now().UTC()
	return deprecationDate.Equal(now) || deprecationDate.Before(now)
}

func (b BuildModuleDependency) IsSoonDeprecated() bool {
	deprecationDate := b.GetEOLDate().UTC()
	now := time.Now().UTC()
	return deprecationDate.Add(-30*24*time.Hour).Before(now) && deprecationDate.After(now)
}

func (b BuildModuleDependency) GetEOLDate() time.Time {
	if !b.EOLDate.IsZero() {
		return b.EOLDate
	}

	return b.DeprecationDate
}

func (b BuildModuleDependency) GetPURLS() []string {
	if len(b.PURLS) > 0 {
		return b.PURLS
	}

	return []string{b.PURL}
}

func (b BuildModuleDependency) GetChecksum() Checksum {
	if b.Checksum != "" {
		return b.Checksum
	}

	return b.SHA256
}

func (b BuildModuleDependency) GetArch() (string, error) {
	if b.Arch != "" {
		return b.Arch, nil
	}

	return archFromPURL(b.GetPURLS()[0])
}

func archFromPURL(rawPURL string) (string, error) {
	if len(strings.TrimSpace(rawPURL)) == 0 {
		return "amd64", nil
	}

	purl, err := url.Parse(rawPURL)
	if err != nil {
		return "", fmt.Errorf("unable to parse PURL\n%w", err)
	}

	queryParams := purl.Query()
	if arch, ok := queryParams["arch"]; ok {
		return arch[0], nil
	}

	return archFromSystem(), nil
}

func archFromSystem() string {
	archFromEnv, ok := os.LookupEnv("BP_ARCH")
	if !ok {
		archFromEnv = runtime.GOARCH
	}

	return archFromEnv
}
