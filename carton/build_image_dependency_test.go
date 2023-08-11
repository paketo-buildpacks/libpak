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
	"os"
	"testing"

	"github.com/buildpacks/libcnb/v2/mocks"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"

	"github.com/paketo-buildpacks/libpak/carton"
)

func testBuildImageDependency(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		exitHandler *mocks.ExitHandler
		path        string
	)

	it.Before(func() {
		var err error

		exitHandler = &mocks.ExitHandler{}
		exitHandler.On("Error", mock.Anything)

		f, err := os.CreateTemp("", "carton-image-dependency")
		Expect(err).NotTo(HaveOccurred())

		_, err = f.WriteString(`test-prologue
build-image = "image-name:test-version-1"
test-epilogue
`)
		Expect(err).To(Succeed())
		Expect(f.Close()).To(Succeed())
		path = f.Name()
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	it("updates dependency", func() {
		d := carton.BuildImageDependency{
			BuilderPath: path,
			Version:     "test-version-2",
		}

		d.Update(carton.WithExitHandler(exitHandler))

		Expect(os.ReadFile(path)).To(Equal([]byte(`test-prologue
build-image = "image-name:test-version-2"
test-epilogue
`)))
	})
}
