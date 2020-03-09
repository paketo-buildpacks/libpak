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

package carton

import (
	"github.com/buildpacks/libcnb"
	"github.com/packeto-buildpacks/libpak/effect"
)

//go:generate mockery -name EntryWriter -case=underscore

// EntryWriter is the interface implemented by a type that wants to write an entry.
type EntryWriter interface {

	// Write reads a file from source and writes it to the relative path destination.
	Write(source string, destination string) error
}

// Config is an object that contains configurable properties for execution.
type Config struct {
	entryWriter EntryWriter
	executor    effect.Executor
	exitHandler libcnb.ExitHandler
}

// Option is a function for configuring a Config instance.
type Option func(config Config) Config

// WithEntryWriter creates an Option that sets an EntryWriter implementation.
func WithEntryWriter(entryWriter EntryWriter) Option {
	return func(config Config) Config {
		config.entryWriter = entryWriter
		return config
	}
}

// WithExecutor creates an Option that sets an Executor implementation.
func WithExecutor(executor effect.Executor) Option {
	return func(config Config) Config {
		config.executor = executor
		return config
	}
}

// WithExitHandler creates an Option that sets an ExitHandler implementation.
func WithExitHandler(exitHandler libcnb.ExitHandler) Option {
	return func(config Config) Config {
		config.exitHandler = exitHandler
		return config
	}
}
