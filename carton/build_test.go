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
	"github.com/paketoio/libpak/carton"
	cMocks "github.com/paketoio/libpak/carton/mocks"
	"github.com/paketoio/libpak/effect"
	eMocks "github.com/paketoio/libpak/effect/mocks"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		build       carton.Build
		entryWriter *cMocks.EntryWriter
		executor    *eMocks.Executor
		exitHandler *mocks.ExitHandler
		path        string
	)

	it.Before(func() {
		var err error

		build = carton.Build{}

		entryWriter = &cMocks.EntryWriter{}
		entryWriter.On("Write", mock.Anything, mock.Anything).Return(nil)

		executor = &eMocks.Executor{}
		executor.On("Execute", mock.Anything).Return(nil)

		exitHandler = &mocks.ExitHandler{}
		exitHandler.On("Error", mock.Anything)

		path, err = ioutil.TempDir("", "carton-build")
		Expect(err).NotTo(HaveOccurred())

		Expect(ioutil.WriteFile(filepath.Join(path, "buildpack.toml"), []byte(`
api = "0.0.0"

[buildpack]
name    = "test-name"
version = "{{.Version}}"

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
pre_package   = "test-pre-package"
include_files = [
  "test-include-files",
  "buildpack.toml",
]
`), 0644)).To(Succeed())
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	it("executes pre_package script", func() {
		context := carton.Context{Source: path}

		build.Build(context,
			carton.WithEntryWriter(entryWriter),
			carton.WithExecutor(executor),
			carton.WithExitHandler(exitHandler))

		e, ok := executor.Calls[0].Arguments[0].(effect.Execution)
		Expect(ok).To(BeTrue())
		Expect(e.Command).To(Equal("test-pre-package"))
		Expect(e.Dir).To(Equal(path))
	})

	it("includes include_files", func() {
		context := carton.Context{Source: path, Destination: "test-destination"}

		build.Build(context,
			carton.WithEntryWriter(entryWriter),
			carton.WithExecutor(executor),
			carton.WithExitHandler(exitHandler))

		Expect(entryWriter.Calls[0].Arguments[0]).To(Equal(filepath.Join(path, "buildpack.toml")))
		Expect(entryWriter.Calls[0].Arguments[1]).To(Equal(filepath.Join("test-destination", "buildpack.toml")))
		Expect(entryWriter.Calls[1].Arguments[0]).To(Equal(filepath.Join(path, "test-include-files")))
		Expect(entryWriter.Calls[1].Arguments[1]).To(Equal(filepath.Join("test-destination", "test-include-files")))
	})

	it("replaces .Version in buildpack.toml", func() {
		context := carton.Context{Source: path, Destination: "test-destination", Version: "2.2.2"}

		build.Build(context,
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
pre_package   = "test-pre-package"
include_files = [
  "test-include-files",
  "buildpack.toml",
]
`)))
	})
}
