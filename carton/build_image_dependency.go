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
	"os"
	"regexp"

	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/internal"
)

const (
	ImageDependencyPattern      = `(?m)(.*build-image[\s]+=[\s]+"[^"]+:)[^"]+(".*)`
	ImageDependencySubstitution = "${1}%s${2}"
)

type BuildImageDependency struct {
	BuilderPath string
	Version     string
}

func (i BuildImageDependency) Update(options ...Option) {
	config := Config{
		exitHandler: internal.NewExitHandler(),
	}

	for _, option := range options {
		config = option(config)
	}

	logger := bard.NewLogger(os.Stdout)
	_, _ = fmt.Fprintf(logger.TitleWriter(), "\n%s\n", bard.FormatIdentity("Build Image", i.Version))

	c, err := os.ReadFile(i.BuilderPath)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to read %s\n%w", i.BuilderPath, err))
		return
	}

	r := regexp.MustCompile(ImageDependencyPattern)

	if !r.Match(c) {
		config.exitHandler.Error(fmt.Errorf("unable to match '%s'", r.String()))
		return
	}

	s := fmt.Sprintf(ImageDependencySubstitution, i.Version)
	c = r.ReplaceAll(c, []byte(s))

	if err := os.WriteFile(i.BuilderPath, c, 0644); err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to write %s\n%w", i.BuilderPath, err))
		return
	}

}
