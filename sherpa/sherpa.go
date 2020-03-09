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

package sherpa

import (
	"github.com/packeto-buildpacks/libpak/internal"
)

// ExecuteFunc is the callback function for buildpack helper application implementations.
type ExecuteFunc func() error

// Execute is called by the main function of a buildpack helper application, for execution.
func Execute(f ExecuteFunc, options ...Option) {
	config := Config{
		exitHandler: internal.NewExitHandler(),
	}

	for _, option := range options {
		config = option(config)
	}

	if err := f(); err != nil {
		config.exitHandler.Error(err)
		return
	}
}
