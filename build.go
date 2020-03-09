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

package libpak

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/buildpacks/libcnb"
	"github.com/heroku/color"
	"github.com/imdario/mergo"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/internal"
)

// Build is called by the main function of a buildpack, for build.
func Build(f libcnb.BuildFunc, options ...libcnb.Option) {
	logger := bard.NewLogger(os.Stdout)

	libcnb.Build(
		func(context libcnb.BuildContext) (libcnb.BuildResult, error) {

			// TODO: Remove once pack supports bindings natively
			bindings, err := libcnb.NewBindingsFromEnvironment("CNB_BINDINGS")
			if err != nil {
				return libcnb.BuildResult{}, fmt.Errorf("unable to get bindings from $CNB_BINDINGS: %w", err)
			}
			if err = mergo.Merge(&context.Platform.Bindings, bindings); err != nil {
				return libcnb.BuildResult{}, fmt.Errorf("unable to merge bindings %+v and %+v: %w", context.Platform.Bindings, bindings, err)
			}

			result, err := f(context)
			if err != nil {
				return result, bard.IdentifiableError{
					Name:        context.Buildpack.Info.Name,
					Description: context.Buildpack.Info.Version,
					Err:         err,
				}
			}

			file := filepath.Join(context.Layers.Path, "*.toml")
			existing, err := filepath.Glob(file)
			if err != nil {
				return libcnb.BuildResult{}, fmt.Errorf("unable to list files in %s: %w", file, err)
			}

			var contrib []string
			for _, l := range result.Layers {
				contrib = append(contrib, l.Name())
			}

			var remove []string
			for _, e := range existing {
				if !strings.HasSuffix(e, "store.toml") && !contains(contrib, strings.TrimSuffix(filepath.Base(e), ".toml")) {
					remove = append(remove, e)
				}
			}

			if len(remove) > 0 {
				logger.Header("%s unused layers", color.YellowString("Removing"))

				for _, r := range remove {
					logger.Body(strings.TrimSuffix(filepath.Base(r), ".toml"))
					if err = os.RemoveAll(r); err != nil {
						return libcnb.BuildResult{}, fmt.Errorf("unable to remove file %s: %w", r, err)
					}
				}
			}

			return result, nil
		},
		append([]libcnb.Option{
			libcnb.WithEnvironmentWriter(internal.NewEnvironmentWriter()),
			libcnb.WithExitHandler(internal.NewExitHandler()),
			libcnb.WithTOMLWriter(internal.NewTOMLWriter()),
		}, options...)...,
	)
}

func contains(candidates []string, value string) bool {
	for _, c := range candidates {
		if c == value {
			return true
		}
	}

	return false
}
