/*
 * Copyright 2018-2024 the original author or authors.
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

package sherpa_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak/v2/sherpa"
)

func testCopyFile(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		rootPath   string
		sourcePath string
	)

	expectFilePermissionsToMatch := func(destination string, fileMode os.FileMode) {
		fileInfo, err := os.Stat(destination)
		fmt.Println("eperms:", fileInfo.Mode().Perm().String())
		Expect(err).NotTo(HaveOccurred())
		Expect(fileInfo.Mode().Perm().String()).To(Equal(fileMode.String()))
	}

	it.Before(func() {
		var err error

		rootPath = t.TempDir()

		source, err := os.CreateTemp(rootPath, "source.txt")
		Expect(err).NotTo(HaveOccurred())
		sourcePath = source.Name()

		_, err = source.WriteString("test")
		Expect(err).NotTo(HaveOccurred())
		Expect(source.Close()).To(Succeed())
	})

	it.After(func() {
		Expect(os.RemoveAll(rootPath)).To(Succeed())
	})

	it("copy file with permissions", func() {
		destination := filepath.Join(rootPath, "destination/nested/destination.txt")

		source, err := os.Open(sourcePath)
		Expect(err).NotTo(HaveOccurred())
		defer source.Close()

		expectedMode := os.FileMode(0640)
		Expect(os.Chmod(sourcePath, expectedMode)).To(Succeed())

		Expect(sherpa.CopyFile(source, destination)).To(Succeed())
		Expect(os.ReadFile(destination)).To(Equal([]byte("test")))

		expectFilePermissionsToMatch(destination, expectedMode)
	})

	it.Focus("copy file from string path with permissions", func() {
		destination := filepath.Join(rootPath, "destination/other-nested/destination.txt")

		expectedMode := os.FileMode(0440)
		Expect(os.Chmod(sourcePath, expectedMode)).To(Succeed())

		Expect(sherpa.CopyFileFrom(sourcePath, destination)).To(Succeed())
		Expect(os.ReadFile(destination)).To(Equal([]byte("test")))

		expectFilePermissionsToMatch(destination, expectedMode)
	})
}
