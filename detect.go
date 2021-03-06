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
	"github.com/buildpacks/libcnb"

	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/internal"
)

// Detect is called by the main function of a buildpack, for detection.
func Detect(detector libcnb.Detector, options ...libcnb.Option) {
	libcnb.Detect(detectDelegate{delegate: detector},
		append([]libcnb.Option{
			libcnb.WithExitHandler(internal.NewExitHandler()),
			libcnb.WithTOMLWriter(internal.NewTOMLWriter()),
		}, options...)...,
	)
}

type detectDelegate struct {
	delegate libcnb.Detector
}

func (d detectDelegate) Detect(context libcnb.DetectContext) (libcnb.DetectResult, error) {
	result, err := d.delegate.Detect(context)
	if err != nil {
		err = bard.IdentifiableError{
			Name:        context.Buildpack.Info.Name,
			Description: context.Buildpack.Info.Version,
			Err:         err,
		}
	}

	return result, err
}
