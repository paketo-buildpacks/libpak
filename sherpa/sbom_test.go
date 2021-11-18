package sherpa_test

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
	"github.com/paketo-buildpacks/libpak/sherpa"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"
)

func testSBOM(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layers   libcnb.Layers
		layer    libcnb.Layer
		executor mocks.Executor
		scanner  sherpa.SBOMScanner
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
		it("runs syft once to generate JSON", func() {
			format := libcnb.SyftJSON
			outputPath := layers.BuildSBOMPath(format)

			executor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "syft" &&
					len(e.Args) == 4 &&
					e.Args[2] == "json" &&
					e.Args[3] == "dir:something"
			})).Run(func(args mock.Arguments) {
				Expect(ioutil.WriteFile(outputPath, []byte("succeed1"), 0644)).To(Succeed())
			}).Return(nil)

			// uses interface here intentionally, to force that inteface and implementation match
			scanner = sherpa.NewSyftCLISBOMScanner(layers, &executor, bard.NewLogger(io.Discard))

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
					len(e.Args) == 4 &&
					e.Args[2] == "json" &&
					e.Args[3] == "dir:something"
			})).Run(func(args mock.Arguments) {
				Expect(ioutil.WriteFile(outputPath, []byte("succeed2"), 0644)).To(Succeed())
			}).Return(nil)

			scanner := sherpa.SyftCLISBOMScanner{
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
						len(e.Args) == 4 &&
						e.Args[2] == sherpa.SBOMFormatToSyftOutputFormat(format) &&
						e.Args[3] == "dir:something"
				})).Run(func(args mock.Arguments) {
					Expect(ioutil.WriteFile(outputPath, []byte("succeed3"), 0644)).To(Succeed())
				}).Return(nil)

				scanner := sherpa.SyftCLISBOMScanner{
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

			scanner := sherpa.SyftCLISBOMScanner{
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

			scanner := sherpa.SyftCLISBOMScanner{
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
	})

}
