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

package internal_test

import (
	"io/ioutil"
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak/internal"
	"github.com/sclevine/spec"
)

func testEntryWriter(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		destination string
		source      string
		writer      internal.EntryWriter
	)

	it.Before(func() {
		var (
			err error
			f   *os.File
		)

		f, err = ioutil.TempFile("", "entry-writer-source")
		Expect(err).NotTo(HaveOccurred())
		_, err = f.WriteString("test-value")
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).To(Succeed())
		source = f.Name()

		f, err = ioutil.TempFile("", "entry-writer-destination")
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).To(Succeed())
		Expect(os.RemoveAll(f.Name())).To(Succeed())
		destination = f.Name()

		writer = internal.EntryWriter{}
	})

	it.After(func() {
		Expect(os.RemoveAll(source)).To(Succeed())
		Expect(os.RemoveAll(destination)).To(Succeed())
	})

	it("writes file", func() {
		Expect(writer.Write(source, destination)).To(Succeed())
		Expect(ioutil.ReadFile(destination)).To(Equal([]byte("test-value")))
	})

	it("sets executable bit", func() {
		Expect(os.Chmod(source, 0755)).To(Succeed())

		Expect(writer.Write(source, destination)).To(Succeed())

		s, err := os.Stat(destination)
		Expect(err).NotTo(HaveOccurred())
		Expect(s.Mode()&0100 == 0100).To(BeTrue())
	})

	context("symlink", func() {
		var (
			symlinkSource string
		)

		it.Before(func() {
			f, err := ioutil.TempFile("", "entry-writer-symlink-source")
			Expect(err).NotTo(HaveOccurred())
			Expect(f.Close()).To(Succeed())
			Expect(os.RemoveAll(f.Name())).To(Succeed())
			Expect(os.Symlink(source, f.Name())).To(Succeed())
			symlinkSource = f.Name()
		})

		it.After(func() {
			Expect(os.RemoveAll(symlinkSource)).To(Succeed())
		})

		it("replicates symlink", func() {
			Expect(writer.Write(symlinkSource, destination)).To(Succeed())

			Expect(os.Readlink(destination)).To(Equal(source))
		})
	})

}
