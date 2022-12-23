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

package carton_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb/mocks"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"

	"github.com/paketo-buildpacks/libpak/carton"
	cMocks "github.com/paketo-buildpacks/libpak/carton/mocks"
	"github.com/paketo-buildpacks/libpak/effect"
	eMocks "github.com/paketo-buildpacks/libpak/effect/mocks"
)

func testPackage(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		entryWriter *cMocks.EntryWriter
		executor    *eMocks.Executor
		exitHandler *mocks.ExitHandler
		path        string
	)

	it.Before(func() {
		var err error

		entryWriter = &cMocks.EntryWriter{}
		entryWriter.On("Write", mock.Anything, mock.Anything).Return(nil)

		executor = &eMocks.Executor{}
		executor.On("Execute", mock.Anything).Return(nil)

		exitHandler = &mocks.ExitHandler{}
		exitHandler.On("Error", mock.Anything)

		path, err = ioutil.TempDir("", "carton-package")
		Expect(err).NotTo(HaveOccurred())

		Expect(ioutil.WriteFile(filepath.Join(path, "buildpack.toml"), []byte(`
api = "0.0.0"

[buildpack]
name    = "test-name"
version = "{{.version}}"

[[metadata.dependencies]]
id      = "test-id"
name    = "test-name"
version = "1.1.1"
uri     = "test-uri"
sha256  = "test-sha256"
stacks  = [ "test-stack" ]

  [[metadata.dependencies.licenses]]
  type = "test-type"
  uri  = "test-uri"

[metadata]
pre-package   = "test-pre-package"
include-files = [
  "test-include-files",
  "buildpack.toml",
]
`), 0644)).To(Succeed())
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	it("executes pre_package script", func() {
		carton.Package{
			Source: path,
		}.Create(
			carton.WithEntryWriter(entryWriter),
			carton.WithExecutor(executor),
			carton.WithExitHandler(exitHandler))

		e, ok := executor.Calls[0].Arguments[0].(effect.Execution)
		Expect(ok).To(BeTrue())
		Expect(e.Command).To(Equal("test-pre-package"))
		Expect(e.Dir).To(Equal(path))
	})

	it("includes include_files", func() {
		carton.Package{
			Source:      path,
			Destination: "test-destination",
		}.Create(
			carton.WithEntryWriter(entryWriter),
			carton.WithExecutor(executor),
			carton.WithExitHandler(exitHandler))

		Expect(entryWriter.Calls[0].Arguments[0]).To(Equal(filepath.Join(path, "buildpack.toml")))
		Expect(entryWriter.Calls[0].Arguments[1]).To(Equal(filepath.Join("test-destination", "buildpack.toml")))
		Expect(entryWriter.Calls[1].Arguments[0]).To(Equal(filepath.Join(path, "test-include-files")))
		Expect(entryWriter.Calls[1].Arguments[1]).To(Equal(filepath.Join("test-destination", "test-include-files")))
	})

	it("replaces .version in buildpack.toml", func() {
		carton.Package{
			Source:      path,
			Destination: "test-destination",
			Version:     "2.2.2",
		}.Create(
			carton.WithEntryWriter(entryWriter),
			carton.WithExecutor(executor),
			carton.WithExitHandler(exitHandler))

		Expect(entryWriter.Calls[0].Arguments[0]).NotTo(Equal(filepath.Join(path, "buildpack.toml")))
		Expect(entryWriter.Calls[0].Arguments[1]).To(Equal(filepath.Join("test-destination", "buildpack.toml")))
		Expect(entryWriter.Calls[1].Arguments[0]).To(Equal(filepath.Join(path, "test-include-files")))
		Expect(entryWriter.Calls[1].Arguments[1]).To(Equal(filepath.Join("test-destination", "test-include-files")))

		Expect(ioutil.ReadFile(entryWriter.Calls[0].Arguments[0].(string))).To(Equal([]byte(`
api = "0.0.0"

[buildpack]
name    = "test-name"
version = "2.2.2"

[[metadata.dependencies]]
id      = "test-id"
name    = "test-name"
version = "1.1.1"
uri     = "test-uri"
sha256  = "test-sha256"
stacks  = [ "test-stack" ]

  [[metadata.dependencies.licenses]]
  type = "test-type"
  uri  = "test-uri"

[metadata]
pre-package   = "test-pre-package"
include-files = [
  "test-include-files",
  "buildpack.toml",
]
`)))
	})

	context("includes dependencies", func() {
		it.Before(func() {
			Expect(ioutil.WriteFile(filepath.Join(path, "buildpack.toml"), []byte(`
api = "0.0.0"

[buildpack]
name    = "test-name"
version = "{{.version}}"

[[metadata.dependencies]]
id      = "test-id"
name    = "test-name"
version = "1.1.1"
uri     = "test-uri-1"
sha256  = "test-sha256-1"

[[metadata.dependencies]]
id      = "test-id"
name    = "test-name"
version = "2.0.5"
uri     = "test-uri-2"
sha256  = "test-sha256-2"

[[metadata.dependencies]]
id      = "another-test-id"
name    = "test-name"
version = "1.1.1"
uri     = "test-uri-3"
sha256  = "test-sha256-3"

[metadata]
pre-package   = "test-pre-package"
include-files = [
  "test-include-files",
  "buildpack.toml",
]
`), 0644)).To(Succeed())
		})

		it("includes all dependencies", func() {
			carton.Package{
				Source:              path,
				Destination:         "test-destination",
				IncludeDependencies: true,
				CacheLocation:       "testdata",
			}.Create(
				carton.WithEntryWriter(entryWriter),
				carton.WithExecutor(executor),
				carton.WithExitHandler(exitHandler))

			Expect(entryWriter.Calls).To(HaveLen(8))
			Expect(entryWriter.Calls[0].Arguments[0]).To(Equal(filepath.Join(path, "buildpack.toml")))
			Expect(entryWriter.Calls[0].Arguments[1]).To(Equal(filepath.Join("test-destination", "buildpack.toml")))

			Expect(entryWriter.Calls[1].Arguments[0]).To(Equal("testdata/test-sha256-1.toml"))
			Expect(entryWriter.Calls[1].Arguments[1]).To(Equal(filepath.Join("test-destination", "dependencies/test-sha256-1.toml")))
			Expect(entryWriter.Calls[2].Arguments[0]).To(Equal("testdata/test-sha256-1/test-uri-1"))
			Expect(entryWriter.Calls[2].Arguments[1]).To(Equal(filepath.Join("test-destination", "dependencies/test-sha256-1/test-uri-1")))

			Expect(entryWriter.Calls[3].Arguments[0]).To(Equal("testdata/test-sha256-2.toml"))
			Expect(entryWriter.Calls[3].Arguments[1]).To(Equal(filepath.Join("test-destination", "dependencies/test-sha256-2.toml")))
			Expect(entryWriter.Calls[4].Arguments[0]).To(Equal("testdata/test-sha256-2/test-uri-2"))
			Expect(entryWriter.Calls[4].Arguments[1]).To(Equal(filepath.Join("test-destination", "dependencies/test-sha256-2/test-uri-2")))

			Expect(entryWriter.Calls[5].Arguments[0]).To(Equal("testdata/test-sha256-3.toml"))
			Expect(entryWriter.Calls[5].Arguments[1]).To(Equal(filepath.Join("test-destination", "dependencies/test-sha256-3.toml")))
			Expect(entryWriter.Calls[6].Arguments[0]).To(Equal("testdata/test-sha256-3/test-uri-3"))
			Expect(entryWriter.Calls[6].Arguments[1]).To(Equal(filepath.Join("test-destination", "dependencies/test-sha256-3/test-uri-3")))

			Expect(entryWriter.Calls[7].Arguments[0]).To(Equal(filepath.Join(path, "test-include-files")))
			Expect(entryWriter.Calls[7].Arguments[1]).To(Equal(filepath.Join("test-destination", "test-include-files")))
		})

		it("includes filter by id", func() {
			carton.Package{
				Source:              path,
				Destination:         "test-destination",
				IncludeDependencies: true,
				CacheLocation:       "testdata",
				DependencyFilters:   []string{`^another-test-id$`},
			}.Create(
				carton.WithEntryWriter(entryWriter),
				carton.WithExecutor(executor),
				carton.WithExitHandler(exitHandler))

			Expect(entryWriter.Calls).To(HaveLen(4))
			Expect(entryWriter.Calls[0].Arguments[0]).To(Equal(filepath.Join(path, "buildpack.toml")))
			Expect(entryWriter.Calls[0].Arguments[1]).To(Equal(filepath.Join("test-destination", "buildpack.toml")))

			Expect(entryWriter.Calls[1].Arguments[0]).To(Equal("testdata/test-sha256-3.toml"))
			Expect(entryWriter.Calls[1].Arguments[1]).To(Equal(filepath.Join("test-destination", "dependencies/test-sha256-3.toml")))
			Expect(entryWriter.Calls[2].Arguments[0]).To(Equal("testdata/test-sha256-3/test-uri-3"))
			Expect(entryWriter.Calls[2].Arguments[1]).To(Equal(filepath.Join("test-destination", "dependencies/test-sha256-3/test-uri-3")))

			Expect(entryWriter.Calls[3].Arguments[0]).To(Equal(filepath.Join(path, "test-include-files")))
			Expect(entryWriter.Calls[3].Arguments[1]).To(Equal(filepath.Join("test-destination", "test-include-files")))
		})

		it("includes filter by version", func() {
			carton.Package{
				Source:              path,
				Destination:         "test-destination",
				IncludeDependencies: true,
				CacheLocation:       "testdata",
				DependencyFilters:   []string{`^1.1.1$`},
			}.Create(
				carton.WithEntryWriter(entryWriter),
				carton.WithExecutor(executor),
				carton.WithExitHandler(exitHandler))

			Expect(entryWriter.Calls).To(HaveLen(6))
			Expect(entryWriter.Calls[0].Arguments[0]).To(Equal(filepath.Join(path, "buildpack.toml")))
			Expect(entryWriter.Calls[0].Arguments[1]).To(Equal(filepath.Join("test-destination", "buildpack.toml")))

			Expect(entryWriter.Calls[1].Arguments[0]).To(Equal("testdata/test-sha256-1.toml"))
			Expect(entryWriter.Calls[1].Arguments[1]).To(Equal(filepath.Join("test-destination", "dependencies/test-sha256-1.toml")))
			Expect(entryWriter.Calls[2].Arguments[0]).To(Equal("testdata/test-sha256-1/test-uri-1"))
			Expect(entryWriter.Calls[2].Arguments[1]).To(Equal(filepath.Join("test-destination", "dependencies/test-sha256-1/test-uri-1")))

			Expect(entryWriter.Calls[3].Arguments[0]).To(Equal("testdata/test-sha256-3.toml"))
			Expect(entryWriter.Calls[3].Arguments[1]).To(Equal(filepath.Join("test-destination", "dependencies/test-sha256-3.toml")))
			Expect(entryWriter.Calls[4].Arguments[0]).To(Equal("testdata/test-sha256-3/test-uri-3"))
			Expect(entryWriter.Calls[4].Arguments[1]).To(Equal(filepath.Join("test-destination", "dependencies/test-sha256-3/test-uri-3")))

			Expect(entryWriter.Calls[5].Arguments[0]).To(Equal(filepath.Join(path, "test-include-files")))
			Expect(entryWriter.Calls[5].Arguments[1]).To(Equal(filepath.Join("test-destination", "test-include-files")))
		})

		it("includes filter by version and id", func() {
			carton.Package{
				Source:                  path,
				Destination:             "test-destination",
				IncludeDependencies:     true,
				CacheLocation:           "testdata",
				DependencyFilters:       []string{`^test-id$|^2\.0\.5$`},
				StrictDependencyFilters: true,
			}.Create(
				carton.WithEntryWriter(entryWriter),
				carton.WithExecutor(executor),
				carton.WithExitHandler(exitHandler))

			Expect(entryWriter.Calls).To(HaveLen(4))
			Expect(entryWriter.Calls[0].Arguments[0]).To(Equal(filepath.Join(path, "buildpack.toml")))
			Expect(entryWriter.Calls[0].Arguments[1]).To(Equal(filepath.Join("test-destination", "buildpack.toml")))

			Expect(entryWriter.Calls[1].Arguments[0]).To(Equal("testdata/test-sha256-2.toml"))
			Expect(entryWriter.Calls[1].Arguments[1]).To(Equal(filepath.Join("test-destination", "dependencies/test-sha256-2.toml")))
			Expect(entryWriter.Calls[2].Arguments[0]).To(Equal("testdata/test-sha256-2/test-uri-2"))
			Expect(entryWriter.Calls[2].Arguments[1]).To(Equal(filepath.Join("test-destination", "dependencies/test-sha256-2/test-uri-2")))

			Expect(entryWriter.Calls[3].Arguments[0]).To(Equal(filepath.Join(path, "test-include-files")))
			Expect(entryWriter.Calls[3].Arguments[1]).To(Equal(filepath.Join("test-destination", "test-include-files")))
		})
	})
}
