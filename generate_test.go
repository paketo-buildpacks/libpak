/*
 * Copyright 2023 the original author or authors.
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

func testGenerate(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		applicationPath string
		extensionPath   string
		buildPlanPath   string
		commandPath     string
		exitHandler     *mocks.ExitHandler
		outputPath      string
		platformPath    string
		tomlWriter      *mocks.TOMLWriter

		workingDir string
	)

	it.Before(func() {
		var err error

		applicationPath = t.TempDir()
		applicationPath, err = filepath.EvalSymlinks(applicationPath)
		Expect(err).NotTo(HaveOccurred())

		f, err := os.CreateTemp("", "generate-buildplan-path")
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).NotTo(HaveOccurred())
		buildPlanPath = f.Name()

		Expect(os.Setenv("CNB_BP_PLAN_PATH", buildPlanPath)).To(Succeed())

		extensionPath = t.TempDir()
		Expect(err).NotTo(HaveOccurred())

		Expect(os.Setenv("CNB_EXTENSION_DIR", extensionPath)).To(Succeed())

		outputPath = t.TempDir()
		Expect(err).NotTo(HaveOccurred())

		Expect(os.Setenv("CNB_OUTPUT_DIR", outputPath)).To(Succeed())

		commandPath = filepath.Join(extensionPath, "bin", "generate")

		exitHandler = &mocks.ExitHandler{}
		exitHandler.On("Error", mock.Anything)

		platformPath = t.TempDir()
		Expect(err).NotTo(HaveOccurred())

		Expect(os.Setenv("CNB_PLATFORM_DIR", platformPath)).To(Succeed())

		tomlWriter = &mocks.TOMLWriter{}
		tomlWriter.On("Write", mock.Anything, mock.Anything).Return(nil)

		Expect(os.Setenv("CNB_STACK_ID", "test-stack-id")).To(Succeed())

		exitHandler = &mocks.ExitHandler{}
		exitHandler.On("Error", mock.Anything)
		exitHandler.On("Fail")
		exitHandler.On("Pass")

		workingDir, err = os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chdir(applicationPath)).To(Succeed())
	})

	it.After(func() {
		Expect(os.Chdir(workingDir)).To(Succeed())
		Expect(os.Unsetenv("CNB_STACK_ID")).To(Succeed())
		Expect(os.Unsetenv("CNB_EXTENSION_DIR")).To(Succeed())
		Expect(os.Unsetenv("CNB_PLATFORM_DIR")).To(Succeed())
		Expect(os.Unsetenv("CNB_BP_PLAN_PATH")).To(Succeed())
		Expect(os.Unsetenv("CNB_OUTPUT_PATH")).To(Succeed())

		Expect(os.RemoveAll(applicationPath)).To(Succeed())
		Expect(os.RemoveAll(extensionPath)).To(Succeed())
		Expect(os.RemoveAll(buildPlanPath)).To(Succeed())
		Expect(os.RemoveAll(platformPath)).To(Succeed())
		Expect(os.RemoveAll(outputPath)).To(Succeed())
	})

	it("handles error from Generate", func() {
		Expect(os.WriteFile(filepath.Join(extensionPath, "extension.toml"), []byte(`
api = "0.8"

[extension]
name    = "test-name"
version = "test-version"`),
			0644)).To(Succeed())

		libpak.Generate(func(ctx libcnb.GenerateContext) (libcnb.GenerateResult, error) {
			return libcnb.GenerateResult{}, fmt.Errorf("test-error")
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
