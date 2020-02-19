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
	"github.com/imdario/mergo"
	"github.com/paketoio/libpak/bard"
	"github.com/paketoio/libpak/internal"
)

// Build is called by the main function of a buildpack, for build.
func Build(f libcnb.BuildFunc, options ...libcnb.Option) {
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

			file := filepath.Join(context.Layers.Path, "*.toml")
			existing, err := filepath.Glob(file)
			if err != nil {
				return libcnb.BuildResult{}, fmt.Errorf("unable to list files in %s: %w", file, err)
			}
			for _, e := range existing {
				if !strings.HasSuffix(e, "store.toml") {
					if err = os.RemoveAll(e); err != nil {
						return libcnb.BuildResult{}, fmt.Errorf("unable to remove file %s: %w", e, err)
					}
				}
			}

			result, err := f(context)
			if err != nil {
				err = bard.IdentifiableError{
					Name:        context.Buildpack.Info.Name,
					Description: context.Buildpack.Info.Version,
					Err:         err,
				}
			}

			return result, err
		},
		append([]libcnb.Option{
			libcnb.WithEnvironmentWriter(internal.NewEnvironmentWriter()),
			libcnb.WithExitHandler(internal.NewExitHandler()),
			libcnb.WithTOMLWriter(internal.NewTOMLWriter()),
		}, options...)...,
	)
}
