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

package internal

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/paketo-buildpacks/libpak/bard"
)

// EnvironmentWriter is an implementation of the libcnb.EnvironmentWriter interface.
type EnvironmentWriter struct {
	logger bard.Logger
}

// EnvironmentWriterOption is a function for configuring a EnvironmentWriter instance.
type EnvironmentWriterOption func(writer EnvironmentWriter) EnvironmentWriter

// WithEnvironmentWriterLogger creates an EnvironmentWriterOption that configures the logger.
func WithEnvironmentWriterLogger(logger bard.Logger) EnvironmentWriterOption {
	return func(writer EnvironmentWriter) EnvironmentWriter {
		writer.logger = logger
		return writer
	}
}

// NewEnvironmentWriter creates a new instance that writes to the filesystem and writes to the default bard.Logger.
func NewEnvironmentWriter(options ...EnvironmentWriterOption) EnvironmentWriter {
	w := EnvironmentWriter{
		logger: bard.NewLogger(os.Stdout),
	}

	for _, option := range options {
		w = option(w)
	}

	return w
}

// Write creates the path directory, and creates a new file for each key with the value as the contents of each file.
func (w EnvironmentWriter) Write(path string, environment map[string]string) error {
	if len(environment) == 0 {
		return nil
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("unable to mkdir %s\n%w", path, err)
	}

	var keys []string
	for k, _ := range environment {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	base := filepath.Base(path)
	for _, k := range keys {
		w.logger.Bodyf("Writing %s/%s", base, k)
		f := filepath.Join(path, k)
		if err := ioutil.WriteFile(f, []byte(environment[k]), 0644); err != nil {
			return fmt.Errorf("unable to write file %s\n%w", f, err)
		}
	}

	return nil
}
