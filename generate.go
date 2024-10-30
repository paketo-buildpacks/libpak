/*
 * Copyright 2023 the original author or authors.
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

// Generate is called by the main function of an extension, for generation.
func Generate(generator libcnb.GenerateFunc, options ...libcnb.Option) {
	libcnb.Generate(generateDelegate{delegate: generator}.Generate,
		libcnb.NewConfig(append([]libcnb.Option{
			libcnb.WithEnvironmentWriter(utils.NewEnvironmentWriter()),
			libcnb.WithExitHandler(utils.NewExitHandler()),
			libcnb.WithTOMLWriter(utils.NewTOMLWriter()),
		}, options...)...))
}

type generateDelegate struct {
	delegate libcnb.GenerateFunc
}

func (b generateDelegate) Generate(context libcnb.GenerateContext) (libcnb.GenerateResult, error) {
	result, err := b.delegate(context)
	if err != nil {
		err = log.IdentifiableError{
			Name:        context.Extension.Info.Name,
			Description: context.Extension.Info.Version,
			Err:         err,
		}
	}

	return result, err
}
