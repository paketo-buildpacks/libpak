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

	"github.com/buildpacks/libcnb/mocks"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"

	"github.com/paketo-buildpacks/libpak/carton"
	"github.com/paketo-buildpacks/libpak/internal"
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

		f, err := os.CreateTemp("", "carton-package-dependency")
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).To(Succeed())
		path = f.Name()
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	it("updates paketo-buildpacks dependency without losing other fields", func() {
		Expect(os.WriteFile(path, []byte(`# it should preserve
#   these comments
#      exactly

api = "0.6"
[buildpack]
id = "some-id"
name = "some-name"

[[order]]
group = [
	{ id = "paketo-buildpacks/test-1", version="test-version-1" },
	{ id = "paketo-buildpacks/test-2", version="test-version-2" },
]
[metadata]
include-files = [
  "LICENSE",
  "README.md",
  "buildpack.toml",
]`), 0644)).To(Succeed())

		p := carton.PackageDependency{
			BuildpackPath: path,
			ID:            "gcr.io/paketo-buildpacks/test-1",
			Version:       "test-version-3",
		}

		p.Update(carton.WithExitHandler(exitHandler))

		body, err := os.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(HavePrefix(`# it should preserve
#   these comments
#      exactly

api = "0.6"`))
		Expect(body).To(internal.MatchTOML(`api = "0.6"
[buildpack]
id = "some-id"
name = "some-name"

[[order]]
group = [
	{ id = "paketo-buildpacks/test-1", version="test-version-3" },
	{ id = "paketo-buildpacks/test-2", version="test-version-2" },
]
[metadata]
include-files = [
  "LICENSE",
  "README.md",
  "buildpack.toml",
]`))
	})

	it("updates paketo-buildpacks dependency id partial id", func() {
		Expect(os.WriteFile(path, []byte(`
[[order]]
group = [
	{ id = "paketo-buildpacks/test-1", version="test-version-1" },
	{ id = "paketo-buildpacks/test-2", version="test-version-2" },
]`), 0644)).To(Succeed())

		p := carton.PackageDependency{
			BuildpackPath: path,
			ID:            "paketo-buildpacks/test-1",
			Version:       "test-version-3",
		}

		p.Update(carton.WithExitHandler(exitHandler))

		Expect(os.ReadFile(path)).To(internal.MatchTOML(`[[order]]
group = [
	{ id = "paketo-buildpacks/test-1", version="test-version-3" },
	{ id = "paketo-buildpacks/test-2", version="test-version-2" },
]`))
	})

	it("updates paketocommunity dependency", func() {
		Expect(os.WriteFile(path, []byte(`[[order]]
group = [
	{ id = "paketocommunity/test-1", version="test-version-1" },
	{ id = "paketocommunity/test-2", version="test-version-2" },
]`), 0644)).To(Succeed())

		p := carton.PackageDependency{
			BuildpackPath: path,
			ID:            "docker.io/paketocommunity/test-1",
			Version:       "test-version-3",
		}

		p.Update(carton.WithExitHandler(exitHandler))

		Expect(os.ReadFile(path)).To(internal.MatchTOML(`[[order]]
group = [
	{ id = "paketocommunity/test-1", version="test-version-3" },
	{ id = "paketocommunity/test-2", version="test-version-2" },
]`))
	})

	it("updates builder dependency", func() {
		Expect(os.WriteFile(path, []byte(`buildpacks = [
	{ id = "paketo-buildpacks/test-1", uri = "docker://gcr.io/paketo-buildpacks/test-1:test-version-1" },
	{ id = "paketo-buildpacks/test-2", uri = "docker://gcr.io/paketo-buildpacks/test-2:test-version-2" },
]`), 0644)).To(Succeed())

		p := carton.PackageDependency{
			BuilderPath: path,
			ID:          "gcr.io/paketo-buildpacks/test-1",
			Version:     "test-version-3",
		}

		p.Update(carton.WithExitHandler(exitHandler))

		Expect(os.ReadFile(path)).To(internal.MatchTOML(`buildpacks = [
	{ id = "paketo-buildpacks/test-1", uri = "docker://gcr.io/paketo-buildpacks/test-1:test-version-3" },
	{ id = "paketo-buildpacks/test-2", uri = "docker://gcr.io/paketo-buildpacks/test-2:test-version-2" },
]`))
	})

	it("updates paketo-buildpacks package dependency", func() {
		Expect(os.WriteFile(path, []byte(`dependencies = [
	{ uri = "docker://gcr.io/paketo-buildpacks/test-1:test-version-1" },
	{ uri = "docker://gcr.io/paketo-buildpacks/test-2:test-version-2" },
]`), 0644)).To(Succeed())

		p := carton.PackageDependency{
			PackagePath: path,
			ID:          "gcr.io/paketo-buildpacks/test-1",
			Version:     "test-version-3",
		}

		p.Update(carton.WithExitHandler(exitHandler))

		Expect(os.ReadFile(path)).To(internal.MatchTOML(`dependencies = [
	{ uri = "docker://gcr.io/paketo-buildpacks/test-1:test-version-3" },
	{ uri = "docker://gcr.io/paketo-buildpacks/test-2:test-version-2" },
]`))
	})

	it("updates paketocommunity package dependency", func() {
		Expect(os.WriteFile(path, []byte(`dependencies = [
	{ uri = "docker://docker.io/paketocommunity/test-1:test-version-1" },
	{ uri = "docker://docker.io/paketocommunity/test-2:test-version-2" },
]`), 0644)).To(Succeed())

		p := carton.PackageDependency{
			PackagePath: path,
			ID:          "docker.io/paketocommunity/test-1",
			Version:     "test-version-3",
		}

		p.Update(carton.WithExitHandler(exitHandler))

		Expect(os.ReadFile(path)).To(internal.MatchTOML(`dependencies = [
	{ uri = "docker://docker.io/paketocommunity/test-1:test-version-3" },
	{ uri = "docker://docker.io/paketocommunity/test-2:test-version-2" },
]`))
	})

}
