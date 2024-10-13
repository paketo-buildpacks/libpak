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

package libpak

import (
	"github.com/buildpacks/libcnb/v2"

	"github.com/paketo-buildpacks/libpak/v2/internal"
)

// Main is called by the main function of a buildpack, encapsulating both detection and build in the same binary.
func BuildpackMain(detect libcnb.DetectFunc, build libcnb.BuildFunc, options ...libcnb.Option) {
	libcnb.BuildpackMain(detectDelegate{delegate: detect}.Detect, buildDelegate{delegate: build}.Build,
		append([]libcnb.Option{
			libcnb.WithEnvironmentWriter(internal.NewEnvironmentWriter()),
			libcnb.WithExitHandler(internal.NewExitHandler()),
			libcnb.WithTOMLWriter(internal.NewTOMLWriter()),
		}, options...)...,
	)
}

// Main is called by the main function of an extension, encapsulating both detection and generation in the same binary.
func ExtensionMain(detect libcnb.DetectFunc, generate libcnb.GenerateFunc, options ...libcnb.Option) {
	libcnb.ExtensionMain(detectDelegate{delegate: detect}.Detect, generateDelegate{delegate: generate}.Generate,
		append([]libcnb.Option{
			libcnb.WithEnvironmentWriter(internal.NewEnvironmentWriter()),
			libcnb.WithExitHandler(internal.NewExitHandler()),
			libcnb.WithTOMLWriter(internal.NewTOMLWriter()),
		}, options...)...,
	)
}
