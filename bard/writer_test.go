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

package bard_test

import (
	"bytes"
	"testing"

	"github.com/heroku/color"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/sclevine/spec"
)

func testWriter(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	context("Writer", func() {
		var (
			buffer *bytes.Buffer
			writer bard.Writer
		)

		it.Before(func() {
			buffer = bytes.NewBuffer(nil)
			writer = bard.NewWriter(buffer)
		})

		context("Write", func() {
			it("prints to the writer", func() {
				_, err := writer.Write([]byte("some-text"))
				Expect(err).NotTo(HaveOccurred())
				Expect(buffer.String()).To(Equal("some-text"))
			})

			context("when the writer has a color", func() {
				it.Before(func() {
					writer = bard.NewWriter(buffer, bard.WithAttributes(color.FgBlue))
				})

				it("prints to the writer with the correct color codes", func() {
					_, err := writer.Write([]byte("some-text"))
					Expect(err).NotTo(HaveOccurred())
					Expect(buffer.String()).To(Equal("\x1b[34msome-text\x1b[0m"))
				})
			})

			context("when the writer has an indent", func() {
				it.Before(func() {
					writer = bard.NewWriter(buffer, bard.WithIndent(2))
				})

				it("prints to the writer with the correct indentation", func() {
					_, err := writer.Write([]byte("some-text\nother-text"))
					Expect(err).NotTo(HaveOccurred())
					Expect(buffer.String()).To(Equal("    some-text\n    other-text"))
				})
			})

			context("when the writer has a return prefix", func() {
				it.Before(func() {
					writer = bard.NewWriter(buffer, bard.WithAttributes(color.FgRed), bard.WithIndent(2))
				})

				it("prints to the writer with the correct indentation", func() {
					_, err := writer.Write([]byte("\rsome-text"))
					Expect(err).NotTo(HaveOccurred())
					Expect(buffer.String()).To(Equal("\r\x1b[31m    some-text\x1b[0m"))
				})
			})

			context("when the writer has a newline suffix", func() {
				it.Before(func() {
					writer = bard.NewWriter(buffer, bard.WithAttributes(color.FgRed), bard.WithIndent(2))
				})

				it("prints to the writer with the correct indentation", func() {
					_, err := writer.Write([]byte("some-text\n"))
					Expect(err).NotTo(HaveOccurred())
					Expect(buffer.String()).To(Equal("\x1b[31m    some-text\x1b[0m\n"))
				})
			})

			context("when the input has a percent symbol", func() {
				it.Before(func() {
					writer = bard.NewWriter(buffer, bard.WithAttributes(color.FgMagenta))
				})

				it("prints to the writer with the correct indentation", func() {
					_, err := writer.Write([]byte("some-%"))
					Expect(err).NotTo(HaveOccurred())
					Expect(buffer.String()).To(Equal("\x1b[35msome-%\x1b[0m"))
				})
			})
		})
	})
}
