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
	"github.com/paketo-buildpacks/libpak/v2/bard"
	"github.com/paketo-buildpacks/libpak/v2/internal"
)

const (
	BuildModuleDependencyPattern      = `(?m)([\s]*.*id[\s]+=[\s]+"%s"\n.*\n[\s]*version[\s]+=[\s]+")%s("\n[\s]*uri[\s]+=[\s]+").*("\n[\s]*sha256[\s]+=[\s]+").*(".*)`
	BuildModuleDependencySubstitution = "${1}%s${2}%s${3}%s${4}"
)

type BuildModuleDependency struct {
	BuildModulePath string
	ID              string
	SHA256          string
	URI             string
	Version         string
	VersionPattern  string
	CPE             string
	CPEPattern      string
	PURL            string
	PURLPattern     string
}

func (b BuildModuleDependency) Update(options ...Option) {
	config := Config{
		exitHandler: internal.NewExitHandler(),
	}

	for _, option := range options {
		config = option(config)
	}

	logger := bard.NewLogger(os.Stdout)
	_, _ = fmt.Fprintf(logger.TitleWriter(), "\n%s\n", bard.FormatIdentity(b.ID, b.VersionPattern))
	logger.Headerf("Version: %s", b.Version)
	logger.Headerf("PURL:    %s", b.PURL)
	logger.Headerf("CPEs:    %s", b.CPE)
	logger.Headerf("URI:     %s", b.URI)
	logger.Headerf("SHA256:  %s", b.SHA256)

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

	c, err := os.ReadFile(b.BuildModulePath)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to read %s\n%w", b.BuildModulePath, err))
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
		config.exitHandler.Error(fmt.Errorf("unable to decode md%s\n%w", b.BuildModulePath, err))
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

		if depId == b.ID {
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
			}
		}
	}

	c, err = internal.Marshal(md)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to encode md %s\n%w", b.BuildModulePath, err))
		return
	}

	c = append(comments, c...)

	if err := os.WriteFile(b.BuildModulePath, c, 0644); err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to write %s\n%w", b.BuildModulePath, err))
		return
	}
}
