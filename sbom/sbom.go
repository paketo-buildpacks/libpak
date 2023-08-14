package sbom

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/buildpacks/libcnb/v2"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/paketo-buildpacks/libpak/v2/bard"
	"github.com/paketo-buildpacks/libpak/v2/effect"
)

//go:generate mockery --name SBOMScanner --case=underscore

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
			Version: "0.32.0",
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

	err = os.WriteFile(path, output, 0644)
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
	args := []string{"packages", "-q"}

	for _, format := range formats {
		args = append(args, "-o", fmt.Sprintf("%s=%s", SBOMFormatToSyftOutputFormat(format), sbomPathCreator(format)))
	}

	args = append(args, fmt.Sprintf("dir:%s", scanDir))

	if err := b.Executor.Execute(effect.Execution{
		Command: "syft",
		Args:    args,
		Stdout:  b.Logger.TerminalErrorWriter(),
		Stderr:  b.Logger.TerminalErrorWriter(),
	}); err != nil {
		return fmt.Errorf("unable to run `syft %s`\n%w", args, err)
	}

	// cleans cyclonedx file which has a timestamp and unique id which always change
	for _, format := range formats {
		if format == libcnb.CycloneDXJSON {
			if err := b.makeCycloneDXReproducible(sbomPathCreator(format)); err != nil {
				return fmt.Errorf("unable to make cyclone dx file reproducible\n%w", err)
			}
		}
	}

	return nil
}

func (b SyftCLISBOMScanner) makeCycloneDXReproducible(path string) error {
	input, err := loadCycloneDXFile(path)
	if err != nil {
		return err
	}

	delete(input, "serialNumber")

	if md, exists := input["metadata"]; exists {
		if metadata, ok := md.(map[string]interface{}); ok {
			delete(metadata, "timestamp")
		}
	}

	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to open CycloneDX JSON for writing %s\n%w", path, err)
	}
	defer out.Close()

	if err := json.NewEncoder(out).Encode(input); err != nil {
		return fmt.Errorf("unable to encode CycloneDX\n%w", err)
	}

	return nil
}

func loadCycloneDXFile(path string) (map[string]interface{}, error) {
	in, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read CycloneDX JSON file %s\n%w", path, err)
	}
	defer in.Close()

	raw := map[string]interface{}{}
	if err := json.NewDecoder(in).Decode(&raw); err != nil {
		return nil, fmt.Errorf("unable to decode CycloneDX JSON %s\n%w", path, err)
	}

	return raw, nil
}

// SBOMFormatToSyftOutputFormat converts a libcnb.SBOMFormat to the syft matching syft output format string
func SBOMFormatToSyftOutputFormat(format libcnb.SBOMFormat) string {
	var formatRaw string

	switch format {
	case libcnb.CycloneDXJSON:
		formatRaw = "cyclonedx-json"
	case libcnb.SPDXJSON:
		formatRaw = "spdx-json"
	case libcnb.SyftJSON:
		formatRaw = "json"
	}

	return formatRaw
}
