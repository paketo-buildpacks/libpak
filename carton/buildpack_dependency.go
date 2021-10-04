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
	"io/ioutil"
	"os"
	"regexp"

	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/internal"
	"github.com/pelletier/go-toml"
)

const (
	BuildpackDependencyPattern      = `(?m)([\s]*.*id[\s]+=[\s]+"%s"\n.*\n[\s]*version[\s]+=[\s]+")%s("\n[\s]*uri[\s]+=[\s]+").*("\n[\s]*sha256[\s]+=[\s]+").*(".*)`
	BuildpackDependencySubstitution = "${1}%s${2}%s${3}%s${4}"
)

type BuildpackDependency struct {
	BuildpackPath  string
	ID             string
	SHA256         string
	URI            string
	Version        string
	VersionPattern string
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
	logger.Headerf("Version: %s", b.Version)
	logger.Headerf("URI:     %s", b.URI)
	logger.Headerf("SHA256:  %s", b.SHA256)

	versionExp, err := regexp.Compile(b.VersionPattern)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to compile regex %s\n%w", b.VersionPattern, err))
		return
	}

	c, err := ioutil.ReadFile(b.BuildpackPath)
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
			}
		}
	}

	c, err = toml.Marshal(md)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to encode md %s\n%w", b.BuildpackPath, err))
		return
	}

	c = append(comments, c...)

	if err := ioutil.WriteFile(b.BuildpackPath, c, 0644); err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to write %s\n%w", b.BuildpackPath, err))
		return
	}
}
