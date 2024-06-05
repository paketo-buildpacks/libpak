/*
 * Copyright 2018-2020 the original author or authors.
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

package carton

import (
	"bytes"
	"fmt"
	"os"
	"regexp"

	"github.com/BurntSushi/toml"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/internal"
)

const (
	BuildpackDependencyPattern      = `(?m)([\s]*.*id[\s]+=[\s]+"%s"\n.*\n[\s]*version[\s]+=[\s]+")%s("\n[\s]*uri[\s]+=[\s]+").*("\n[\s]*sha256[\s]+=[\s]+").*(".*)`
	BuildpackDependencySubstitution = "${1}%s${2}%s${3}%s${4}"
)

type BuildpackDependency struct {
	BuildpackPath  string
	ID             string
	EolID          string
	Arch           string
	SHA256         string
	URI            string
	Version        string
	VersionPattern string
	CPE            string
	CPEPattern     string
	PURL           string
	PURLPattern    string
	Source         string `toml:"source,omitempty"`
	SourceSHA256   string `toml:"source-sha256,omitempty"`
}

func (b BuildpackDependency) Update(options ...Option) {
	config := Config{
		exitHandler: internal.NewExitHandler(),
	}

	for _, option := range options {
		config = option(config)
	}

	logger := bard.NewLogger(os.Stdout)
	_, _ = fmt.Fprintf(logger.TitleWriter(), "\n%s\n", bard.FormatIdentity(b.ID, b.VersionPattern))
	logger.Headerf("Arch:         %s", b.Arch)
	logger.Headerf("Version:      %s", b.Version)
	logger.Headerf("PURL:         %s", b.PURL)
	logger.Headerf("CPEs:         %s", b.CPE)
	logger.Headerf("URI:          %s", b.URI)
	logger.Headerf("SHA256:       %s", b.SHA256)
	logger.Headerf("Source:       %s", b.Source)
	logger.Headerf("SourceSHA256: %s", b.SourceSHA256)
	logger.Headerf("EOL ID:       %s", b.EolID)

	versionExp, err := regexp.Compile(b.VersionPattern)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to compile version regex %s\n%w", b.VersionPattern, err))
		return
	}

	cpeExp, err := regexp.Compile(b.CPEPattern)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to compile cpe regex %s\n%w", b.CPEPattern, err))
		return
	}

	purlExp, err := regexp.Compile(b.PURLPattern)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to compile cpe regex %s\n%w", b.PURLPattern, err))
		return
	}

	c, err := os.ReadFile(b.BuildpackPath)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to read %s\n%w", b.BuildpackPath, err))
		return
	}

	// save any leading comments, this is to preserve license headers
	// inline comments will be lost
	comments := []byte{}
	for i, line := range bytes.SplitAfter(c, []byte("\n")) {
		if bytes.HasPrefix(line, []byte("#")) || (i > 0 && len(bytes.TrimSpace(line)) == 0) {
			comments = append(comments, line...)
		} else {
			break // stop on first comment
		}
	}

	md := make(map[string]interface{})
	if err := toml.Unmarshal(c, &md); err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to decode md%s\n%w", b.BuildpackPath, err))
		return
	}

	metadataUnwrapped, found := md["metadata"]
	if !found {
		config.exitHandler.Error(fmt.Errorf("unable to find metadata block"))
		return
	}

	metadata, ok := metadataUnwrapped.(map[string]interface{})
	if !ok {
		config.exitHandler.Error(fmt.Errorf("unable to cast metadata"))
		return
	}

	dependenciesUnwrapped, found := metadata["dependencies"]
	if !found {
		config.exitHandler.Error(fmt.Errorf("unable to find dependencies block"))
		return
	}

	dependencies, ok := dependenciesUnwrapped.([]map[string]interface{})
	if !ok {
		config.exitHandler.Error(fmt.Errorf("unable to cast dependencies"))
		return
	}

	for _, dep := range dependencies {
		depIdUnwrapped, found := dep["id"]
		if !found {
			continue
		}
		depId, ok := depIdUnwrapped.(string)
		if !ok {
			continue
		}

		// extract the arch from the PURL, it's the only place it lives consistently at the moment
		var depArch string
		purlUnwrapped, found := dep["purl"]
		if found {
			purl, ok := purlUnwrapped.(string)
			if ok {
				purlArchExp := regexp.MustCompile(`arch=(.*)`)
				purlArchMatches := purlArchExp.FindStringSubmatch(purl)
				if len(purlArchMatches) == 2 {
					depArch = purlArchMatches[1]
				}
			}
		}

		// if not set, we presently need to default to amd64 because a lot of deps do not specify arch
		//   in the future when we add the arch field to our deps, then we can remove this because empty should then mean noarch
		if depArch == "" {
			depArch = "amd64"
		}

		if depId == b.ID && depArch == b.Arch {

			depVersionUnwrapped, found := dep["version"]
			if !found {
				continue
			}

			depVersion, ok := depVersionUnwrapped.(string)
			if !ok {
				continue
			}

			if versionExp.MatchString(depVersion) {
				dep["version"] = b.Version
				dep["uri"] = b.URI
				dep["sha256"] = b.SHA256
				if b.SourceSHA256 != "" {
					dep["source-sha256"] = b.SourceSHA256
				}
				if b.Source != "" {
					dep["source"] = b.Source
				}

				purlUnwrapped, found := dep["purl"]
				if found {
					purl, ok := purlUnwrapped.(string)
					if ok {
						dep["purl"] = purlExp.ReplaceAllString(purl, b.PURL)
					}
				}

				cpesUnwrapped, found := dep["cpes"]
				if found {
					cpes, ok := cpesUnwrapped.([]interface{})
					if ok {
						for i := 0; i < len(cpes); i++ {
							cpe, ok := cpes[i].(string)
							if !ok {
								continue
							}

							cpes[i] = cpeExp.ReplaceAllString(cpe, b.CPE)
						}
					}
				}

				if b.EolID != "" {
					eolDate, err := GetEolDate(b.EolID, b.Version)
					if err != nil {
						config.exitHandler.Error(fmt.Errorf("unable to fetch deprecation_date"))
						return
					}

					if eolDate != "" {
						dep["deprecation_date"] = eolDate
					}
				}
			}
		}
	}

	c, err = internal.Marshal(md)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to encode md %s\n%w", b.BuildpackPath, err))
		return
	}

	c = append(comments, c...)

	if err := os.WriteFile(b.BuildpackPath, c, 0644); err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to write %s\n%w", b.BuildpackPath, err))
		return
	}
}
