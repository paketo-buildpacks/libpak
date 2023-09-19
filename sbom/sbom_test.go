package sbom_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/effect"
	"github.com/paketo-buildpacks/libpak/effect/mocks"
	"github.com/paketo-buildpacks/libpak/sbom"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"
)

func testSBOM(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layers   libcnb.Layers
		layer    libcnb.Layer
		executor mocks.Executor
		scanner  sbom.SBOMScanner
	)

	it.Before(func() {
		executor = mocks.Executor{}

		layers.Path = t.TempDir()

		layer = libcnb.Layer{
			Path: filepath.Join(layers.Path, "layer"),
			Name: "test-layer",
		}

		Expect(os.MkdirAll(layer.Path, 0755)).To(Succeed())
	})

	context("syft", func() {
		it("generates artifact id", func() {
			artifact := sbom.SyftArtifact{Name: "foo", Version: "1.2.3"}
			ID, err := artifact.Hash()
			Expect(err).ToNot(HaveOccurred())
			Expect(ID).To(Equal("7f6c18a85645bd7c"))
		})

		it("runs syft once to generate JSON", func() {
			format := libcnb.SyftJSON
			outputPath := layers.BuildSBOMPath(format)

			executor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "syft" &&
					len(e.Args) == 5 &&
					strings.HasPrefix(e.Args[3], "json=") &&
					e.Args[4] == "dir:something"
			})).Run(func(args mock.Arguments) {
				Expect(os.WriteFile(outputPath, []byte("succeed1"), 0644)).To(Succeed())
			}).Return(nil)

			// uses interface here intentionally, to force that inteface and implementation match
			scanner = sbom.NewSyftCLISBOMScanner(layers, &executor, bard.NewLogger(io.Discard))

			Expect(scanner.ScanBuild("something", format)).To(Succeed())

			result, err := os.ReadFile(outputPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(result)).To(Equal("succeed1"))
		})

		it("runs syft to generate reproducible cycloneDX JSON", func() {
			format := libcnb.CycloneDXJSON
			outputPath := layers.BuildSBOMPath(format)

			executor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "syft" &&
					len(e.Args) == 5 &&
					strings.HasPrefix(e.Args[3], "cyclonedx-json=") &&
					e.Args[4] == "dir:something"
			})).Run(func(args mock.Arguments) {
				Expect(os.WriteFile(outputPath, []byte(`{
  "bomFormat": "CycloneDX",
  "specVersion": "1.4",
  "serialNumber": "urn:uuid:fcfa5e19-bf49-47b4-8c85-ab61e2728f8e",
  "version": 1,
  "metadata": {
    "timestamp": "2022-05-05T11:33:13-04:00",
    "tools": [
      {
        "vendor": "anchore",
        "name": "syft",
        "version": "0.45.1"
      }
    ],
    "component": {
      "bom-ref": "555d623e4777b7ae",
      "type": "file",
      "name": "target/demo-0.0.1-SNAPSHOT.jar"
    }
  }
}`), 0644)).To(Succeed())
			}).Return(nil)

			// uses interface here intentionally, to force that inteface and implementation match
			scanner = sbom.NewSyftCLISBOMScanner(layers, &executor, bard.NewLogger(io.Discard))

			Expect(scanner.ScanBuild("something", format)).To(Succeed())

			result, err := os.ReadFile(outputPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(result)).ToNot(ContainSubstring("serialNumber"))
			Expect(string(result)).ToNot(ContainSubstring("urn:uuid:fcfa5e19-bf49-47b4-8c85-ab61e2728f8e"))
			Expect(string(result)).ToNot(ContainSubstring("timestamp"))
			Expect(string(result)).ToNot(ContainSubstring("2022-05-05T11:33:13-04:00"))
		})

		it("runs syft once to generate layer-specific JSON", func() {
			format := libcnb.SyftJSON
			outputPath := layer.SBOMPath(format)

			executor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "syft" &&
					len(e.Args) == 5 &&
					strings.HasPrefix(e.Args[3], "json=") &&
					e.Args[4] == "dir:something"
			})).Run(func(args mock.Arguments) {
				Expect(os.WriteFile(outputPath, []byte("succeed2"), 0644)).To(Succeed())
			}).Return(nil)

			scanner := sbom.SyftCLISBOMScanner{
				Executor: &executor,
				Layers:   layers,
				Logger:   bard.NewLogger(io.Discard),
			}

			Expect(scanner.ScanLayer(layer, "something", format)).To(Succeed())

			result, err := os.ReadFile(outputPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(result)).To(Equal("succeed2"))
		})

		it("runs syft once for all three formats", func() {
			executor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "syft" &&
					len(e.Args) == 9 &&
					strings.HasPrefix(e.Args[3], sbom.SBOMFormatToSyftOutputFormat(libcnb.CycloneDXJSON)) &&
					strings.HasPrefix(e.Args[5], sbom.SBOMFormatToSyftOutputFormat(libcnb.SyftJSON)) &&
					strings.HasPrefix(e.Args[7], sbom.SBOMFormatToSyftOutputFormat(libcnb.SPDXJSON)) &&
					e.Args[8] == "dir:something"
			})).Run(func(args mock.Arguments) {
				Expect(os.WriteFile(layers.LaunchSBOMPath(libcnb.CycloneDXJSON), []byte(`{"succeed":1}`), 0644)).To(Succeed())
				Expect(os.WriteFile(layers.LaunchSBOMPath(libcnb.SyftJSON), []byte(`{"succeed":2}`), 0644)).To(Succeed())
				Expect(os.WriteFile(layers.LaunchSBOMPath(libcnb.SPDXJSON), []byte(`{"succeed":3}`), 0644)).To(Succeed())
			}).Return(nil)

			scanner := sbom.SyftCLISBOMScanner{
				Executor: &executor,
				Layers:   layers,
				Logger:   bard.NewLogger(io.Discard),
			}

			Expect(scanner.ScanLaunch("something", libcnb.CycloneDXJSON, libcnb.SyftJSON, libcnb.SPDXJSON)).To(Succeed())

			result, err := os.ReadFile(layers.LaunchSBOMPath(libcnb.CycloneDXJSON))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(result)).To(HavePrefix(`{"succeed":1}`))

			result, err = os.ReadFile(layers.LaunchSBOMPath(libcnb.SyftJSON))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(result)).To(HavePrefix(`{"succeed":2}`))

			result, err = os.ReadFile(layers.LaunchSBOMPath(libcnb.SPDXJSON))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(result)).To(HavePrefix(`{"succeed":3}`))
		})

		it("writes out a manual BOM entry", func() {
			dep := sbom.SyftDependency{
				Artifacts: []sbom.SyftArtifact{
					{
						ID:      "1234",
						Name:    "test-dep",
						Version: "1.2.3",
						Type:    "UnknownPackage",
						FoundBy: "java-buildpack",
						Locations: []sbom.SyftLocation{
							{Path: "/some/path"},
						},
						Licenses: []string{"GPL-2.0 WITH Classpath-exception-2.0"},
						Language: "java",
						CPEs: []string{
							"cpe:2.3:a:some:jre:11.0.2:*:*:*:*:*:*:*",
						},
						PURL: "pkg:generic/some-java11@11.0.2?arch=amd64",
					},
				},
				Source: sbom.SyftSource{
					Type:   "directory",
					Target: "path/to/layer",
				},
				Descriptor: sbom.SyftDescriptor{
					Name:    "syft",
					Version: "0.32.0",
				},
				Schema: sbom.SyftSchema{
					Version: "1.1.0",
					URL:     "https://raw.githubusercontent.com/anchore/syft/main/schema/json/schema-1.1.0.json",
				},
			}
			outputFile := filepath.Join(layers.Path, "test-bom.json")
			Expect(dep.WriteTo(outputFile)).To(Succeed())
			Expect(outputFile).To(BeARegularFile())

			data, err := os.ReadFile(outputFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(`"Artifacts":[`))
			Expect(string(data)).To(ContainSubstring(`"FoundBy":"java-buildpack",`))
			Expect(string(data)).To(ContainSubstring(`"PURL":"pkg:generic/some-java11@11.0.2?arch=amd64"`))
			Expect(string(data)).To(ContainSubstring(`"Schema":{`))
			Expect(string(data)).To(ContainSubstring(`"Descriptor":{`))
			Expect(string(data)).To(ContainSubstring(`"Source":{`))
		})

		it("writes out a manual BOM entry with help", func() {
			dep := sbom.NewSyftDependency("path/to/layer", []sbom.SyftArtifact{
				{
					ID:      "1234",
					Name:    "test-dep",
					Version: "1.2.3",
					Type:    "UnknownPackage",
					FoundBy: "java-buildpack",
					Locations: []sbom.SyftLocation{
						{Path: "/some/path"},
					},
					Licenses: []string{"GPL-2.0 WITH Classpath-exception-2.0"},
					Language: "java",
					CPEs: []string{
						"cpe:2.3:a:some:jre:11.0.2:*:*:*:*:*:*:*",
					},
					PURL: "pkg:generic/some-java11@11.0.2?arch=amd64",
				},
			})

			outputFile := filepath.Join(layers.Path, "test-bom.json")
			Expect(dep.WriteTo(outputFile)).To(Succeed())
			Expect(outputFile).To(BeARegularFile())

			data, err := os.ReadFile(outputFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(data)).To(ContainSubstring(`"Artifacts":[`))
			Expect(string(data)).To(ContainSubstring(`"FoundBy":"java-buildpack",`))
			Expect(string(data)).To(ContainSubstring(`"PURL":"pkg:generic/some-java11@11.0.2?arch=amd64"`))
			Expect(string(data)).To(ContainSubstring(`"Schema":{`))
			Expect(string(data)).To(ContainSubstring(`"Descriptor":{`))
			Expect(string(data)).To(ContainSubstring(`"Source":{`))
		})
	})

}
