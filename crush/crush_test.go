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

package crush_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak/crush"
)

func testCrush(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		path string
	)

	it.Before(func() {
		var err error

		path, err = ioutil.TempDir("", "crush")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	context("Create", func() {
		var (
			Expect = NewWithT(t).Expect

			out      *os.File
			testPath string
		)
		it.Before(func() {
			var err error

			out, err = ioutil.TempFile("", "crush-tar")
			Expect(err).NotTo(HaveOccurred())

			testPath, err = ioutil.TempDir("", "crush-tar")
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(out.Close())
			Expect(os.RemoveAll(out.Name())).To(Succeed())
			Expect(os.RemoveAll(testPath)).To(Succeed())
		})

		it("writes a TAR", func() {
			Expect(ioutil.WriteFile(filepath.Join(path, "fileA.txt"), []byte(""), 0644)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(path, "dirA"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(path, "dirA", "fileB.txt"), []byte(""), 0644)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(path, "dirA", "fileC.txt"), []byte(""), 0644)).To(Succeed())
			Expect(os.Symlink(filepath.Join(path, "dirA", "fileC.txt"), filepath.Join(path, "dirA", "fileD.txt"))).To(Succeed())

			Expect(crush.CreateTar(out, path)).To(Succeed())

			in, err := os.Open(out.Name())
			Expect(err).NotTo(HaveOccurred())

			Expect(crush.ExtractTar(in, testPath, 0)).To(Succeed())
			Expect(filepath.Join(testPath, "fileA.txt")).To(BeARegularFile())
			Expect(filepath.Join(testPath, "dirA", "fileB.txt")).To(BeARegularFile())
			Expect(filepath.Join(testPath, "dirA", "fileC.txt")).To(BeARegularFile())
			Expect(os.Readlink(filepath.Join(testPath, "dirA", "fileD.txt"))).To(Equal(filepath.Join(path, "dirA", "fileC.txt")))
		})

		it("writes a TAR.GZ", func() {
			Expect(ioutil.WriteFile(filepath.Join(path, "fileA.txt"), []byte(""), 0644)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(path, "dirA"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(path, "dirA", "fileB.txt"), []byte(""), 0644)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(path, "dirA", "fileC.txt"), []byte(""), 0644)).To(Succeed())
			Expect(os.Symlink(filepath.Join(path, "dirA", "fileC.txt"), filepath.Join(path, "dirA", "fileD.txt"))).To(Succeed())

			Expect(crush.CreateTarGz(out, path)).To(Succeed())

			in, err := os.Open(out.Name())
			Expect(err).NotTo(HaveOccurred())

			Expect(crush.ExtractTarGz(in, testPath, 0)).To(Succeed())
			Expect(filepath.Join(testPath, "fileA.txt")).To(BeARegularFile())
			Expect(filepath.Join(testPath, "dirA", "fileB.txt")).To(BeARegularFile())
			Expect(filepath.Join(testPath, "dirA", "fileC.txt")).To(BeARegularFile())
			Expect(os.Readlink(filepath.Join(testPath, "dirA", "fileD.txt"))).To(Equal(filepath.Join(path, "dirA", "fileC.txt")))
		})
	})

	context("Extract", func() {
		var (
			Expect = NewWithT(t).Expect

			in *os.File
		)

		it.After(func() {
			Expect(in.Close()).NotTo(HaveOccurred())
		})

		context("ExtractTar", func() {
			it.Before(func() {
				var err error
				in, err = os.Open(filepath.Join("testdata", "test-archive.tar"))
				Expect(err).NotTo(HaveOccurred())
			})

			it("extracts the archive", func() {
				Expect(crush.ExtractTar(in, path, 0)).To(Succeed())
				Expect(filepath.Join(path, "fileA.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileC.txt")).To(BeARegularFile())
			})

			it("skips stripped components", func() {
				Expect(crush.ExtractTar(in, path, 1)).To(Succeed())
				Expect(filepath.Join(path, "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "fileC.txt")).To(BeARegularFile())
			})
		})

		context("ExtractTarBz2", func() {
			it.Before(func() {
				var err error
				in, err = os.Open(filepath.Join("testdata", "test-archive.tar.bz2"))
				Expect(err).NotTo(HaveOccurred())
			})

			it("extracts the archive", func() {
				Expect(crush.ExtractTarBz2(in, path, 0)).To(Succeed())
				Expect(filepath.Join(path, "fileA.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileC.txt")).To(BeARegularFile())
			})

			it("skips stripped components", func() {
				Expect(crush.ExtractTarBz2(in, path, 1)).To(Succeed())
				Expect(filepath.Join(path, "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "fileC.txt")).To(BeARegularFile())
			})
		})

		context("ExtractTarGz", func() {
			it.Before(func() {
				var err error
				in, err = os.Open(filepath.Join("testdata", "test-archive.tar.gz"))
				Expect(err).NotTo(HaveOccurred())
			})

			it("extracts the archive", func() {
				Expect(crush.ExtractTarGz(in, path, 0)).To(Succeed())
				Expect(filepath.Join(path, "fileA.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileC.txt")).To(BeARegularFile())
			})

			it("skips stripped components", func() {
				Expect(crush.ExtractTarGz(in, path, 1)).To(Succeed())
				Expect(filepath.Join(path, "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "fileC.txt")).To(BeARegularFile())
			})
		})

		context("ExtractTarXZ", func() {
			it.Before(func() {
				var err error
				in, err = os.Open(filepath.Join("testdata", "test-archive.tar.xz"))
				Expect(err).NotTo(HaveOccurred())
			})

			it("extracts the archive", func() {
				Expect(crush.ExtractTarXz(in, path, 0)).To(Succeed())
				Expect(filepath.Join(path, "fileA.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileC.txt")).To(BeARegularFile())
			})

			it("skips stripped components", func() {
				Expect(crush.ExtractTarXz(in, path, 1)).To(Succeed())
				Expect(filepath.Join(path, "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "fileC.txt")).To(BeARegularFile())
			})
		})

		context("ExtractZip", func() {
			it.Before(func() {
				var err error
				in, err = os.Open(filepath.Join("testdata", "test-archive.zip"))
				Expect(err).NotTo(HaveOccurred())
			})

			it("extracts the archive", func() {
				Expect(crush.ExtractZip(in, path, 0)).To(Succeed())
				Expect(filepath.Join(path, "fileA.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileC.txt")).To(BeARegularFile())
			})

			it("skips stripped components", func() {
				Expect(crush.ExtractZip(in, path, 1)).To(Succeed())
				Expect(filepath.Join(path, "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "fileC.txt")).To(BeARegularFile())
			})
		})
	})
}
