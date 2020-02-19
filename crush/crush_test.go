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
	"github.com/paketoio/libpak/crush"
	"github.com/sclevine/spec"
)

func testCrush(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		crush crush.Crush
		in    *os.File
		path  string
	)

	it.Before(func() {
		var err error

		path, err = ioutil.TempDir("", "crush")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(in.Close()).NotTo(HaveOccurred())
		Expect(os.RemoveAll(path)).To(Succeed())
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
			Expect(crush.ExtractTarXZ(in, path, 0)).To(Succeed())
			Expect(filepath.Join(path, "fileA.txt")).To(BeARegularFile())
			Expect(filepath.Join(path, "dirA", "fileB.txt")).To(BeARegularFile())
			Expect(filepath.Join(path, "dirA", "fileC.txt")).To(BeARegularFile())
		})

		it("skips stripped components", func() {
			Expect(crush.ExtractTarXZ(in, path, 1)).To(Succeed())
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
			Expect(crush.ExtractZIP(in, path, 0)).To(Succeed())
			Expect(filepath.Join(path, "fileA.txt")).To(BeARegularFile())
			Expect(filepath.Join(path, "dirA", "fileB.txt")).To(BeARegularFile())
			Expect(filepath.Join(path, "dirA", "fileC.txt")).To(BeARegularFile())
		})

		it("skips stripped components", func() {
			Expect(crush.ExtractZIP(in, path, 1)).To(Succeed())
			Expect(filepath.Join(path, "fileB.txt")).To(BeARegularFile())
			Expect(filepath.Join(path, "fileC.txt")).To(BeARegularFile())
		})
	})
}
