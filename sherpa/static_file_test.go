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
	"testing"

	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak/sherpa"
	"github.com/sclevine/spec"

	_ "github.com/paketo-buildpacks/libpak/sherpa/testdata/statik"
)

//go:generate statik -src testdata -dest testdata -include *.txt

func testStaticFile(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	it("reads static file", func() {
		Expect(sherpa.StaticFile("/test-file.txt")).To(Equal("fixture-marker-{{.value}}\n"))
	})

	it("reads template file", func() {
		Expect(sherpa.TemplateFile("/test-file.txt", map[string]string{"value": "alpha"})).To(Equal("fixture-marker-alpha\n"))
	})
}
