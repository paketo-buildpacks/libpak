package sbom_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buildpacks/libcnb/v2"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"

	"github.com/paketo-buildpacks/libpak/v2/effect"
	"github.com/paketo-buildpacks/libpak/v2/effect/mocks"
	"github.com/paketo-buildpacks/libpak/v2/log"
	"github.com/paketo-buildpacks/libpak/v2/sbom"
)

func testSBOM(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layers   libcnb.Layers
		layer    libcnb.Layer
		executor mocks.Executor
		scanner  sbom.Scanner
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
		it("runs syft once to generate SPDX JSON", func() {
			format := libcnb.SPDXJSON
			outputPath := layers.BuildSBOMPath(format)

			executor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "syft" &&
					len(e.Args) == 5 &&
					strings.HasPrefix(e.Args[3], "spdx-json=") &&
					e.Args[4] == "dir:something"
			})).Run(func(_ mock.Arguments) {
				Expect(os.WriteFile(outputPath, []byte("succeed1"), 0600)).To(Succeed())
			}).Return(nil)

			// uses interface here intentionally, to force that inteface and implementation match
			scanner = sbom.NewSyftCLISBOMScanner(layers, &executor, log.NewDiscardLogger())

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
			})).Run(func(_ mock.Arguments) {
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
}`), 0600)).To(Succeed())
			}).Return(nil)

			// uses interface here intentionally, to force that inteface and implementation match
			scanner = sbom.NewSyftCLISBOMScanner(layers, &executor, log.NewDiscardLogger())

			Expect(scanner.ScanBuild("something", format)).To(Succeed())

			result, err := os.ReadFile(outputPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(result)).ToNot(ContainSubstring("serialNumber"))
			Expect(string(result)).ToNot(ContainSubstring("urn:uuid:fcfa5e19-bf49-47b4-8c85-ab61e2728f8e"))
			Expect(string(result)).ToNot(ContainSubstring("timestamp"))
			Expect(string(result)).ToNot(ContainSubstring("2022-05-05T11:33:13-04:00"))
		})

		it("runs syft once to generate layer-specific SPDX JSON", func() {
			format := libcnb.SPDXJSON
			outputPath := layer.SBOMPath(format)

			executor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "syft" &&
					len(e.Args) == 5 &&
					strings.HasPrefix(e.Args[3], "spdx-json=") &&
					e.Args[4] == "dir:something"
			})).Run(func(_ mock.Arguments) {
				Expect(os.WriteFile(outputPath, []byte("succeed2"), 0600)).To(Succeed())
			}).Return(nil)

			scanner := sbom.SyftCLISBOMScanner{
				Executor: &executor,
				Layers:   layers,
				Logger:   log.NewDiscardLogger(),
			}

			Expect(scanner.ScanLayer(layer, "something", format)).To(Succeed())

			result, err := os.ReadFile(outputPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(result)).To(Equal("succeed2"))
		})

		it("runs syft once for CycloneDX and SPDX, and no longer writes Syft JSON", func() {
			executor.On("Execute", mock.MatchedBy(func(e effect.Execution) bool {
				return e.Command == "syft" &&
					len(e.Args) == 7 &&
					strings.HasPrefix(e.Args[3], sbom.FormatToSyftOutputFormat(libcnb.CycloneDXJSON)) &&
					strings.HasPrefix(e.Args[5], sbom.FormatToSyftOutputFormat(libcnb.SPDXJSON)) &&
					e.Args[6] == "dir:something"
			})).Run(func(_ mock.Arguments) {
				Expect(os.WriteFile(layers.LaunchSBOMPath(libcnb.CycloneDXJSON), []byte(`{"succeed":1}`), 0600)).To(Succeed())
				Expect(os.WriteFile(layers.LaunchSBOMPath(libcnb.SPDXJSON), []byte(`{"succeed":3}`), 0600)).To(Succeed())
			}).Return(nil)

			scanner := sbom.SyftCLISBOMScanner{
				Executor: &executor,
				Layers:   layers,
				Logger:   log.NewDiscardLogger(),
			}

			Expect(scanner.ScanLaunch("something", libcnb.CycloneDXJSON, libcnb.SyftJSON, libcnb.SPDXJSON)).To(Succeed())

			result, err := os.ReadFile(layers.LaunchSBOMPath(libcnb.CycloneDXJSON))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(result)).To(HavePrefix(`{"succeed":1}`))

			result, err = os.ReadFile(layers.LaunchSBOMPath(libcnb.SPDXJSON))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(result)).To(HavePrefix(`{"succeed":3}`))

			Expect(layers.LaunchSBOMPath(libcnb.SyftJSON)).NotTo(BeAnExistingFile())
		})

		it("does not run syft when only Syft JSON is asked for", func() {
			scanner := sbom.SyftCLISBOMScanner{
				Executor: &executor,
				Layers:   layers,
				Logger:   log.NewDiscardLogger(),
			}

			Expect(scanner.ScanLaunch("something", libcnb.SyftJSON)).To(Succeed())

			Expect(executor.Calls).To(BeEmpty())
			Expect(layers.LaunchSBOMPath(libcnb.SyftJSON)).NotTo(BeAnExistingFile())
		})
	})
}
