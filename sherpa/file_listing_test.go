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

package sherpa_test

import (
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak/sherpa"
)

func testFileListing(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		path string
	)

	it.Before(func() {
		var err error

		path, err = ioutil.TempDir("", "file-listing")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	it("create listing", func() {
		Expect(ioutil.WriteFile(filepath.Join(path, "alpha.txt"), []byte{1}, 0644)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(path, "test-directory"), 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(path, "test-directory", "bravo.txt"), []byte{2}, 0644)).To(Succeed())

		e, err := sherpa.NewFileListing(path)
		Expect(err).NotTo(HaveOccurred())

		Expect(e).To(HaveLen(3))
	})

	it("create listing as hash with non-regular file", func() {
		Expect(ioutil.WriteFile(filepath.Join(path, "alpha.txt"), []byte{1}, 0644)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(path, "test-directory"), 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(path, "test-directory", "bravo.txt"), []byte{2}, 0644)).To(Succeed())
		Expect(os.Symlink(filepath.Join(path, "test-directory"), filepath.Join(path, "symlink-test-dir")))
		Expect(os.Symlink(filepath.Join(path, "test-directory", "bravo.txt"), filepath.Join(path, "symlink-bravo.txt")))

		e, err := sherpa.NewFileListing(path)
		Expect(err).NotTo(HaveOccurred())

		Expect(e).To(HaveLen(5))
		Expect(e[0].Path).To(HaveSuffix("alpha.txt"))
		Expect(e[1].Path).To(HaveSuffix("symlink-bravo.txt"))
		Expect(e[2].Path).To(HaveSuffix("symlink-test-dir"))
		Expect(e[3].Path).To(HaveSuffix("test-directory"))
		Expect(e[4].Path).To(HaveSuffix("bravo.txt"))
		Expect(e[1].SHA256).To(Equal(e[4].SHA256)) // symlink to file should have hash of target file
	})

	it("create listing and get SHA256", func() {
		Expect(ioutil.WriteFile(filepath.Join(path, "alpha.txt"), []byte{}, 0644)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(path, "test-directory"), 0755)).To(Succeed())
		Expect(ioutil.WriteFile(filepath.Join(path, "test-directory", "bravo.txt"), []byte{}, 0644)).To(Succeed())

		e, err := sherpa.NewFileListing(path)
		Expect(err).NotTo(HaveOccurred())

		hash := sha256.New()
		for _, file := range e {
			hash.Write([]byte(file.Path + file.Mode + file.SHA256 + "\n"))
		}

		s, err := sherpa.NewFileListingHash(path)
		Expect(err).NotTo(HaveOccurred())

		Expect(s).To(Equal(hex.EncodeToString(hash.Sum(nil))))
	})
}
