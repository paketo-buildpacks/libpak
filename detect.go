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

	"github.com/buildpacks/libcnb"
	"github.com/imdario/mergo"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/internal"
)

// Detect is called by the main function of a buildpack, for detection.
func Detect(f libcnb.DetectFunc, options ...libcnb.Option) {
	libcnb.Detect(
		func(context libcnb.DetectContext) (libcnb.DetectResult, error) {

			// TODO: Remove once pack supports bindings natively
			bindings, err := libcnb.NewBindingsFromEnvironment("CNB_BINDINGS")
			if err != nil {
				return libcnb.DetectResult{}, fmt.Errorf("unable to get bindings from $CNB_BINDINGS: %w", err)
			}
			if err := mergo.Merge(&context.Platform.Bindings, bindings); err != nil {
				return libcnb.DetectResult{}, fmt.Errorf("unable to merge bindings %+v and %+v: %w", context.Platform.Bindings, bindings, err)
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
			libcnb.WithExitHandler(internal.NewExitHandler()),
			libcnb.WithTOMLWriter(internal.NewTOMLWriter()),
		}, options...)...,
	)

}
