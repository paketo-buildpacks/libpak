/*
 * Copyright 2018-2025 the original author or authors.
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

package sbom

import (
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"strings"

	"github.com/buildpacks/libcnb/v2"
)

const (
	// CycloneDXSpecVersion is the version of the CycloneDX specification written by libpak. It matches the version
	// written by the Syft CLI, so that every CycloneDX document in an image uses the same one.
	CycloneDXSpecVersion = "1.7"

	// CycloneDXExtraCPEProperty is the property Syft and Grype use for the CPEs of a component beyond the first one,
	// as CycloneDX only has room for a single CPE.
	CycloneDXExtraCPEProperty = "syft:cpe23"
)

// CycloneDXDocument is the subset of a CycloneDX BOM needed to describe what a buildpack knows about the dependencies
// it installs. It deliberately has no serial number and no timestamp, which would make builds irreproducible.
type CycloneDXDocument struct {
	Schema      string               `json:"$schema"`
	BOMFormat   string               `json:"bomFormat"`
	SpecVersion string               `json:"specVersion"`
	Version     int                  `json:"version"`
	Metadata    CycloneDXMetadata    `json:"metadata"`
	Components  []CycloneDXComponent `json:"components"`
	// Dependencies says which components a component is made of
	Dependencies []CycloneDXDependency `json:"dependencies,omitempty"`
}

type CycloneDXDependency struct {
	Ref       string   `json:"ref"`
	DependsOn []string `json:"dependsOn"`
}

type CycloneDXMetadata struct {
	Tools CycloneDXTools `json:"tools"`
}

type CycloneDXTools struct {
	Components []CycloneDXComponent `json:"components"`
}

type CycloneDXComponent struct {
	Type               string                       `json:"type"`
	BOMRef             string                       `json:"bom-ref,omitempty"`
	Author             string                       `json:"author,omitempty"`
	Name               string                       `json:"name"`
	Version            string                       `json:"version,omitempty"`
	Hashes             []CycloneDXHash              `json:"hashes,omitempty"`
	Licenses           []CycloneDXLicenseChoice     `json:"licenses,omitempty"`
	CPE                string                       `json:"cpe,omitempty"`
	PURL               string                       `json:"purl,omitempty"`
	ExternalReferences []CycloneDXExternalReference `json:"externalReferences,omitempty"`
	Properties         []CycloneDXProperty          `json:"properties,omitempty"`
}

type CycloneDXHash struct {
	Algorithm string `json:"alg"`
	Content   string `json:"content"`
}

// CycloneDXLicenseChoice is either a license or an SPDX expression, never both.
type CycloneDXLicenseChoice struct {
	License    *CycloneDXLicense `json:"license,omitempty"`
	Expression string            `json:"expression,omitempty"`
}

type CycloneDXLicense struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type CycloneDXExternalReference struct {
	Type   string          `json:"type"`
	URL    string          `json:"url"`
	Hashes []CycloneDXHash `json:"hashes,omitempty"`
}

type CycloneDXProperty struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// NewCycloneDXDependency creates a CycloneDX document listing the given components.
func NewCycloneDXDependency(components []CycloneDXComponent) CycloneDXDocument {
	return CycloneDXDocument{
		Schema:      fmt.Sprintf("http://cyclonedx.org/schema/bom-%s.schema.json", CycloneDXSpecVersion),
		BOMFormat:   "CycloneDX",
		SpecVersion: CycloneDXSpecVersion,
		Version:     1,
		Metadata: CycloneDXMetadata{
			Tools: CycloneDXTools{
				Components: []CycloneDXComponent{{Type: "application", Author: "Paketo Buildpacks", Name: "libpak"}},
			},
		},
		Components: components,
	}
}

func (d CycloneDXDocument) WriteTo(path string) error {
	output, err := json.Marshal(&d)
	if err != nil {
		return fmt.Errorf("unable to marshal to JSON\n%w", err)
	}

	// #nosec G306 - permissions need to be 644 on the sbom file
	err = os.WriteFile(path, output, 0644)
	if err != nil {
		return fmt.Errorf("unable to write to path %s\n%w", path, err)
	}

	return nil
}

// NewCycloneDXHash converts a checksum algorithm and its hex digest to a CycloneDX hash. It returns false when
// CycloneDX has no name for the algorithm or the digest is empty.
func NewCycloneDXHash(algorithm string, digest string) (CycloneDXHash, bool) {
	algorithms := map[string]string{
		"md5":    "MD5",
		"sha1":   "SHA-1",
		"sha256": "SHA-256",
		"sha384": "SHA-384",
		"sha512": "SHA-512",
	}

	alg, ok := algorithms[strings.ToLower(algorithm)]
	if !ok || digest == "" {
		return CycloneDXHash{}, false
	}

	return CycloneDXHash{Algorithm: alg, Content: strings.ToLower(digest)}, true
}

// NewCycloneDXLicenses converts license types to CycloneDX: an SPDX id, or an expression when the type has
// whitespace. CycloneDX only allows an expression on its own, so with other licenses it becomes a name.
func NewCycloneDXLicenses(types []string, uris []string) []CycloneDXLicenseChoice {
	licenses := []CycloneDXLicenseChoice{}

	for i, t := range types {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}

		var uri string
		if i < len(uris) {
			uri = uris[i]
		}

		switch {
		case !strings.ContainsAny(t, " \t"):
			licenses = append(licenses, CycloneDXLicenseChoice{License: &CycloneDXLicense{ID: t, URL: uri}})
		case len(types) == 1:
			licenses = append(licenses, CycloneDXLicenseChoice{Expression: t})
		default:
			licenses = append(licenses, CycloneDXLicenseChoice{License: &CycloneDXLicense{Name: t, URL: uri}})
		}
	}

	return licenses
}

// FormatDeclared reports whether format is one of the SBOM media types a buildpack declares in the sbom-formats of
// its buildpack.toml. libcnb and the lifecycle both fail a build that writes an SBOM in an undeclared format.
func FormatDeclared(declared []string, format libcnb.SBOMFormat) bool {
	for _, d := range declared {
		mediaType, _, err := mime.ParseMediaType(d)
		if err != nil {
			continue
		}

		if mediaType == format.MediaType() {
			return true
		}
	}

	return false
}
