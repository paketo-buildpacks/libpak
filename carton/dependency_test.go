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
	"github.com/paketoio/libpak/carton"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"
)

func testDependency(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		exitHandler *mocks.ExitHandler
		path        string
	)

	it.Before(func() {
		var err error

		exitHandler = &mocks.ExitHandler{}
		exitHandler.On("Error", mock.Anything)

		f, err := ioutil.TempFile("", "carton-dependency")
		Expect(err).NotTo(HaveOccurred())

		_, err = f.WriteString(`id      = "test-id"
name    = "Test Name"
version = "test-version-1"
uri     = "test-uri-1"
sha256  = "test-sha256-1"
stacks  = [ "test-stack" ]
`)
		Expect(err).To(Succeed())
		Expect(f.Close()).To(Succeed())
		path = f.Name()
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	it("updates dependency", func() {
		d := carton.Dependency{
			BuildpackPath:  path,
			ID:             "test-id",
			SHA256:         "test-sha256-2",
			URI:            "test-uri-2",
			Version:        "test-version-2",
			VersionPattern: `test-version-[\d]`,
		}

		d.Build()

		Expect(ioutil.ReadFile(path)).To(Equal([]byte(`id      = "test-id"
name    = "Test Name"
version = "test-version-2"
uri     = "test-uri-2"
sha256  = "test-sha256-2"
stacks  = [ "test-stack" ]
`)))
	})
}
