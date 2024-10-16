/*
 * Copyright 2018-2022 the original author or authors.
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
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/libpak/v2/sherpa"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

func testCopyDir(t *testing.T, _ spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		testPath string
		destPath string

		testDirs = []struct {
			path string
			perm os.FileMode
		}{
			{"foo", 0755},
			{"bar", 0755},
			{"baz", 0755},
			{filepath.Join("baz", "qux"), 0700},
		}

		testFiles = []struct {
			path string
			perm os.FileMode
		}{
			{filepath.Join("foo", "foo-test-file"), 0644},
			{filepath.Join("bar", "bar-test-file"), 0644},
			{filepath.Join("baz", "baz-test-file"), 0644},
			{filepath.Join("baz", "qux", "qux-test-file"), 0600},
		}
	)

	it.Before(func() {
		var err error

		testPath, err = os.MkdirTemp("", "sherpa-copy-dir")
		Expect(err).ToNot(HaveOccurred())

		destPath, err = os.MkdirTemp("", "sherpa-dest-dir")
		Expect(err).ToNot(HaveOccurred())

		// Test directory:
		//
		// .
		// ├── bar
		// │	└── bar-test-file
		// ├── baz
		// │	├── baz-test-file
		// │	└── qux
		// │		└── qux-test-file
		// └── foo
		// 		└── foo-test-file

		for _, testDir := range testDirs {
			Expect(os.Mkdir(filepath.Join(testPath, testDir.path), testDir.perm)).To(Succeed())
		}

		for _, testFile := range testFiles {
			path := filepath.Join(testPath, testFile.path)
			Expect(os.WriteFile(path, []byte(testFile.path), testFile.perm)).To(Succeed())
		}
	})

	it.After(func() {
		Expect(os.RemoveAll(testPath)).To(Succeed())
		Expect(os.RemoveAll(destPath)).To(Succeed())
	})

	it("copies a directory", func() {
		dest := filepath.Join(destPath, "test-dir")
		Expect(sherpa.CopyDir(testPath, dest)).To(Succeed())
		for _, testDir := range testDirs {
			dir := filepath.Join(dest, testDir.path)
			Expect(dir).To(BeADirectory())

			// Verify directory permissions
			info, err := os.Stat(dir)
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Mode().Perm()).To(Equal(testDir.perm))
		}
		for _, testFile := range testFiles {
			file := filepath.Join(dest, testFile.path)
			Expect(file).To(BeARegularFile())

			// Verify file permissions
			info, err := os.Stat(file)
			Expect(err).ToNot(HaveOccurred())
			Expect(info.Mode().Perm()).To(Equal(testFile.perm))

			// Verify file content
			content, err := os.ReadFile(file)
			Expect(err).ToNot(HaveOccurred())
			Expect(content).To(Equal([]byte(testFile.path)))
		}
	})
}
