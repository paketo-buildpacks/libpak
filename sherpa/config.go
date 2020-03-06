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
	"github.com/buildpacks/libcnb"
)

// Config is an object that contains configurable properties for execution.
type Config struct {
	exitHandler libcnb.ExitHandler
}

// Option is a function for configuring a Config instance.
type Option func(config Config) Config

// WithExitHandler creates an Option that sets an ExitHandler implementation.
func WithExitHandler(exitHandler libcnb.ExitHandler) Option {
	return func(config Config) Config {
		config.exitHandler = exitHandler
		return config
	}
}
