/*
 * Copyright 2018-2025 the original author or authors.
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

package sherpa

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/paketo-buildpacks/libpak/v2/utils"
)

// ExecuteFunc is the callback function for buildpack helper application implementations.
type ExecuteFunc func() error

// Execute is called by the main function of a buildpack helper application, for execution.
func Execute(f ExecuteFunc, options ...Option) {
	config := Config{
		exitHandler: utils.NewExitHandler(),
	}

	for _, option := range options {
		config = option(config)
	}

	if err := f(); err != nil {
		config.exitHandler.Error(err)
		return
	}
}

//go:generate mockery --name ExecD --case=underscore

// ExecD describes an interface for types that Exec.d specification.
type ExecD interface {
	Execute() (map[string]string, error)
}

// Helpers is called by the main function of a buildpack's helper application, for execution.
func Helpers(helpers map[string]ExecD, options ...Option) error {
	config := Config{
		arguments:   os.Args,
		execdWriter: os.NewFile(3, "/dev/fd/3"),
	}

	for _, option := range options {
		config = option(config)
	}

	if len(config.arguments) == 0 {
		return fmt.Errorf("expected command name")
	}

	c := filepath.Base(config.arguments[0])
	e, ok := helpers[c]
	if !ok {
		return fmt.Errorf("unsupported command %s", c)
	}

	r, err := e.Execute()
	if err != nil {
		return err
	}

	// TODO: Enable once exec.d is implemented
	// if err := toml.NewEncoder(config.execdWriter).Encode(r); err != nil {
	// 	return fmt.Errorf("unable to write environment\n%w", err)
	// }

	// TODO: Remove once exec.d is implemented
	for k, v := range r {
		if _, err := fmt.Fprintf(config.execdWriter, "%s=%s\n", k, strconv.Quote(v)); err != nil {
			return fmt.Errorf("unable to write environment\n%w", err)
		}
	}

	return nil
}
