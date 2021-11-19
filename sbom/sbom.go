package sbom

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/CycloneDX/cyclonedx-go"
	"github.com/buildpacks/libcnb"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/effect"
)

//go:generate mockery -name SBOMScanner -case=underscore

type SBOMScanner interface {
	ScanLayer(layer libcnb.Layer, scanDir string, formats ...libcnb.SBOMFormat) error
	ScanBuild(scanDir string, formats ...libcnb.SBOMFormat) error
	ScanLaunch(scanDir string, formats ...libcnb.SBOMFormat) error
}

type SyftDependency struct {
	Artifacts  []SyftArtifact
	Source     SyftSource
	Descriptor SyftDescriptor
	Schema     SyftSchema
}

func NewSyftDependency(dependencyPath string, artifacts []SyftArtifact) SyftDependency {
	return SyftDependency{
		Artifacts: artifacts,
		Source: SyftSource{
			Type:   "directory",
			Target: dependencyPath,
		},
		Descriptor: SyftDescriptor{
			Name:    "syft",
			Version: "0.30.1",
		},
		Schema: SyftSchema{
			Version: "1.1.0",
			URL:     "https://raw.githubusercontent.com/anchore/syft/main/schema/json/schema-1.1.0.json",
		},
	}
}

func (s SyftDependency) WriteTo(path string) error {
	output, err := json.Marshal(&s)
	if err != nil {
		return fmt.Errorf("unable to marshal to JSON\n%w", err)
	}

	err = ioutil.WriteFile(path, output, 0644)
	if err != nil {
		return fmt.Errorf("unable to write to path %s\n%w", path, err)
	}

	return nil
}

type SyftArtifact struct {
	ID        string
	Name      string
	Version   string
	Type      string
	FoundBy   string
	Locations []SyftLocation
	Licenses  []string
	Language  string
	CPEs      []string
	PURL      string
}

func (s SyftArtifact) Hash() (string, error) {
	f, err := hashstructure.Hash(s, hashstructure.FormatV2, &hashstructure.HashOptions{
		ZeroNil:      true,
		SlicesAsSets: true,
	})
	if err != nil {
		return "", fmt.Errorf("could not build ID for artifact=%+v: %+v", s, err)
	}

	return fmt.Sprintf("%x", f), nil
}

type SyftLocation struct {
	Path string
}

type SyftSource struct {
	Type   string
	Target string
}

type SyftDescriptor struct {
	Name    string
	Version string
}

type SyftSchema struct {
	Version string
	URL     string
}

type SyftCLISBOMScanner struct {
	Executor effect.Executor
	Layers   libcnb.Layers
	Logger   bard.Logger
}

func NewSyftCLISBOMScanner(layers libcnb.Layers, executor effect.Executor, logger bard.Logger) SyftCLISBOMScanner {
	return SyftCLISBOMScanner{
		Executor: executor,
		Layers:   layers,
		Logger:   logger,
	}
}

// ScanLayer will use syft CLI to scan the scanDir and write it's output to the layer SBoM file in the given formats
func (b SyftCLISBOMScanner) ScanLayer(layer libcnb.Layer, scanDir string, formats ...libcnb.SBOMFormat) error {
	return b.scan(func(fmt libcnb.SBOMFormat) string {
		return layer.SBOMPath(fmt)
	}, scanDir, formats...)
}

// ScanBuild will use syft CLI to scan the scanDir and write it's output to the build SBoM file in the given formats
func (b SyftCLISBOMScanner) ScanBuild(scanDir string, formats ...libcnb.SBOMFormat) error {
	return b.scan(func(fmt libcnb.SBOMFormat) string {
		return b.Layers.BuildSBOMPath(fmt)
	}, scanDir, formats...)
}

// ScanLaunch will use syft CLI to scan the scanDir and write it's output to the launch SBoM file in the given formats
func (b SyftCLISBOMScanner) ScanLaunch(scanDir string, formats ...libcnb.SBOMFormat) error {
	return b.scan(func(fmt libcnb.SBOMFormat) string {
		return b.Layers.LaunchSBOMPath(fmt)
	}, scanDir, formats...)
}

func (b SyftCLISBOMScanner) scan(sbomPathCreator func(libcnb.SBOMFormat) string, scanDir string, formats ...libcnb.SBOMFormat) error {
	// syft doesn't presently support outputting multiple formats at once
	// to workaround this we are running syft multiple times
	// when syft supports multiple output formats or conversion between formats, this method should change
	for _, format := range formats {
		sbomLocation := sbomPathCreator(format)

		if err := b.runSyft(sbomLocation, scanDir, format); err != nil {
			return fmt.Errorf("unable to run syft\n%w", err)
		}

		if format == libcnb.CycloneDXJSON {
			// syft doesn't presently support cyclonedx JSON output and we need to convert
			// until https://github.com/anchore/syft/issues/631 is addressed
			if err := b.ConvertCycloneDXXMLtoJSON(sbomLocation, false); err != nil {
				return fmt.Errorf("unable convert XML to JSON\n%w", err)
			}
		}
	}

	return nil
}

// ConvertCycloneDXXMLtoJSON reads input CycloneDX XML, converts to JSON and overwrites the XML optionally keeping a backup copy of the xml
func (b SyftCLISBOMScanner) ConvertCycloneDXXMLtoJSON(inputPath string, backup bool) error {
	if backup {
		if err := b.backupXMLFile(inputPath); err != nil {
			return fmt.Errorf("unable to backup file\n%w", err)
		}
	}

	bom, err := b.readXMLSBOM(inputPath)
	if err != nil {
		return fmt.Errorf("unable to read XML file for conversion\n%w", err)
	}

	if err := b.writeJSONSBOM(inputPath, bom); err != nil {
		return fmt.Errorf("unable to write converted JSON BOM file\n%w", err)
	}

	return nil
}

func (b SyftCLISBOMScanner) writeJSONSBOM(outputPath string, bom cyclonedx.BOM) error {
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("unable to create BOM file %s\n%w", outputPath, err)
	}
	defer outputFile.Close()

	decoder := cyclonedx.NewBOMEncoder(outputFile, cyclonedx.BOMFileFormatJSON)
	if err = decoder.Encode(&bom); err != nil {
		return fmt.Errorf("unable to decode BOM\n%w", err)
	}

	return nil
}

func (b SyftCLISBOMScanner) readXMLSBOM(inputPath string) (cyclonedx.BOM, error) {
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return cyclonedx.BOM{}, fmt.Errorf("unable to read file to convert %s\n%w", inputPath, err)
	}
	defer inputFile.Close()

	var bom cyclonedx.BOM
	decoder := cyclonedx.NewBOMDecoder(inputFile, cyclonedx.BOMFileFormatXML)
	if err = decoder.Decode(&bom); err != nil {
		return cyclonedx.BOM{}, fmt.Errorf("unable to decode BOM\n%w", err)
	}

	return bom, nil
}

func (b SyftCLISBOMScanner) backupXMLFile(inputPath string) error {
	backupPath := fmt.Sprintf("%s.bak", inputPath)
	outputFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("unable to create backup file %s\n%w", backupPath, err)
	}
	defer outputFile.Close()

	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("unable to read file for backup %s\n%w", inputPath, err)
	}
	defer inputFile.Close()

	_, err = io.Copy(outputFile, inputFile)
	return err
}

func (b SyftCLISBOMScanner) runSyft(sbomOutputPath string, scanDir string, format libcnb.SBOMFormat) error {
	writer, err := os.Create(sbomOutputPath)
	if err != nil {
		return fmt.Errorf("unable to open output BOM file %s\n%w", sbomOutputPath, err)
	}
	defer writer.Close()

	err = b.Executor.Execute(effect.Execution{
		Command: "syft",
		Args:    []string{"packages", "-q", "-o", SBOMFormatToSyftOutputFormat(format), fmt.Sprintf("dir:%s", scanDir)},
		Stdout:  writer,
		Stderr:  b.Logger.TerminalErrorWriter(),
	})
	if err != nil {
		return fmt.Errorf("unable to run syft on directory %s\n%w", scanDir, err)
	}

	return nil
}

// SBOMFormatToSyftOutputFormat converts a libcnb.SBOMFormat to the syft matching syft output format string
func SBOMFormatToSyftOutputFormat(format libcnb.SBOMFormat) string {
	var formatRaw string

	switch format {
	case libcnb.CycloneDXJSON:
		formatRaw = "cyclonedx"
	case libcnb.SPDXJSON:
		formatRaw = "spdx-json"
	case libcnb.SyftJSON:
		formatRaw = "json"
	}

	return formatRaw
}
