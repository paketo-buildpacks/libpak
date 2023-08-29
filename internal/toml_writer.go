/*
 * Copyright 2018-2023 the original author or authors.
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
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/libcnb/v2"
	"github.com/heroku/color"
	"github.com/paketo-buildpacks/libpak/v2/log"
)

// TOMLWriter is an implementation of the libcnb.TOMLWriter interface.
type TOMLWriter struct {
	logger log.Logger
}

// TOMLWriterOption is a function for configuring a TOMLWriter instance.
type TOMLWriterOption func(writer TOMLWriter) TOMLWriter

// WithTOMLWriterLogger creates an TOMLWriterOption that configures the logger.
func WithTOMLWriterLogger(logger log.Logger) TOMLWriterOption {
	return func(writer TOMLWriter) TOMLWriter {
		writer.logger = logger
		return writer
	}
}

// NewTOMLWriter creates a new instance that writes to the filesystem and writes to the default log.Logger.
func NewTOMLWriter(options ...TOMLWriterOption) TOMLWriter {
	w := TOMLWriter{
		logger: log.NewPaketoLogger(os.Stdout),
	}

	for _, option := range options {
		w = option(w)
	}

	return w
}

// Write creates the path's parent directories, and creates a new file or truncates an existing file and then marshals
// the value to the file.
func (t TOMLWriter) Write(path string, value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case libcnb.LaunchTOML:
		if len(v.Slices) > 0 {
			t.logger.Headerf("%d application slices", len(v.Slices))
		}

		if len(v.Labels) > 0 {
			t.logger.Header("Image labels:")

			sort.Slice(v.Labels, func(i, j int) bool {
				return v.Labels[i].Key < v.Labels[j].Key
			})

			for _, l := range v.Labels {
				t.logger.Headerf("  %s", l.Key)
			}
		}

		if len(v.Processes) > 0 {
			t.logger.Header("Process types:")

			sort.Slice(v.Processes, func(i int, j int) bool {
				return v.Processes[i].Type < v.Processes[j].Type
			})

			max := t.maxTypeLength(v.Processes)
			for _, p := range v.Processes {
				sb := strings.Builder{}
				sb.WriteString(fmt.Sprintf("  %s: ", color.CyanString(p.Type)))

				for i := 0; i < max-len(p.Type); i++ {
					sb.WriteString(" ")
				}

				for i, c := range p.Command {
					if i != 0 {
						sb.WriteString(" ")
					}
					sb.WriteString(c)
				}

				for _, a := range p.Arguments {
					sb.WriteString(fmt.Sprintf(" %s", a))
				}

				t.logger.Header(sb.String())
			}
		}
	case libcnb.Store:
		if len(v.Metadata) > 0 {
			t.logger.Header("Persistent metadata:")

			var names []string
			for k := range v.Metadata {
				names = append(names, k)
			}

			sort.Strings(names)

			for _, n := range names {
				t.logger.Headerf("  %s", n)
			}
		}
	}

	d := filepath.Dir(path)
	if err := os.MkdirAll(d, 0755); err != nil {
		return fmt.Errorf("unable to mkdir %s\n%w", d, err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to open file %s\n%w", path, err)
	}
	defer file.Close()

	return toml.NewEncoder(file).Encode(value)
}

func (TOMLWriter) maxTypeLength(processes []libcnb.Process) int {
	max := 0

	for _, p := range processes {
		if l := len(p.Type); l > max {
			max = l
		}
	}

	return max
}
