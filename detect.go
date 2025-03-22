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

package libpak

import (
	"github.com/buildpacks/libcnb/v2"

	"github.com/paketo-buildpacks/libpak/v2/log"
	"github.com/paketo-buildpacks/libpak/v2/utils"
)

// Detect is called by the main function of a buildpack, for detection.
func Detect(detector libcnb.DetectFunc, options ...libcnb.Option) {
	libcnb.Detect(detectDelegate{delegate: detector}.Detect,
		libcnb.NewConfig(append([]libcnb.Option{
			libcnb.WithExitHandler(utils.NewExitHandler()),
			libcnb.WithTOMLWriter(utils.NewTOMLWriter()),
		}, options...)...))
}

type detectDelegate struct {
	delegate libcnb.DetectFunc
}

func (d detectDelegate) Detect(context libcnb.DetectContext) (libcnb.DetectResult, error) {
	result, err := d.delegate(context)
	if err != nil {
		err = log.IdentifiableError{
			Name:        context.Buildpack.Info.Name,
			Description: context.Buildpack.Info.Version,
			Err:         err,
		}
	}

	return result, err
}
