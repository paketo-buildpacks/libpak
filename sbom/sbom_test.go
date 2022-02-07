package sbom_test

import (
	"io"
	"io/ioutil"
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
					strings.HasPrefix(e.Args[3], "json=") &&
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
					strings.HasPrefix(e.Args[3], "json=") &&
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

		it("runs syft once for all three formats", func() {
			executor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "syft" &&
					len(e.Args) == 9 &&
					strings.HasPrefix(e.Args[3], sbom.SBOMFormatToSyftOutputFormat(libcnb.CycloneDXJSON)) &&
					strings.HasPrefix(e.Args[5], sbom.SBOMFormatToSyftOutputFormat(libcnb.SyftJSON)) &&
					strings.HasPrefix(e.Args[7], sbom.SBOMFormatToSyftOutputFormat(libcnb.SPDXJSON)) &&
					e.Args[8] == "dir:something"
			})).Run(func(args mock.Arguments) {
				Expect(ioutil.WriteFile(layers.LaunchSBOMPath(libcnb.CycloneDXJSON), []byte("succeed1"), 0644)).To(Succeed())
				Expect(ioutil.WriteFile(layers.LaunchSBOMPath(libcnb.SyftJSON), []byte("succeed2"), 0644)).To(Succeed())
				Expect(ioutil.WriteFile(layers.LaunchSBOMPath(libcnb.SPDXJSON), []byte("succeed3"), 0644)).To(Succeed())
			}).Return(nil)

			scanner := sbom.SyftCLISBOMScanner{
				Executor: &executor,
				Layers:   layers,
				Logger:   bard.NewLogger(io.Discard),
			}

			Expect(scanner.ScanLaunch("something", libcnb.CycloneDXJSON, libcnb.SyftJSON, libcnb.SPDXJSON)).To(Succeed())

			result, err := ioutil.ReadFile(layers.LaunchSBOMPath(libcnb.CycloneDXJSON))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(result)).To(Equal("succeed1"))

			result, err = ioutil.ReadFile(layers.LaunchSBOMPath(libcnb.SyftJSON))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(result)).To(Equal("succeed2"))

			result, err = ioutil.ReadFile(layers.LaunchSBOMPath(libcnb.SPDXJSON))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(result)).To(Equal("succeed3"))
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
