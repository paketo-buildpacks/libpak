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
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

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

	c, err := ioutil.ReadFile(b.BuildpackPath)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to read %s\n%w", b.BuildpackPath, err))
		return
	}

	s := fmt.Sprintf(BuildpackDependencyPattern, b.ID, b.VersionPattern)
	r, err := regexp.Compile(s)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to compile regex %s\n%w", s, err))
		return
	}

	if !r.Match(c) {
		config.exitHandler.Error(fmt.Errorf("unable to match '%s' '%s'", b.ID, b.VersionPattern))
		return
	}

	s = fmt.Sprintf(BuildpackDependencySubstitution, b.Version, b.URI, b.SHA256)
	c = r.ReplaceAll(c, []byte(s))

	if err := ioutil.WriteFile(b.BuildpackPath, c, 0644); err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to write %s\n%w", b.BuildpackPath, err))
		return
	}
}
