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
	"strings"

	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/internal"
)

const (
	PackageIdDependencyPattern    = `(?m)(.*id[\s]+=[\s]+".+/%s",[\s]+version=")[^"]+(".*)`
	PackageImageDependencyPattern = `(?m)(.*uri[\s]+=[\s]+".*%s:)[^"]+(".*)`
	PackageDependencySubstitution = "${1}%s${2}"
)

type PackageDependency struct {
	BuilderPath   string
	BuildpackPath string
	ID            string
	Version       string
	PackagePath   string
}

func (p PackageDependency) Update(options ...Option) {
	config := Config{
		exitHandler: internal.NewExitHandler(),
	}

	for _, option := range options {
		config = option(config)
	}

	logger := bard.NewLogger(os.Stdout)
	_, _ = fmt.Fprintf(logger.TitleWriter(), "\n%s\n", bard.FormatIdentity(p.ID, p.Version))

	var paths []string
	if p.BuilderPath != "" {
		paths = append(paths, p.BuilderPath)
	}
	if p.PackagePath != "" {
		paths = append(paths, p.PackagePath)
	}

	for _, path := range paths {
		c, err := ioutil.ReadFile(path)
		if err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to read %s\n%w", path, err))
			return
		}

		s := fmt.Sprintf(PackageImageDependencyPattern, p.ID)
		r, err := regexp.Compile(s)
		if err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to compile regex %s\n%w", s, err))
			return
		}

		if !r.Match(c) {
			config.exitHandler.Error(fmt.Errorf("unable to match '%s'", s))
			return
		}

		s = fmt.Sprintf(PackageDependencySubstitution, p.Version)
		c = r.ReplaceAll(c, []byte(s))

		if err := ioutil.WriteFile(path, c, 0644); err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to write %s\n%w", path, err))
			return
		}
	}

	if p.BuildpackPath != "" {
		c, err := ioutil.ReadFile(p.BuildpackPath)
		if err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to read %s\n%w", p.BuildpackPath, err))
			return
		}

		id := strings.Join(strings.Split(p.ID, "/")[2:], "/")
		s := fmt.Sprintf(PackageIdDependencyPattern, id)
		r, err := regexp.Compile(s)
		if err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to compile regex %s\n%w", s, err))
			return
		}

		if !r.Match(c) {
			config.exitHandler.Error(fmt.Errorf("unable to match '%s'", s))
			return
		}

		s = fmt.Sprintf(PackageDependencySubstitution, p.Version)
		c = r.ReplaceAll(c, []byte(s))

		if err := ioutil.WriteFile(p.BuildpackPath, c, 0644); err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to write %s\n%w", p.BuildpackPath, err))
			return
		}
	}
}
