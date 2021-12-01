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
	"testing"

	"github.com/buildpacks/libcnb/mocks"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"

	"github.com/paketo-buildpacks/libpak/carton"
	"github.com/paketo-buildpacks/libpak/internal"
)

func testBuildpackDependency(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		exitHandler *mocks.ExitHandler
		path        string
	)

	it.Before(func() {
		var err error

		exitHandler = &mocks.ExitHandler{}
		exitHandler.On("Error", mock.Anything)

		f, err := ioutil.TempFile("", "carton-buildpack-dependency")
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).To(Succeed())
		path = f.Name()
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	it("updates dependency", func() {
		Expect(ioutil.WriteFile(path, []byte(`api = "0.6"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
id      = "test-id"
name    = "Test Name"
version = "test-version-1"
uri     = "test-uri-1"
sha256  = "test-sha256-1"
stacks  = [ "test-stack" ]
`), 0644)).To(Succeed())

		d := carton.BuildpackDependency{
			BuildpackPath:  path,
			ID:             "test-id",
			SHA256:         "test-sha256-2",
			URI:            "test-uri-2",
			Version:        "test-version-2",
			VersionPattern: `test-version-[\d]`,
		}

		d.Update(carton.WithExitHandler(exitHandler))

		Expect(ioutil.ReadFile(path)).To(internal.MatchTOML(`api = "0.6"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]id      = "test-id"
name    = "Test Name"
version = "test-version-2"
uri     = "test-uri-2"
sha256  = "test-sha256-2"
stacks  = [ "test-stack" ]
`))
	})

	it("updates dependency with purl & cpes", func() {
		Expect(ioutil.WriteFile(path, []byte(`api = "0.7"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
id      = "test-id"
name    = "Test Name"
version = "test-version-1"
uri     = "test-uri-1"
sha256  = "test-sha256-1"
stacks  = [ "test-stack" ]
purl    = "pkg:generic/test-jre@different-version-1?arch=amd64"
cpes    = ["cpe:2.3:a:test-vendor:test-product:test-version-1:patch1:*:*:*:*:*:*:*"]
`), 0644)).To(Succeed())

		d := carton.BuildpackDependency{
			BuildpackPath:  path,
			ID:             "test-id",
			SHA256:         "test-sha256-2",
			URI:            "test-uri-2",
			Version:        "test-version-2",
			VersionPattern: `test-version-[\d]`,
			PURL:           "different-version-2",
			PURLPattern:    `different-version-[\d]`,
			CPE:            "test-version-2:patch2",
			CPEPattern:     `test-version-[\d]:patch[\d]`,
		}

		d.Update(carton.WithExitHandler(exitHandler))

		Expect(ioutil.ReadFile(path)).To(internal.MatchTOML(`api = "0.7"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]id      = "test-id"
name    = "Test Name"
version = "test-version-2"
uri     = "test-uri-2"
sha256  = "test-sha256-2"
stacks  = [ "test-stack" ]
purl    = "pkg:generic/test-jre@different-version-2?arch=amd64"
cpes    = ["cpe:2.3:a:test-vendor:test-product:test-version-2:patch2:*:*:*:*:*:*:*"]
`))
	})

	it("updates indented dependency", func() {
		Expect(ioutil.WriteFile(path, []byte(`# it should preserve
#   these comments
#      exactly

api = "0.6"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
  id      = "test-id"
  name    = "Test Name"
  version = "test-version-1"
  uri     = "test-uri-1"
  sha256  = "test-sha256-1"
  stacks  = [ "test-stack" ]
`), 0644)).To(Succeed())

		d := carton.BuildpackDependency{
			BuildpackPath:  path,
			ID:             "test-id",
			SHA256:         "test-sha256-2",
			URI:            "test-uri-2",
			Version:        "test-version-2",
			VersionPattern: `test-version-[\d]`,
		}

		d.Update(carton.WithExitHandler(exitHandler))

		body, err := ioutil.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(HavePrefix(`# it should preserve
#   these comments
#      exactly

api = "0.6"`))
		Expect(body).To(internal.MatchTOML(`api = "0.6"
[buildpack]
id = "some-buildpack"
name = "Some Buildpack"
version = "1.2.3"

[[metadata.dependencies]]
  id      = "test-id"
  name    = "Test Name"
  version = "test-version-2"
  uri     = "test-uri-2"
  sha256  = "test-sha256-2"
  stacks  = [ "test-stack" ]
`))
	})
}
