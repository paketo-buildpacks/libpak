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
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testExists(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect   = NewWithT(t).Expect
		testPath string
	)

	it.Before(func() {
		testPath = t.TempDir()
	})

	when("checking something exists", func() {
		it("should return true if path is a file", func() {
			path := filepath.Join(testPath, "test-file")
			Expect(os.WriteFile(path, []byte{}, 0644)).To(Succeed())
			exists, err := sherpa.Exists(path)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		it("should return true if path is a directory", func() {
			path := filepath.Join(testPath, "test-dir")
			Expect(os.Mkdir(path, 0755)).To(Succeed())
			exists, err := sherpa.Exists(path)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		it("should return false if path does not exist", func() {
			exists, err := sherpa.Exists(filepath.Join(testPath, "does-not-exist"))
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
	})

	when("checking a directory exists", func() {
		it("should return true if path is a directory", func() {
			path := filepath.Join(testPath, "test-dir")
			Expect(os.Mkdir(path, 0755)).To(Succeed())
			exists, err := sherpa.DirExists(path)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		it("should return false if path is a file", func() {
			path := filepath.Join(testPath, "test-file")
			Expect(os.WriteFile(path, []byte{}, 0644)).To(Succeed())
			exists, err := sherpa.DirExists(path)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		it("should return false if path does not exist", func() {
			exists, err := sherpa.FileExists(filepath.Join(testPath, "does-not-exist"))
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
	})

	when("checking a file exists", func() {
		it("should return true if path is a file", func() {
			path := filepath.Join(testPath, "test-file")
			Expect(os.WriteFile(path, []byte{}, 0644)).To(Succeed())
			exists, err := sherpa.FileExists(path)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		it("should return false if path is a directory", func() {
			path := filepath.Join(testPath, "test-dir")
			Expect(os.Mkdir(path, 0755)).To(Succeed())
			exists, err := sherpa.FileExists(path)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
			BeARegularFile()
		})

		it("should return false if path does not exist", func() {
			exists, err := sherpa.FileExists(filepath.Join(testPath, "does-not-exist"))
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
	})

	when("checking a symlink exists", func() {
		it("should return true if path is a symlink", func() {
			path := filepath.Join(testPath, "tmp-link")
			Expect(os.Symlink(".", path)).ToNot(HaveOccurred())
			exists, err := sherpa.SymlinkExists(path)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		it("should return false if path is a directory", func() {
			path := filepath.Join(testPath, "test-dir")
			Expect(os.Mkdir(path, 0755)).To(Succeed())
			exists, err := sherpa.SymlinkExists(path)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		it("should return false if path does not exist", func() {
			exists, err := sherpa.SymlinkExists(filepath.Join(testPath, "does-not-exist"))
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})
	})
}
