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

package bard

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/buildpacks/libcnb"
	"github.com/buildpacks/libcnb/poet"
	"github.com/heroku/color"
)

// TODO: Remove once TTY support is in place
func init() {
	color.Enabled()
}

// Logger logs message to a writer.
type Logger struct {
	poet.Logger

	body           io.Writer
	header         io.Writer
	terminalBody   io.Writer
	terminalHeader io.Writer
	title          io.Writer
}

// Option is a function for configuring a Logger instance.
type Option func(logger Logger) Logger

// WithDebug configures the debug Writer.
func WithDebug(writer io.Writer) Option {
	return func(logger Logger) Logger {
		logger.Logger = poet.WithDebug(writer)(logger.Logger)
		return logger
	}
}

// NewLoggerWithOptions create a new instance of Logger.  It configures the Logger with options.
func NewLoggerWithOptions(writer io.Writer, options ...Option) Logger {
	l := Logger{
		Logger:         poet.NewLogger(writer),
		body:           NewWriter(writer, WithAttributes(color.Faint), WithIndent(2)),
		header:         NewWriter(writer, WithIndent(1)),
		terminalBody:   NewWriter(writer, WithAttributes(color.FgRed, color.Bold), WithIndent(1)),
		terminalHeader: NewWriter(writer, WithAttributes(color.FgRed)),
		title:          NewWriter(writer, WithAttributes(color.FgBlue)),
	}

	for _, option := range options {
		l = option(l)
	}

	return l
}

// NewLogger creates a new instance of Logger.  It configures debug logging if $BP_DEBUG is set.
func NewLogger(writer io.Writer) Logger {
	var options []Option

	// check for presence and value of log level environment variable
	options = LogLevel(options, writer)

	return NewLoggerWithOptions(writer, options...)
}

func LogLevel(options []Option, writer io.Writer) []Option {

	// Check for older log level env variable
	_, dbSet := os.LookupEnv("BP_DEBUG")

	// Then check for common buildpack log level env variable - if either are set to DEBUG/true, enable Debug Writer
	if level, ok := os.LookupEnv("BP_LOG_LEVEL"); (ok && strings.ToLower(level) == "debug") || dbSet {

		options = append(options, WithDebug(writer))
	}
	return options
}

// Body formats using the default formats for its operands and logs a message to the configured body writer. Spaces
// are added between operands when neither is a string.
func (l Logger) Body(a ...interface{}) {
	if !l.IsBodyEnabled() {
		return
	}

	l.print(l.body, a...)
}

// Bodyf formats according to a format specifier and logs a message to the configured body writer.
func (l Logger) Bodyf(format string, a ...interface{}) {
	if !l.IsBodyEnabled() {
		return
	}

	l.printf(l.body, format, a...)
}

// BodyWriter returns the configured body writer.
func (l Logger) BodyWriter() io.Writer {
	return l.body
}

// IsBodyEnabled indicates whether body logging is enabled.
func (l Logger) IsBodyEnabled() bool {
	return l.body != nil
}

// Header formats using the default formats for its operands and logs a message to the configured header writer. Spaces
// are added between operands when neither is a string.
func (l Logger) Header(a ...interface{}) {
	if !l.IsHeaderEnabled() {
		return
	}

	l.print(l.header, a...)
}

// Headerf formats according to a format specifier and logs a message to the configured header writer.
func (l Logger) Headerf(format string, a ...interface{}) {
	if !l.IsHeaderEnabled() {
		return
	}

	l.printf(l.header, format, a...)
}

// HeaderWriter returns the configured header writer.
func (l Logger) HeaderWriter() io.Writer {
	return l.header
}

// IsHeaderEnabled indicates whether header logging is enabled.
func (l Logger) IsHeaderEnabled() bool {
	return l.header != nil
}

// IdentifiableError is an error associated with an Identifiable for logging purposes.
type IdentifiableError struct {

	// Name is the name of the identified object.
	Name string

	// Description is the description of the identified object.
	Description string

	// Err is the nested error.
	Err error
}

func (i IdentifiableError) Error() string {
	return i.Err.Error()
}

// TerminalError logs a message to the configured terminal error writer.
func (l Logger) TerminalError(err IdentifiableError) {
	if !l.IsTerminalErrorEnabled() {
		return
	}

	l.printf(l.terminalHeader, "\n%s", FormatIdentity(err.Name, err.Description))
	l.print(l.terminalBody, err.Err)
}

// TerminalErrorWriter returns the configured terminal error writer.
func (l Logger) TerminalErrorWriter() io.Writer {
	return l.terminalBody
}

// IsTerminalErrorEnabled indicates whether terminal error logging is enabled.
func (l Logger) IsTerminalErrorEnabled() bool {
	return l.terminalHeader != nil && l.terminalBody != nil
}

// Title logs a message to the configured title writer.
func (l Logger) Title(buildpack libcnb.Buildpack) {
	if !l.IsTitleEnabled() {
		return
	}

	l.printf(l.title, "\n%s", FormatIdentity(buildpack.Info.Name, buildpack.Info.Version))
	l.Header(color.New(color.FgBlue, color.Faint, color.Italic).Sprint(buildpack.Info.Homepage))
}

// TitleWriter returns the configured title writer.
func (l Logger) TitleWriter() io.Writer {
	return l.title
}

// IsTitleEnabled indicates whether title logging is enabled.
func (l Logger) IsTitleEnabled() bool {
	return l.title != nil
}

func (Logger) print(writer io.Writer, a ...interface{}) {
	s := fmt.Sprint(a...)

	if !strings.HasSuffix(s, "\n") {
		s = s + "\n"
	}

	_, _ = fmt.Fprint(writer, s)
}

func (Logger) printf(writer io.Writer, format string, a ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format = format + "\n"
	}

	_, _ = fmt.Fprintf(writer, format, a...)
}
