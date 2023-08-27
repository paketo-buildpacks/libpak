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

	"github.com/paketo-buildpacks/libpak/v2/internal"
	"github.com/paketo-buildpacks/libpak/v2/log"
)

const (
	LifecycleDependencyPattern      = `(?m)(.*\[lifecycle\]\nuri[\s]+=[\s]+")[^"]+(".*)`
	LifecycleDependencySubstitution = "${1}https://github.com/buildpacks/lifecycle/releases/download/v%[1]s/lifecycle-v%[1]s+linux.x86-64.tgz${2}"
)

type LifecycleDependency struct {
	BuilderPath string
	Version     string
}

func (l LifecycleDependency) Update(options ...Option) {
	config := Config{
		exitHandler: internal.NewExitHandler(),
	}

	for _, option := range options {
		config = option(config)
	}

	logger := log.NewLogger(os.Stdout)
	_, _ = fmt.Fprintf(logger.TitleWriter(), "\n%s\n", log.FormatIdentity("Lifecycle", l.Version))

	c, err := os.ReadFile(l.BuilderPath)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to read %s\n%w", l.BuilderPath, err))
		return
	}

	r := regexp.MustCompile(LifecycleDependencyPattern)

	if !r.Match(c) {
		config.exitHandler.Error(fmt.Errorf("unable to match '%s'", LifecycleDependencyPattern))
		return
	}

	s := fmt.Sprintf(LifecycleDependencySubstitution, l.Version)
	c = r.ReplaceAll(c, []byte(s))

	if err := os.WriteFile(l.BuilderPath, c, 0644); err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to write %s\n%w", l.BuilderPath, err))
		return
	}

}
