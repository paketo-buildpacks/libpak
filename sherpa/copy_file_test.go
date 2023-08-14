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
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak/v2/sherpa"
)

func testCopyFile(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		source      *os.File
		destination string
	)

	it.Before(func() {
		var err error

		source, err = os.CreateTemp("", "copy-file")
		Expect(err).NotTo(HaveOccurred())
		_, err = source.WriteString("test")
		Expect(err).NotTo(HaveOccurred())
		Expect(source.Close()).To(Succeed())
		source, err = os.Open(source.Name())
		Expect(err).NotTo(HaveOccurred())

		f, err := os.CreateTemp("", "copy-file")
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).To(Succeed())
		destination = f.Name()
	})

	it.After(func() {
		Expect(os.RemoveAll(source.Name())).To(Succeed())
		Expect(os.RemoveAll(destination)).To(Succeed())
	})

	it("create listing", func() {
		defer source.Close()
		Expect(sherpa.CopyFile(source, destination)).To(Succeed())
		Expect(os.ReadFile(destination)).To(Equal([]byte("test")))
	})
}
