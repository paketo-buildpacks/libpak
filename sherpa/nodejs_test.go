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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak/sherpa"
	"github.com/sclevine/spec"
)

func testNodeJS(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		path string
	)

	it.Before(func() {
		var err error

		path, err = ioutil.TempDir("", "nodejs")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	context("NodeJSMainModule", func() {
		it("returns main module", func() {
			Expect(ioutil.WriteFile(filepath.Join(path, "package.json"), []byte(`{ "main": "test-main" }`), 0644)).
				To(Succeed())

			Expect(sherpa.NodeJSMainModule(path)).To(Equal("test-main"))
		})

		it("returns error if no main module defined", func() {
			Expect(ioutil.WriteFile(filepath.Join(path, "package.json"), []byte(`{}`), 0644)).To(Succeed())

			_, err := sherpa.NodeJSMainModule(path)
			Expect(err).To(MatchError(fmt.Errorf("no main module defined in %s: map[]", filepath.Join(path, "package.json"))))
		})
	})
}
