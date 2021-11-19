package sbom_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
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
		var err error

		executor = mocks.Executor{}

		layers.Path, err = ioutil.TempDir("", "buildpack-layers")
		Expect(err).NotTo(HaveOccurred())

		layer = libcnb.Layer{
			Path: filepath.Join(layers.Path, "layer"),
			Name: "test-layer",
		}

		Expect(os.MkdirAll(layer.Path, 0755)).To(Succeed())
	})

	it.After(func() {
		Expect(os.RemoveAll(layers.Path)).To(Succeed())
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
					e.Args[3] == "json" &&
					e.Args[4] == "dir:something"
			})).Run(func(args mock.Arguments) {
				Expect(ioutil.WriteFile(outputPath, []byte("succeed1"), 0644)).To(Succeed())
			}).Return(nil)

			// uses interface here intentionally, to force that inteface and implementation match
			scanner = sbom.NewSyftCLISBOMScanner(layers, &executor, bard.NewLogger(io.Discard))

			Expect(scanner.ScanBuild("something", format)).To(Succeed())

			result, err := ioutil.ReadFile(outputPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(result)).To(Equal("succeed1"))
		})

		it("runs syft once to generate layer-specific JSON", func() {
			format := libcnb.SyftJSON
			outputPath := layer.SBOMPath(format)

			executor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "syft" &&
					len(e.Args) == 5 &&
					e.Args[3] == "json" &&
					e.Args[4] == "dir:something"
			})).Run(func(args mock.Arguments) {
				Expect(ioutil.WriteFile(outputPath, []byte("succeed2"), 0644)).To(Succeed())
			}).Return(nil)

			scanner := sbom.SyftCLISBOMScanner{
				Executor: &executor,
				Layers:   layers,
				Logger:   bard.NewLogger(io.Discard),
			}

			Expect(scanner.ScanLayer(layer, "something", format)).To(Succeed())

			result, err := ioutil.ReadFile(outputPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(result)).To(Equal("succeed2"))
		})

		it("runs syft twice, once per format", func() {
			outputPaths := map[libcnb.SBOMFormat]string{
				libcnb.SPDXJSON: layers.LaunchSBOMPath(libcnb.SPDXJSON),
				libcnb.SyftJSON: layers.LaunchSBOMPath(libcnb.SyftJSON),
			}

			for format, outputPath := range outputPaths {
				executor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
					return e.Command == "syft" &&
						len(e.Args) == 5 &&
						e.Args[3] == sbom.SBOMFormatToSyftOutputFormat(format) &&
						e.Args[4] == "dir:something"
				})).Run(func(args mock.Arguments) {
					Expect(ioutil.WriteFile(outputPath, []byte("succeed3"), 0644)).To(Succeed())
				}).Return(nil)

				scanner := sbom.SyftCLISBOMScanner{
					Executor: &executor,
					Layers:   layers,
					Logger:   bard.NewLogger(io.Discard),
				}

				Expect(scanner.ScanLaunch("something", format)).To(Succeed())

				result, err := ioutil.ReadFile(outputPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(result)).To(Equal("succeed3"))
			}
		})

		it("converts between cyclonedx XML and JSON", func() {
			outputPath := layers.BuildSBOMPath(libcnb.CycloneDXJSON)
			Expect(ioutil.WriteFile(outputPath, []byte(`<?xml version="1.0" encoding="UTF-8"?>
<bom xmlns="http://cyclonedx.org/schema/bom/1.2" version="1" serialNumber="urn:uuid:48051e17-8720-4503-a2ef-47efab3fc03f">
  <metadata>
    <timestamp>2021-11-15T16:15:46-05:00</timestamp>
    <tools>
      <tool>
        <vendor>anchore</vendor>
        <name>syft</name>
        <version>0.29.0</version>
      </tool>
    </tools>
    <component type="file">
      <name>.</name>
      <version></version>
    </component>
  </metadata>
  <components>
    <component type="library">
      <name>github.com/BurntSushi/toml</name>
      <version>v0.4.1</version>
      <purl>pkg:golang/github.com/BurntSushi/toml@v0.4.1</purl>
    </component>
  </components>
</bom>`), 0644))

			scanner := sbom.SyftCLISBOMScanner{
				Executor: &executor,
				Layers:   layers,
				Logger:   bard.NewLogger(io.Discard),
			}

			Expect(scanner.ConvertCycloneDXXMLtoJSON(outputPath, false)).To(Succeed())

			Expect(outputPath).To(BeARegularFile())
			Expect(fmt.Sprintf("%s.bak", outputPath)).ToNot(BeARegularFile())

			input, err := ioutil.ReadFile(outputPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(input)).To(ContainSubstring(`{"type":"library","name":"github.com/BurntSushi/toml","version":"v0.4.1","purl":"pkg:golang/github.com/BurntSushi/toml@v0.4.1"}`))
		})

		it("converts between cyclonedx XML and JSON with backup", func() {
			outputPath := layers.LaunchSBOMPath(libcnb.CycloneDXJSON)
			Expect(ioutil.WriteFile(outputPath, []byte(`<?xml version="1.0" encoding="UTF-8"?>
<bom xmlns="http://cyclonedx.org/schema/bom/1.2" version="1" serialNumber="urn:uuid:48051e17-8720-4503-a2ef-47efab3fc03f">
  <metadata>
    <timestamp>2021-11-15T16:15:46-05:00</timestamp>
    <tools>
      <tool>
        <vendor>anchore</vendor>
        <name>syft</name>
        <version>0.29.0</version>
      </tool>
    </tools>
    <component type="file">
      <name>.</name>
      <version></version>
    </component>
  </metadata>
  <components>
    <component type="library">
      <name>github.com/BurntSushi/toml</name>
      <version>v0.4.1</version>
      <purl>pkg:golang/github.com/BurntSushi/toml@v0.4.1</purl>
    </component>
  </components>
</bom>`), 0644))

			scanner := sbom.SyftCLISBOMScanner{
				Executor: &executor,
				Layers:   layers,
				Logger:   bard.NewLogger(io.Discard),
			}

			Expect(scanner.ConvertCycloneDXXMLtoJSON(outputPath, true)).To(Succeed())

			Expect(outputPath).To(BeARegularFile())

			input, err := ioutil.ReadFile(outputPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(input)).To(ContainSubstring(`{"type":"library","name":"github.com/BurntSushi/toml","version":"v0.4.1","purl":"pkg:golang/github.com/BurntSushi/toml@v0.4.1"}`))

			outputPath = fmt.Sprintf("%s.bak", outputPath)
			Expect(outputPath).To(BeARegularFile())

			input, err = ioutil.ReadFile(outputPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(input)).To(ContainSubstring(`<bom xmlns="http://cyclonedx.org/schema/bom/1.2" version="1" serialNumber="urn:uuid:48051e17-8720-4503-a2ef-47efab3fc03f">`))
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
					Version: "0.30.1",
				},
				Schema: sbom.SyftSchema{
					Version: "1.1.0",
					URL:     "https://raw.githubusercontent.com/anchore/syft/main/schema/json/schema-1.1.0.json",
				},
			}
			outputFile := filepath.Join(layers.Path, "test-bom.json")
			Expect(dep.WriteTo(outputFile)).To(Succeed())
			Expect(outputFile).To(BeARegularFile())

			data, err := ioutil.ReadFile(outputFile)
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

			data, err := ioutil.ReadFile(outputFile)
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
