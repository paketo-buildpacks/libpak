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

package internal_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak/v2/internal"
	"github.com/paketo-buildpacks/libpak/v2/log"
)

func testEnvironmentWriter(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		path   string
		writer internal.EnvironmentWriter
	)

	it.Before(func() {
		path = t.TempDir()
		Expect(os.RemoveAll(path)).To(Succeed())

		writer = internal.NewEnvironmentWriter()
	})

	it("writes the given environment to a directory", func() {
		err := writer.Write(path, map[string]string{
			"some-name":  "some-content",
			"other-name": "other-content",
		})
		Expect(err).NotTo(HaveOccurred())

		content, err := os.ReadFile(filepath.Join(path, "some-name"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(Equal("some-content"))

		content, err = os.ReadFile(filepath.Join(path, "other-name"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(Equal("other-content"))
	})

	it("writes the given environment with process specific envs to a directory", func() {
		err := writer.Write(path, map[string]string{
			"some-proc/some-name": "some-content",
		})
		Expect(err).NotTo(HaveOccurred())

		content, err := os.ReadFile(filepath.Join(path, "some-proc", "some-name"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(Equal("some-content"))
	})

	it("writes does not create a directory of the env map is empty", func() {
		err := writer.Write(path, map[string]string{})
		Expect(err).NotTo(HaveOccurred())

		Expect(path).NotTo(BeAnExistingFile())
	})

	context("Logging", func() {
		var (
			b *bytes.Buffer
		)

		it.Before(func() {
			b = bytes.NewBuffer(nil)
			writer = internal.NewEnvironmentWriter(internal.WithEnvironmentWriterLogger(log.NewPaketoLogger(b)))
		})

		it("logs environment", func() {
			err := writer.Write(filepath.Join(path, "env"), map[string]string{
				"some-name":  "some-content",
				"other-name": "other-content",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(b.String()).To(Equal("\x1b[2m    Writing env/other-name\x1b[0m\n\x1b[2m    Writing env/some-name\x1b[0m\n"))
		})
	})
}
