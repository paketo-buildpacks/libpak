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

package sbom_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/buildpacks/libcnb/v2"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak/v2/sbom"
)

func testCycloneDX(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	it("writes a reproducible document", func() {
		path := filepath.Join(t.TempDir(), "sbom.cdx.json")

		doc := sbom.NewCycloneDXDependency([]sbom.CycloneDXComponent{{Type: "library", Name: "test-name", Version: "1.1.1"}})
		Expect(doc.WriteTo(path)).To(Succeed())

		data, err := os.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(Equal(`{"$schema":"http://cyclonedx.org/schema/bom-1.7.schema.json","bomFormat":"CycloneDX","specVersion":"1.7","version":1,` +
			`"metadata":{"tools":{"components":[{"type":"application","author":"Paketo Buildpacks","name":"libpak"}]}},` +
			`"components":[{"type":"library","name":"test-name","version":"1.1.1"}]}`))
	})

	context("hashes", func() {
		it("names the algorithm the CycloneDX way", func() {
			hash, ok := sbom.NewCycloneDXHash("sha256", "ABCD")
			Expect(ok).To(BeTrue())
			Expect(hash).To(Equal(sbom.CycloneDXHash{Algorithm: "SHA-256", Content: "abcd"}))

			hash, ok = sbom.NewCycloneDXHash("SHA512", "abcd")
			Expect(ok).To(BeTrue())
			Expect(hash).To(Equal(sbom.CycloneDXHash{Algorithm: "SHA-512", Content: "abcd"}))
		})

		it("skips an unknown algorithm or an empty digest", func() {
			_, ok := sbom.NewCycloneDXHash("crc32", "abcd")
			Expect(ok).To(BeFalse())

			_, ok = sbom.NewCycloneDXHash("sha256", "")
			Expect(ok).To(BeFalse())
		})
	})

	context("licenses", func() {
		it("writes an SPDX identifier as a license id", func() {
			Expect(sbom.NewCycloneDXLicenses([]string{"Apache-2.0", "MIT"}, []string{"test-uri"})).To(Equal([]sbom.CycloneDXLicenseChoice{
				{License: &sbom.CycloneDXLicense{ID: "Apache-2.0", URL: "test-uri"}},
				{License: &sbom.CycloneDXLicense{ID: "MIT"}},
			}))
		})

		it("writes a sole SPDX expression as an expression", func() {
			Expect(sbom.NewCycloneDXLicenses([]string{"GPL-2.0 WITH Classpath-exception-2.0"}, nil)).To(Equal([]sbom.CycloneDXLicenseChoice{
				{Expression: "GPL-2.0 WITH Classpath-exception-2.0"},
			}))
		})

		it("writes an SPDX expression as a license name alongside other licenses", func() {
			Expect(sbom.NewCycloneDXLicenses([]string{"GPL-2.0 WITH Classpath-exception-2.0", "MIT"}, nil)).To(Equal([]sbom.CycloneDXLicenseChoice{
				{License: &sbom.CycloneDXLicense{Name: "GPL-2.0 WITH Classpath-exception-2.0"}},
				{License: &sbom.CycloneDXLicense{ID: "MIT"}},
			}))
		})
	})

	context("Go modules", func() {
		it("lists the standard library and the modules a Go binary is built from", func() {
			// the test binary itself is a Go binary built from known modules
			binary, err := os.Executable()
			Expect(err).NotTo(HaveOccurred())

			components, err := sbom.GoModuleComponents(binary, "test-owner")
			Expect(err).NotTo(HaveOccurred())

			version := strings.TrimPrefix(runtime.Version(), "go")
			Expect(components[0]).To(Equal(sbom.CycloneDXComponent{
				Type:    "library",
				BOMRef:  "pkg:golang/stdlib@" + version + "?package-id=test-owner",
				Name:    "stdlib",
				Version: runtime.Version(),
				CPE:     "cpe:2.3:a:golang:go:" + version + ":-:*:*:*:*:*:*",
				PURL:    "pkg:golang/stdlib@" + version,
			}))

			var gomega *sbom.CycloneDXComponent
			for i := range components {
				if components[i].Name == "github.com/onsi/gomega" {
					gomega = &components[i]
				}
			}
			Expect(gomega).NotTo(BeNil())
			Expect(gomega.Version).To(HavePrefix("v"))
			Expect(gomega.PURL).To(Equal("pkg:golang/github.com/onsi/gomega@" + gomega.Version))
			Expect(gomega.BOMRef).To(Equal(gomega.PURL + "?package-id=test-owner"))
		})

		it("fails for a file that is not a Go binary", func() {
			path := filepath.Join(t.TempDir(), "not-a-binary")
			Expect(os.WriteFile(path, []byte("test"), 0600)).To(Succeed())

			_, err := sbom.GoModuleComponents(path, "test-owner")
			Expect(err).To(MatchError(ContainSubstring("unable to read Go build information")))
		})
	})

	context("declared formats", func() {
		it("matches a declared media type", func() {
			declared := []string{"application/vnd.syft+json", "application/vnd.cyclonedx+json;version=1.4"}

			Expect(sbom.FormatDeclared(declared, libcnb.CycloneDXJSON)).To(BeTrue())
			Expect(sbom.FormatDeclared(declared, libcnb.SyftJSON)).To(BeTrue())
			Expect(sbom.FormatDeclared(declared, libcnb.SPDXJSON)).To(BeFalse())
		})

		it("matches nothing when no format is declared", func() {
			Expect(sbom.FormatDeclared(nil, libcnb.CycloneDXJSON)).To(BeFalse())
		})
	})
}
