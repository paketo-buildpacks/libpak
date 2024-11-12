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
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak/v2/crush"
)

func testCrush(t *testing.T, context spec.G, it spec.S) {
	var (
		path string
	)

	it.Before(func() {
		path = t.TempDir()
	})

	context("Create", func() {
		var (
			Expect = NewWithT(t).Expect

			out      *os.File
			testPath string
		)

		it.Before(func() {
			var err error

			out, err = os.CreateTemp("", "crush-tar")
			Expect(err).NotTo(HaveOccurred())

			testPath = t.TempDir()
		})

		it.After(func() {
			Expect(out.Close())
			Expect(os.RemoveAll(out.Name())).To(Succeed())
			Expect(os.RemoveAll(testPath)).To(Succeed())
		})

		it("writes a TAR", func() {
			Expect(os.WriteFile(filepath.Join(path, "fileA.txt"), []byte(""), 0600)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(path, "dirA"), 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(path, "dirA", "fileB.txt"), []byte(""), 0600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(path, "dirA", "fileC.txt"), []byte(""), 0600)).To(Succeed())
			Expect(os.Symlink(filepath.Join(path, "dirA", "fileC.txt"), filepath.Join(path, "dirA", "fileD.txt"))).To(Succeed())

			Expect(crush.CreateTar(out, path)).To(Succeed())

			in, err := os.Open(out.Name())
			Expect(err).NotTo(HaveOccurred())

			Expect(crush.Extract(in, testPath, 0)).To(Succeed())
			Expect(filepath.Join(testPath, "fileA.txt")).To(BeARegularFile())
			Expect(filepath.Join(testPath, "dirA", "fileB.txt")).To(BeARegularFile())
			Expect(filepath.Join(testPath, "dirA", "fileC.txt")).To(BeARegularFile())
			Expect(os.Readlink(filepath.Join(testPath, "dirA", "fileD.txt"))).To(Equal(filepath.Join(path, "dirA", "fileC.txt")))
		})

		it("writes a TAR.GZ", func() {
			Expect(os.WriteFile(filepath.Join(path, "fileA.txt"), []byte(""), 0600)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(path, "dirA"), 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(path, "dirA", "fileB.txt"), []byte(""), 0600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(path, "dirA", "fileC.txt"), []byte(""), 0600)).To(Succeed())
			Expect(os.Symlink(filepath.Join(path, "dirA", "fileC.txt"), filepath.Join(path, "dirA", "fileD.txt"))).To(Succeed())

			Expect(crush.CreateTarGz(out, path)).To(Succeed())

			in, err := os.Open(out.Name())
			Expect(err).NotTo(HaveOccurred())

			Expect(crush.Extract(in, testPath, 0)).To(Succeed())
			Expect(filepath.Join(testPath, "fileA.txt")).To(BeARegularFile())
			Expect(filepath.Join(testPath, "dirA", "fileB.txt")).To(BeARegularFile())
			Expect(filepath.Join(testPath, "dirA", "fileC.txt")).To(BeARegularFile())
			Expect(os.Readlink(filepath.Join(testPath, "dirA", "fileD.txt"))).To(Equal(filepath.Join(path, "dirA", "fileC.txt")))
		})

		it("writes a JAR", func() {
			cwd, _ := os.Getwd()
			Expect(os.MkdirAll(filepath.Join(path, "META-INF"), 0700)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(path, "META-INF", "MANIFEST.MF"), []byte(`
	Spring-Boot-Version: 3.3.1
	Spring-Boot-Classes: BOOT-INF/classes
	Spring-Boot-Lib: BOOT-INF/lib
	`), 0600)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(path, "BOOT-INF"), 0700)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(path, "BOOT-INF", "classes"), 0700)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(path, "BOOT-INF", "classes", "OtherClass.class"), []byte(""), 0600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(path, "BOOT-INF", "classes", "YetOther.class"), []byte(""), 0600)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(path, "BOOT-INF", "lib"), 0700)).To(Succeed())
			os.Symlink(filepath.Join(cwd, "testdata", "test-archive.jar"), filepath.Join(path, "BOOT-INF", "lib", "test-archive.jar"))

			Expect(crush.CreateJar(path+"/", out.Name()+".jar")).To(Succeed())

			in, err := os.Open(out.Name() + ".jar")
			Expect(err).NotTo(HaveOccurred())

			Expect(crush.Extract(in, testPath, 0)).To(Succeed())
			Expect(filepath.Join(testPath, "BOOT-INF", "classes", "OtherClass.class")).To(BeARegularFile())
			Expect(filepath.Join(testPath, "META-INF", "MANIFEST.MF")).To(BeARegularFile())
			Expect(filepath.Join(testPath, "BOOT-INF", "lib", "test-archive.jar")).To(BeARegularFile())
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

		context("Tar", func() {
			it.Before(func() {
				var err error
				in, err = os.Open(filepath.Join("testdata", "test-archive.tar"))
				Expect(err).NotTo(HaveOccurred())
			})

			it("extracts the archive", func() {
				Expect(crush.Extract(in, path, 0)).To(Succeed())
				Expect(filepath.Join(path, "fileA.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileC.txt")).To(BeARegularFile())
			})

			it("skips stripped components", func() {
				Expect(crush.Extract(in, path, 1)).To(Succeed())
				Expect(filepath.Join(path, "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "fileC.txt")).To(BeARegularFile())
			})
		})

		context("TarGZ", func() {
			it.Before(func() {
				var err error
				in, err = os.Open(filepath.Join("testdata", "test-archive.tar.gz"))
				Expect(err).NotTo(HaveOccurred())
			})

			it("extracts the archive", func() {
				Expect(crush.Extract(in, path, 0)).To(Succeed())
				Expect(filepath.Join(path, "fileA.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileC.txt")).To(BeARegularFile())
			})

			it("skips stripped components", func() {
				Expect(crush.Extract(in, path, 1)).To(Succeed())
				Expect(filepath.Join(path, "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "fileC.txt")).To(BeARegularFile())
			})
		})

		context("TarBz2", func() {
			it.Before(func() {
				var err error
				in, err = os.Open(filepath.Join("testdata", "test-archive.tar.bz2"))
				Expect(err).NotTo(HaveOccurred())
			})

			it("extracts the archive", func() {
				Expect(crush.Extract(in, path, 0)).To(Succeed())
				Expect(filepath.Join(path, "fileA.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileC.txt")).To(BeARegularFile())
			})

			it("skips stripped components", func() {
				Expect(crush.Extract(in, path, 1)).To(Succeed())
				Expect(filepath.Join(path, "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "fileC.txt")).To(BeARegularFile())
			})
		})

		context("TarXZ", func() {
			it.Before(func() {
				var err error
				in, err = os.Open(filepath.Join("testdata", "test-archive.tar.xz"))
				Expect(err).NotTo(HaveOccurred())
			})

			it("extracts the archive", func() {
				Expect(crush.Extract(in, path, 0)).To(Succeed())
				Expect(filepath.Join(path, "fileA.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileC.txt")).To(BeARegularFile())
			})

			it("skips stripped components", func() {
				Expect(crush.Extract(in, path, 1)).To(Succeed())
				Expect(filepath.Join(path, "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "fileC.txt")).To(BeARegularFile())
			})
		})

		context("Zip", func() {
			it.Before(func() {
				var err error
				in, err = os.Open(filepath.Join("testdata", "test-archive.zip"))
				Expect(err).NotTo(HaveOccurred())
			})

			it("extracts the archive", func() {
				Expect(crush.Extract(in, path, 0)).To(Succeed())
				Expect(filepath.Join(path, "fileA.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileC.txt")).To(BeARegularFile())
			})

			it("skips stripped components", func() {
				Expect(crush.Extract(in, path, 1)).To(Succeed())
				Expect(filepath.Join(path, "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "fileC.txt")).To(BeARegularFile())
			})
		})

		context("Tar", func() {
			it.Before(func() {
				var err error
				in, err = os.Open(filepath.Join("testdata", "test-archive.tar"))
				Expect(err).NotTo(HaveOccurred())
			})

			it("extracts the archive", func() {
				Expect(crush.Extract(in, path, 0)).To(Succeed())
				Expect(filepath.Join(path, "fileA.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "dirA", "fileC.txt")).To(BeARegularFile())
			})

			it("skips stripped components", func() {
				Expect(crush.Extract(in, path, 1)).To(Succeed())
				Expect(filepath.Join(path, "fileB.txt")).To(BeARegularFile())
				Expect(filepath.Join(path, "fileC.txt")).To(BeARegularFile())
			})
		})

		context("compression only", func() {
			it("decompresses gzip", func() {
				var err error
				in, err = os.Open(filepath.Join("testdata", "test-compress.gz"))
				Expect(err).NotTo(HaveOccurred())

				Expect(crush.Extract(in, filepath.Join(path, "test-compress"), 0)).To(Succeed())
				Expect(filepath.Join(path, "test-compress")).To(BeARegularFile())
			})

			it("decompresses xz", func() {
				var err error
				in, err = os.Open(filepath.Join("testdata", "test-compress.xz"))
				Expect(err).NotTo(HaveOccurred())

				Expect(crush.Extract(in, filepath.Join(path, "test-compress"), 0)).To(Succeed())
				Expect(filepath.Join(path, "test-compress")).To(BeARegularFile())
			})

			it("decompresses bz2", func() {
				var err error
				in, err = os.Open(filepath.Join("testdata", "test-compress.bz2"))
				Expect(err).NotTo(HaveOccurred())

				Expect(crush.Extract(in, filepath.Join(path, "test-compress"), 0)).To(Succeed())
				Expect(filepath.Join(path, "test-compress")).To(BeARegularFile())
			})
		})
	})
}
