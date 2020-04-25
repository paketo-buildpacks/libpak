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
	"github.com/paketo-buildpacks/libpak/carton"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"
)

func testPackageDependency(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		exitHandler *mocks.ExitHandler
		path        string
	)

	it.Before(func() {
		var err error

		exitHandler = &mocks.ExitHandler{}
		exitHandler.On("Error", mock.Anything)

		f, err := ioutil.TempFile("", "carton-package-dependency")
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).To(Succeed())
		path = f.Name()
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	it("updates buildpack dependency", func() {
		Expect(ioutil.WriteFile(path, []byte(`
{ id = "paketo-buildpacks/test-1", version="test-version-1" },
{ id = "paketo-buildpacks/test-2", version="test-version-2" },
`), 0644)).To(Succeed())

		p := carton.PackageDependency{
			BuildpackPath: path,
			ID:            "gcr.io/paketo-buildpacks/test-1",
			Version:       "test-version-3",
		}

		p.Update(carton.WithExitHandler(exitHandler))

		Expect(ioutil.ReadFile(path)).To(Equal([]byte(`
{ id = "paketo-buildpacks/test-1", version="test-version-3" },
{ id = "paketo-buildpacks/test-2", version="test-version-2" },
`)))
	})

	it("updates builder dependency", func() {
		Expect(ioutil.WriteFile(path, []byte(`
{ id = "paketo-buildpacks/test-1", image = "gcr.io/paketo-buildpacks/test-1:test-version-1" },
{ id = "paketo-buildpacks/test-2", image = "gcr.io/paketo-buildpacks/test-2:test-version-2" },
`), 0644)).To(Succeed())

		p := carton.PackageDependency{
			BuilderPath: path,
			ID:            "gcr.io/paketo-buildpacks/test-1",
			Version:       "test-version-3",
		}

		p.Update(carton.WithExitHandler(exitHandler))

		Expect(ioutil.ReadFile(path)).To(Equal([]byte(`
{ id = "paketo-buildpacks/test-1", image = "gcr.io/paketo-buildpacks/test-1:test-version-3" },
{ id = "paketo-buildpacks/test-2", image = "gcr.io/paketo-buildpacks/test-2:test-version-2" },
`)))
	})

	it("updates package dependency", func() {
		Expect(ioutil.WriteFile(path, []byte(`
{ image = "gcr.io/paketo-buildpacks/test-1:test-version-1" },
{ image = "gcr.io/paketo-buildpacks/test-2:test-version-2" },
`), 0644)).To(Succeed())

		p := carton.PackageDependency{
			PackagePath: path,
			ID:            "gcr.io/paketo-buildpacks/test-1",
			Version:       "test-version-3",
		}

		p.Update(carton.WithExitHandler(exitHandler))

		Expect(ioutil.ReadFile(path)).To(Equal([]byte(`
{ image = "gcr.io/paketo-buildpacks/test-1:test-version-3" },
{ image = "gcr.io/paketo-buildpacks/test-2:test-version-2" },
`)))
	})

}
