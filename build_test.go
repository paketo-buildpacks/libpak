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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpacks/libcnb"
	"github.com/buildpacks/libcnb/mocks"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"

	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		applicationPath   string
		builder           *mocks.Builder
		buildpackPath     string
		buildpackPlanPath string
		commandPath       string
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
		Expect(err).NotTo(HaveOccurred())
		applicationPath, err = filepath.EvalSymlinks(applicationPath)
		Expect(err).NotTo(HaveOccurred())

		builder = &mocks.Builder{}

		buildpackPath = t.TempDir()
		Expect(err).NotTo(HaveOccurred())

		f, err := os.CreateTemp("", "build-buildpackplan-path")
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).NotTo(HaveOccurred())
		buildpackPlanPath = f.Name()

		commandPath = filepath.Join(buildpackPath, "bin", "build")

		environmentWriter = &mocks.EnvironmentWriter{}
		environmentWriter.On("Write", mock.Anything, mock.Anything).Return(nil)

		exitHandler = &mocks.ExitHandler{}
		exitHandler.On("Error", mock.Anything)

		layersPath = t.TempDir()
		Expect(err).NotTo(HaveOccurred())

		platformPath = t.TempDir()
		Expect(err).NotTo(HaveOccurred())

		tomlWriter = &mocks.TOMLWriter{}
		tomlWriter.On("Write", mock.Anything, mock.Anything).Return(nil)

		Expect(os.Setenv("CNB_STACK_ID", "test-stack-id")).To(Succeed())

		workingDir, err = os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chdir(applicationPath)).To(Succeed())
	})

	it.After(func() {
		Expect(os.Chdir(workingDir)).To(Succeed())
		Expect(os.Unsetenv("CNB_STACK_ID")).To(Succeed())

		Expect(os.RemoveAll(applicationPath)).To(Succeed())
		Expect(os.RemoveAll(buildpackPath)).To(Succeed())
		Expect(os.RemoveAll(buildpackPlanPath)).To(Succeed())
		Expect(os.RemoveAll(layersPath)).To(Succeed())
		Expect(os.RemoveAll(platformPath)).To(Succeed())
	})

	it("handles error from Builder", func() {
		Expect(os.WriteFile(filepath.Join(buildpackPath, "buildpack.toml"), []byte(`
api = "0.6"

[buildpack]
name    = "test-name"
version = "test-version"`),
			0644)).To(Succeed())
		builder.On("Build", mock.Anything).Return(libcnb.NewBuildResult(), fmt.Errorf("test-error"))

		libpak.Build(builder,
			libcnb.WithArguments([]string{commandPath, layersPath, platformPath, buildpackPlanPath}),
			libcnb.WithExitHandler(exitHandler),
		)

		Expect(exitHandler.Calls[0].Arguments.Get(0)).To(MatchError(bard.IdentifiableError{
			Name:        "test-name",
			Description: "test-version",
			Err:         fmt.Errorf("test-error"),
		}))
	})
}
