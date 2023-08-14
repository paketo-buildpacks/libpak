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

package internal

import (
	"fmt"
	"io"
	"os"

	"github.com/paketo-buildpacks/libpak/v2/bard"
)

const (
	// ErrorStatusCode is the status code returned for error.
	ErrorStatusCode = 1

	// FailStatusCode is the status code returned for fail.
	FailStatusCode = 100

	// PassStatusCode is the status code returned for pass.
	PassStatusCode = 0
)

// ExitHandler is an implementation of the libcnb.ExitHandler interface.
type ExitHandler struct {
	exitFunc func(int)
	logger   bard.Logger
	writer   io.Writer
}

// ExitHandlerOption is a function for configuring an ExitHandler instance.
type ExitHandlerOption func(handler ExitHandler) ExitHandler

// WithExitHandler creates an ExitHandlerOption that configures the exit function.
func WithExitHandlerExitFunc(exitFunc func(int)) ExitHandlerOption {
	return func(handler ExitHandler) ExitHandler {
		handler.exitFunc = exitFunc
		return handler
	}
}

// WithExitHandlerLogger creates an ExitHandlerOption that configures the logger.
func WithExitHandlerLogger(logger bard.Logger) ExitHandlerOption {
	return func(handler ExitHandler) ExitHandler {
		handler.logger = logger
		return handler
	}
}

// WithExitHandlerWriter creates an ExitHandlerOption that configures the writer.
func WithExitHandlerWriter(writer io.Writer) ExitHandlerOption {
	return func(handler ExitHandler) ExitHandler {
		handler.writer = writer
		return handler
	}
}

// NewExitHandler creates a new instance that calls os.Exit and writes to the default bard.Logger and os.stderr.
func NewExitHandler(options ...ExitHandlerOption) ExitHandler {
	h := ExitHandler{
		exitFunc: os.Exit,
		logger:   bard.NewLogger(os.Stdout),
		writer:   os.Stderr,
	}

	for _, option := range options {
		h = option(h)
	}

	return h
}

func (e ExitHandler) Error(err error) {
	if i, ok := err.(bard.IdentifiableError); ok {
		e.logger.TerminalError(i)
	} else {
		_, _ = fmt.Fprintln(e.writer, err)
	}

	e.exitFunc(ErrorStatusCode)
}

func (e ExitHandler) Fail() {
	e.exitFunc(FailStatusCode)
}

func (e ExitHandler) Pass() {
	e.exitFunc(PassStatusCode)
}
