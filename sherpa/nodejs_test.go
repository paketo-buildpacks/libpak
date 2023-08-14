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
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak/v2/sherpa"
)

func testNodeJS(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		path string
	)

	it.Before(func() {
		path = t.TempDir()
	})

	it("returns server.js if no package.json exists", func() {
		Expect(sherpa.NodeJSMainModule(path)).To(Equal("server.js"))
	})

	it("returns server.js if package.json does not have a main entry", func() {
		Expect(os.WriteFile(filepath.Join(path, "package.json"), []byte(`{}`), 0644)).To(Succeed())

		Expect(sherpa.NodeJSMainModule(path)).To(Equal("server.js"))
	})

	it("returns main module", func() {
		Expect(os.WriteFile(filepath.Join(path, "package.json"), []byte(`{ "main": "test-main" }`), 0644)).
			To(Succeed())

		Expect(sherpa.NodeJSMainModule(path)).To(Equal("test-main"))
	})
}
