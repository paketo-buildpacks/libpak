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

package log

import (
	"bytes"
	"io"
	"strings"

	"github.com/heroku/color"
)

const (
	escape     = "\x1b["
	endCode    = "m"
	delimiter  = ";"
	colorReset = "\x1b[0m"
)

// Writer is an object that will indent and color all output flowing through it.
type Writer struct {
	code         string
	color        *color.Color
	indent       int
	shouldIndent bool
	writer       io.Writer
}

// NewWriter creates a instance that wraps another writer.
func NewWriter(writer io.Writer, options ...WriterOption) *Writer {
	w := Writer{writer: writer, shouldIndent: true}
	for _, option := range options {
		w = option(w)
	}

	return &w
}

func (w *Writer) Write(b []byte) (int, error) {
	var (
		prefix, suffix []byte
		reset          = []byte("\r")
		newline        = []byte("\n")
		n              = len(b)
	)

	if bytes.HasPrefix(b, reset) {
		b = bytes.TrimPrefix(b, reset)
		prefix = reset
	}

	if bytes.HasSuffix(b, newline) {
		b = bytes.TrimSuffix(b, newline)
		suffix = newline
	}

	lines := bytes.Split(b, newline)

	var indentedLines [][]byte
	for i, line := range lines {
		if w.shouldIndent || i > 0 {
			for i := 0; i < w.indent; i++ {
				line = append([]byte("  "), line...)
			}
			w.shouldIndent = false
		}

		if w.color != nil {
			s := string(line)
			s = strings.ReplaceAll(s, colorReset, colorReset+w.code)
			line = []byte(w.color.Sprint(s))
		}

		indentedLines = append(indentedLines, line)
	}

	b = bytes.Join(indentedLines, newline)

	if prefix != nil {
		b = append(prefix, b...)
	}

	if suffix != nil {
		b = append(b, suffix...)
	}

	if bytes.HasSuffix(b, newline) {
		w.shouldIndent = true
	}

	if _, err := w.writer.Write(b); err != nil {
		return n, err
	}

	return n, nil
}

// WriterOption is a function for configuring a Writer instance.
type WriterOption func(Writer) Writer

// WithAttributes creates an WriterOption that sets the output color.
func WithAttributes(attributes ...color.Attribute) WriterOption {
	return func(l Writer) Writer {
		l.code = chainSGRCodes(attributes)
		l.color = color.New(attributes...)
		return l
	}
}

// WithIndent creates an WriterOption that sets the depth of the output indent.
func WithIndent(indent int) WriterOption {
	return func(l Writer) Writer {
		l.indent = indent
		return l
	}
}

func chainSGRCodes(a []color.Attribute) string {
	codes := toCodes(a)

	if len(codes) == 0 {
		return colorReset
	}

	if len(codes) == 1 {
		return escape + codes[0] + endCode
	}

	var b strings.Builder
	b.Grow((len(codes) * 2) + len(escape) + len(endCode))
	b.WriteString(escape)

	delimsAdded := 0
	for i := 0; i < len(a); i++ {
		if delimsAdded > 0 {
			_, _ = b.WriteString(delimiter)
		}
		b.WriteString(codes[i])
		delimsAdded++
	}

	b.WriteString(endCode)

	return b.String()
}

func toCodes(attrs []color.Attribute) []string {
	var codes []string

	for _, a := range attrs {
		codes = append(codes, a.String())
	}

	return codes
}
