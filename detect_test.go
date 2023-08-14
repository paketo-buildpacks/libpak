/*
 * Copyright 2018-2023 the original author or authors.
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

	"github.com/buildpacks/libcnb/v2"
	"github.com/buildpacks/libcnb/v2/mocks"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"

	"github.com/paketo-buildpacks/libpak/v2"
	"github.com/paketo-buildpacks/libpak/v2/bard"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		applicationPath string
		buildpackPath   string
		buildPlanPath   string
		commandPath     string
		exitHandler     *mocks.ExitHandler
		layersPath      string
		platformPath    string
		tomlWriter      *mocks.TOMLWriter

		workingDir string
	)

	it.Before(func() {
		var err error

		applicationPath = t.TempDir()
		applicationPath, err = filepath.EvalSymlinks(applicationPath)
		Expect(err).NotTo(HaveOccurred())

		buildpackPath = t.TempDir()

		f, err := os.CreateTemp("", "detect-buildplan-path")
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).NotTo(HaveOccurred())
		buildPlanPath = f.Name()

		Expect(os.Setenv("CNB_BUILD_PLAN_PATH", buildPlanPath)).To(Succeed())

		commandPath = filepath.Join(buildpackPath, "bin", "detect")

		exitHandler = &mocks.ExitHandler{}
		exitHandler.On("Error", mock.Anything)
		exitHandler.On("Fail")
		exitHandler.On("Pass")

		Expect(os.Setenv("CNB_STACK_ID", "test-stack-id")).To(Succeed())

		layersPath = t.TempDir()
		Expect(err).NotTo(HaveOccurred())

		Expect(os.Setenv("CNB_LAYERS_DIR", layersPath)).To(Succeed())

		platformPath = t.TempDir()
		Expect(err).NotTo(HaveOccurred())

		Expect(os.Setenv("CNB_PLATFORM_DIR", platformPath)).To(Succeed())

		tomlWriter = &mocks.TOMLWriter{}
		tomlWriter.On("Write", mock.Anything, mock.Anything).Return(nil)

		Expect(os.Setenv("CNB_STACK_ID", "test-stack-id")).To(Succeed())

		buildpackPath = t.TempDir()
		Expect(err).NotTo(HaveOccurred())

		Expect(os.Setenv("CNB_BUILDPACK_DIR", buildpackPath)).To(Succeed())

		workingDir, err = os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chdir(applicationPath)).To(Succeed())
	})

	it.After(func() {
		Expect(os.Chdir(workingDir)).To(Succeed())
		Expect(os.Unsetenv("CNB_BINDINGS")).To(Succeed())
		Expect(os.Unsetenv("CNB_STACK_ID")).To(Succeed())
		Expect(os.Unsetenv("CNB_BUILDPACK_DIR")).To(Succeed())
		Expect(os.Unsetenv("CNB_LAYERS_DIR")).To(Succeed())
		Expect(os.Unsetenv("CNB_PLATFORM_DIR")).To(Succeed())
		Expect(os.Unsetenv("CNB_BUILD_PLAN_PATH")).To(Succeed())

		Expect(os.RemoveAll(applicationPath)).To(Succeed())
		Expect(os.RemoveAll(buildpackPath)).To(Succeed())
		Expect(os.RemoveAll(buildPlanPath)).To(Succeed())
		Expect(os.RemoveAll(layersPath)).To(Succeed())
		Expect(os.RemoveAll(platformPath)).To(Succeed())
		Expect(os.RemoveAll(buildpackPath)).To(Succeed())
	})

	it("handles error from Detector", func() {
		Expect(os.WriteFile(filepath.Join(buildpackPath, "buildpack.toml"), []byte(`
api = "0.8"

[buildpack]
name    = "test-name"
version = "test-version"`),
			0644)).To(Succeed())

		libpak.Detect(func(ctx libcnb.DetectContext) (libcnb.DetectResult, error) {
			return libcnb.DetectResult{}, fmt.Errorf("test-error")
		},
			libcnb.WithArguments([]string{commandPath, platformPath, buildPlanPath}),
			libcnb.WithExitHandler(exitHandler),
		)

		Expect(exitHandler.Calls[0].Arguments.Get(0)).To(MatchError(bard.IdentifiableError{
			Name:        "test-name",
			Description: "test-version",
			Err:         fmt.Errorf("test-error"),
		}))
	})
}
