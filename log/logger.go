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

package log

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/buildpacks/libcnb/v2/log"
	"github.com/heroku/color"
)

// TODO: Remove once TTY support is in place
func init() {
	color.Enabled()
}

type Logger interface {
	log.Logger

	Body(a ...interface{})
	Bodyf(format string, a ...interface{})
	BodyWriter() io.Writer
	IsBodyEnabled() bool

	Header(a ...interface{})
	Headerf(format string, a ...interface{})
	HeaderWriter() io.Writer
	IsHeaderEnabled() bool

	TerminalError(err IdentifiableError)
	TerminalErrorWriter() io.Writer
	IsTerminalErrorEnabled() bool

	Title(name string, version string, homepage string)
	Titlef(format string, a ...interface{})
	TitleWriter() io.Writer
	IsTitleEnabled() bool
}

// Logger logs message to a writer.
type PaketoLogger struct {
	debug          io.Writer
	body           io.Writer
	header         io.Writer
	terminalBody   io.Writer
	terminalHeader io.Writer
	title          io.Writer
}

// NewDiscardLogger creates a new instance of PaketoLogger that discards all log messages. Useful in testing.
func NewDiscardLogger() PaketoLogger {
	return PaketoLogger{
		debug:          io.Discard,
		body:           io.Discard,
		header:         io.Discard,
		terminalBody:   io.Discard,
		terminalHeader: io.Discard,
		title:          io.Discard,
	}
}

// NewPaketoLogger creates a new instance of PaketoLogger.  It configures debug logging if $BP_DEBUG is set.
func NewPaketoLogger(writer io.Writer) PaketoLogger {
	var options []Option
	return NewPaketoLoggerWithOptions(writer, options...)
}

// Option is a function for configuring a Logger instance.
type Option func(logger PaketoLogger) PaketoLogger

func NewPaketoLoggerWithOptions(writer io.Writer, options ...Option) PaketoLogger {
	var debugWriter io.Writer
	if strings.ToLower(os.Getenv("BP_LOG_LEVEL")) == "debug" || os.Getenv("BP_DEBUG") != "" {
		debugWriter = NewWriter(writer, WithAttributes(color.BgCyan))
	}

	l := PaketoLogger{
		debug:          debugWriter,
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

// Body formats using the default formats for its operands and logs a message to the configured body writer. Spaces
// are added between operands when neither is a string.
func (l PaketoLogger) Body(a ...interface{}) {
	if !l.IsBodyEnabled() {
		return
	}

	l.print(l.body, a...)
}

// Bodyf formats according to a format specifier and logs a message to the configured body writer.
func (l PaketoLogger) Bodyf(format string, a ...interface{}) {
	if !l.IsBodyEnabled() {
		return
	}

	l.printf(l.body, format, a...)
}

// BodyWriter returns the configured body writer.
func (l PaketoLogger) BodyWriter() io.Writer {
	return l.body
}

// IsBodyEnabled indicates whether body logging is enabled.
func (l PaketoLogger) IsBodyEnabled() bool {
	return l.body != nil
}

// Header formats using the default formats for its operands and logs a message to the configured header writer. Spaces
// are added between operands when neither is a string.
func (l PaketoLogger) Header(a ...interface{}) {
	if !l.IsHeaderEnabled() {
		return
	}

	l.print(l.header, a...)
}

// Headerf formats according to a format specifier and logs a message to the configured header writer.
func (l PaketoLogger) Headerf(format string, a ...interface{}) {
	if !l.IsHeaderEnabled() {
		return
	}

	l.printf(l.header, format, a...)
}

// HeaderWriter returns the configured header writer.
func (l PaketoLogger) HeaderWriter() io.Writer {
	return l.header
}

// IsHeaderEnabled indicates whether header logging is enabled.
func (l PaketoLogger) IsHeaderEnabled() bool {
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
func (l PaketoLogger) TerminalError(err IdentifiableError) {
	if !l.IsTerminalErrorEnabled() {
		return
	}

	l.printf(l.terminalHeader, "\n%s", FormatIdentity(err.Name, err.Description))
	l.print(l.terminalBody, err.Err)
}

// TerminalErrorWriter returns the configured terminal error writer.
func (l PaketoLogger) TerminalErrorWriter() io.Writer {
	return l.terminalBody
}

// IsTerminalErrorEnabled indicates whether terminal error logging is enabled.
func (l PaketoLogger) IsTerminalErrorEnabled() bool {
	return l.terminalHeader != nil && l.terminalBody != nil
}

func (l PaketoLogger) Title(name string, version string, homepage string) {
	if !l.IsTitleEnabled() {
		return
	}

	l.printf(l.title, "\n%s", FormatIdentity(name, version))
	l.Header(color.New(color.FgBlue, color.Faint, color.Italic).Sprint(homepage))
}

func (l PaketoLogger) Titlef(format string, a ...interface{}) {
	if !l.IsTitleEnabled() {
		return
	}

	l.printf(l.title, format, a...)
}

// TitleWriter returns the configured title writer.
func (l PaketoLogger) TitleWriter() io.Writer {
	return l.title
}

// IsTitleEnabled indicates whether title logging is enabled.
func (l PaketoLogger) IsTitleEnabled() bool {
	return l.title != nil
}

// Debug formats using the default formats for its operands and writes to the configured debug writer. Spaces are added
// between operands when neither is a string.
func (l PaketoLogger) Debug(a ...interface{}) {
	if !l.IsDebugEnabled() {
		return
	}

	s := fmt.Sprint(a...)

	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}

	_, _ = fmt.Fprint(l.debug, s)
}

// Debugf formats according to a format specifier and writes to the configured debug writer.
func (l PaketoLogger) Debugf(format string, a ...interface{}) {
	if !l.IsDebugEnabled() {
		return
	}

	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}

	_, _ = fmt.Fprintf(l.debug, format, a...)
}

// DebugWriter returns the configured debug writer.
func (l PaketoLogger) DebugWriter() io.Writer {
	if l.IsDebugEnabled() {
		return l.debug
	}
	return io.Discard
}

// IsDebugEnabled indicates whether debug logging is enabled.
func (l PaketoLogger) IsDebugEnabled() bool {
	return l.debug != nil
}

func (PaketoLogger) print(writer io.Writer, a ...interface{}) {
	s := fmt.Sprint(a...)

	if !strings.HasSuffix(s, "\n") {
		s = s + "\n"
	}

	_, _ = fmt.Fprint(writer, s)
}

func (PaketoLogger) printf(writer io.Writer, format string, a ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format = format + "\n"
	}

	_, _ = fmt.Fprintf(writer, format, a...)
}
