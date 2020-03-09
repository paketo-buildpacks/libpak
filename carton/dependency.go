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
	DependencyPattern      = `(?m)(.*id[\s]+=[\s]+"%s"\n.*\nversion[\s]+=[\s]+")%s("\nuri[\s]+=[\s]+").*("\nsha256[\s]+=[\s]+").*(".*)`
	DependencySubstitution = "${1}%s${2}%s${3}%s${4}"
)

type Dependency struct {
	BuildpackPath  string
	ID             string
	SHA256         string
	URI            string
	Version        string
	VersionPattern string
}

func (d Dependency) Build(options ...Option) {
	config := Config{
		exitHandler: internal.NewExitHandler(),
	}

	for _, option := range options {
		config = option(config)
	}

	logger := bard.NewLogger(os.Stdout)
	_, _ = fmt.Fprintf(logger.TitleWriter(), "\n%s\n", bard.IdentityFormatter{Name: d.ID, Description: d.VersionPattern})
	logger.Header("Version: %s", d.Version)
	logger.Header("URI:     %s", d.URI)
	logger.Header("SHA256:  %s", d.SHA256)

	c, err := ioutil.ReadFile(d.BuildpackPath)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to read %s: %w", d.BuildpackPath, err))
		return
	}

	s := fmt.Sprintf(DependencyPattern, d.ID, d.VersionPattern)
	r, err := regexp.Compile(s)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to compile regex %s: %w", s, err))
		return
	}

	if !r.Match(c) {
		config.exitHandler.Error(fmt.Errorf("unable to match '%s' '%s'", d.ID, d.VersionPattern))
		return
	}

	s = fmt.Sprintf(DependencySubstitution, d.Version, d.URI, d.SHA256)
	c = r.ReplaceAll(c, []byte(s))

	if err := ioutil.WriteFile(d.BuildpackPath, c, 0644); err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to write %s: %w", d.BuildpackPath, err))
		return
	}
}
