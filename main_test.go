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

package libpak_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	"github.com/buildpacks/libcnb/mocks"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"

	"github.com/paketo-buildpacks/libpak"
)

func testMain(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		applicationPath   string
		builder           *mocks.Builder
		buildpackPath     string
		buildpackPlanPath string
		buildPlanPath     string
		detector          *mocks.Detector
		environmentWriter *mocks.EnvironmentWriter
		exitHandler       *mocks.ExitHandler
		layersPath        string
		platformPath      string
		tomlWriter        *mocks.TOMLWriter

		workingDir string
	)

	it.Before(func() {
		var err error

		applicationPath = t.TempDir()
		applicationPath, err = filepath.EvalSymlinks(applicationPath)
		Expect(err).NotTo(HaveOccurred())

		builder = &mocks.Builder{}

		buildpackPath = t.TempDir()
		Expect(os.Setenv("CNB_BUILDPACK_DIR", buildpackPath)).To(Succeed())

		Expect(os.WriteFile(filepath.Join(buildpackPath, "buildpack.toml"),
			[]byte(`
api = "0.6"

[buildpack]
id = "test-id"
name = "test-name"
version = "1.1.1"
clear-env = true

[[order]]
[[order.group]]
id = "test-id"
version = "2.2.2"
optional = true

[[stacks]]
id = "test-id"
mixins = ["test-name"]

[metadata]
test-key = "test-value"
`),
			0644),
		).To(Succeed())

		f, err := os.CreateTemp("", "main-buildpackplan-path")
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).NotTo(HaveOccurred())
		buildpackPlanPath = f.Name()

		Expect(os.WriteFile(buildpackPlanPath,
			[]byte(`
[[entries]]
name = "test-name"
version = "test-version"

[entries.metadata]
test-key = "test-value"
`),
			0644),
		).To(Succeed())

		f, err = os.CreateTemp("", "main-buildplan-path")
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).NotTo(HaveOccurred())
		buildPlanPath = f.Name()

		detector = &mocks.Detector{}

		environmentWriter = &mocks.EnvironmentWriter{}
		environmentWriter.On("Write", mock.Anything, mock.Anything).Return(nil)

		exitHandler = &mocks.ExitHandler{}
		exitHandler.On("Error", mock.Anything)
		exitHandler.On("Pass", mock.Anything)
		exitHandler.On("Fail", mock.Anything)

		layersPath = t.TempDir()

		Expect(os.WriteFile(filepath.Join(layersPath, "store.toml"),
			[]byte(`
[metadata]
test-key = "test-value"
`),
			0644),
		).To(Succeed())

		platformPath = t.TempDir()

		Expect(os.MkdirAll(filepath.Join(platformPath, "bindings", "alpha", "metadata"), 0755)).To(Succeed())
		Expect(os.WriteFile(
			filepath.Join(platformPath, "bindings", "alpha", "metadata", "test-metadata-key"),
			[]byte("test-metadata-value"),
			0644,
		)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(platformPath, "bindings", "alpha", "secret"), 0755)).To(Succeed())
		Expect(os.WriteFile(
			filepath.Join(platformPath, "bindings", "alpha", "secret", "test-secret-key"),
			[]byte("test-secret-value"),
			0644,
		)).To(Succeed())

		Expect(os.MkdirAll(filepath.Join(platformPath, "env"), 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(platformPath, "env", "TEST_ENV"), []byte("test-value"), 0644)).
			To(Succeed())

		tomlWriter = &mocks.TOMLWriter{}
		tomlWriter.On("Write", mock.Anything, mock.Anything).Return(nil)

		Expect(os.Setenv("CNB_STACK_ID", "test-stack-id")).To(Succeed())

		workingDir, err = os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chdir(applicationPath)).To(Succeed())
	})

	it.After(func() {
		Expect(os.Chdir(workingDir)).To(Succeed())
		Expect(os.Unsetenv("CNB_BUILDPACK_DIR")).To(Succeed())
		Expect(os.Unsetenv("CNB_STACK_ID")).To(Succeed())

		Expect(os.RemoveAll(applicationPath)).To(Succeed())
		Expect(os.RemoveAll(buildpackPath)).To(Succeed())
		Expect(os.RemoveAll(buildpackPlanPath)).To(Succeed())
		Expect(os.RemoveAll(layersPath)).To(Succeed())
		Expect(os.RemoveAll(platformPath)).To(Succeed())
	})

	it("encounters the wrong number of arguments", func() {
		libpak.Main(detector, builder,
			libcnb.WithArguments([]string{}),
			libcnb.WithExitHandler(exitHandler),
		)

		Expect(exitHandler.Calls[0].Arguments.Get(0)).To(MatchError("expected command name"))
	})

	it("calls builder for build command", func() {
		builder.On("Build", mock.Anything).Return(libcnb.NewBuildResult(), nil)
		commandPath := filepath.Join("bin", "build")

		libpak.Main(detector, builder,
			libcnb.WithArguments([]string{commandPath, layersPath, platformPath, buildpackPlanPath}),
			libcnb.WithExitHandler(exitHandler),
		)

		Expect(exitHandler.Calls).To(BeEmpty())
	})

	it("calls detector for detect command", func() {
		detector.On("Detect", mock.Anything).Return(libcnb.DetectResult{Pass: true}, nil)
		commandPath := filepath.Join("bin", "detect")

		libpak.Main(detector, builder,
			libcnb.WithArguments([]string{commandPath, platformPath, buildPlanPath}),
			libcnb.WithExitHandler(exitHandler),
		)
	})

	it("calls exitHandler.Pass() on detection pass", func() {
		detector.On("Detect", mock.Anything).Return(libcnb.DetectResult{Pass: true}, nil)
		commandPath := filepath.Join("bin", "detect")

		libpak.Main(detector, builder,
			libcnb.WithArguments([]string{commandPath, platformPath, buildPlanPath}),
			libcnb.WithExitHandler(exitHandler),
		)

		Expect(exitHandler.Calls[0].Method).To(BeIdenticalTo("Pass"))
	})

	it("calls exitHandler.Fail() on detection fail", func() {
		detector.On("Detect", mock.Anything).Return(libcnb.DetectResult{Pass: false}, nil)
		commandPath := filepath.Join("bin", "detect")

		libpak.Main(detector, builder,
			libcnb.WithArguments([]string{commandPath, platformPath, buildPlanPath}),
			libcnb.WithExitHandler(exitHandler),
		)

		Expect(exitHandler.Calls[0].Method).To(BeIdenticalTo("Fail"))
	})

	it("encounters an unknown command", func() {
		commandPath := filepath.Join("bin", "test-command")

		libpak.Main(detector, builder,
			libcnb.WithArguments([]string{commandPath}),
			libcnb.WithExitHandler(exitHandler),
		)

		Expect(exitHandler.Calls[0].Arguments.Get(0)).To(MatchError("unsupported command test-command"))
	})

}
