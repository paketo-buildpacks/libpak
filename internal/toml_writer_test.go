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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	"github.com/heroku/color"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/internal"
)

func testTOMLWriter(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		parent     string
		path       string
		tomlWriter internal.TOMLWriter
	)

	it.Before(func() {
		parent = t.TempDir()
		path = filepath.Join(parent, "text.toml")
	})

	it("writes the contents of a given object out to a .toml file", func() {
		err := tomlWriter.Write(path, map[string]string{
			"some-field":  "some-value",
			"other-field": "other-value",
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(os.ReadFile(path)).To(internal.MatchTOML(`
some-field = "some-value"
other-field = "other-value"`))
	})

	context("Logging", func() {
		var (
			b *bytes.Buffer
		)

		it.Before(func() {
			b = bytes.NewBuffer(nil)
			tomlWriter = internal.NewTOMLWriter(internal.WithTOMLWriterLogger(bard.NewLogger(b)))
		})

		it("does not log for uninteresting types", func() {
			err := tomlWriter.Write(path, map[string]string{
				"some-field":  "some-value",
				"other-field": "other-value",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(b.String()).To(Equal(""))
		})

		it("logs []libcnb.Slice", func() {
			err := tomlWriter.Write(path, libcnb.LaunchTOML{
				Slices: []libcnb.Slice{
					{},
					{},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(b.String()).To(Equal(`  2 application slices
`))
		})

		it("logs []libcnb.Label", func() {
			err := tomlWriter.Write(path, libcnb.LaunchTOML{
				Labels: []libcnb.Label{
					{Key: "test-key-1"},
					{Key: "test-key-2"},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(b.String()).To(Equal(`  Image labels:
    test-key-1
    test-key-2
`))
		})

		context("[]libcnb.Process", func() {

			it("aligns process types", func() {
				err := tomlWriter.Write(path, libcnb.LaunchTOML{
					Processes: []libcnb.Process{
						{Type: "short", Command: "test-command-1"},
						{Type: "a-very-long-type", Command: "test-command-2"},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(b.String()).To(Equal(fmt.Sprintf(`  Process types:
    %s: test-command-2
    %s:            test-command-1
`,
					color.CyanString("a-very-long-type"), color.CyanString("short"))))
			})

			it("appends arguments", func() {
				err := tomlWriter.Write(path, libcnb.LaunchTOML{
					Processes: []libcnb.Process{
						{Type: "test-type", Command: "test-command", Arguments: []string{"test-arg-1", "test-arg-2"}},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(b.String()).To(Equal(fmt.Sprintf(`  Process types:
    %s: test-command test-arg-1 test-arg-2
`,
					color.CyanString("test-type"))))
			})

			it("indicates direct", func() {
				err := tomlWriter.Write(path, libcnb.LaunchTOML{
					Processes: []libcnb.Process{
						{Type: "test-type", Command: "test-command", Direct: true},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(b.String()).To(Equal(fmt.Sprintf(`  Process types:
    %s: test-command (direct)
`,
					color.CyanString("test-type"))))
			})
		})

		it("logs libcnb.Store", func() {
			err := tomlWriter.Write(path, libcnb.Store{
				Metadata: map[string]interface{}{
					"test-key-1": "test-value-1",
					"test-key-2": "test-value-2",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(b.String()).To(Equal(`  Persistent metadata:
    test-key-1
    test-key-2
`))
		})
	})
}
