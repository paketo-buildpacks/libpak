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

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak/sherpa"
	"github.com/sclevine/spec"
)

func testResolveVersion(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	context("Environment Variable", func() {
		it.Before(func() {
			Expect(os.Setenv("TEST_ENV_VAR_KEY", "test-env-var-value"))
		})

		it.After(func() {
			Expect(os.Unsetenv("TEST_ENV_VAR_KEY"))
		})

		it("chooses environment variable", func() {
			v := sherpa.ResolveVersion("TEST_ENV_VAR_KEY", libcnb.BuildpackPlanEntry{}, "", nil)
			Expect(v).To(Equal("test-env-var-value"))
		})

	})

	it("chooses entry", func() {
		v := sherpa.ResolveVersion("", libcnb.BuildpackPlanEntry{Version: "test-buildpack-plan-entry-value"}, "", nil)
		Expect(v).To(Equal("test-buildpack-plan-entry-value"))
	})

	it("chooses default version", func() {
		v := sherpa.ResolveVersion("", libcnb.BuildpackPlanEntry{}, "test-default-versions-key", map[string]string{"test-default-versions-key": "test-default-versions-value"})
		Expect(v).To(Equal("test-default-versions-value"))
	})

	it("chooses empty version", func() {
		v := sherpa.ResolveVersion("", libcnb.BuildpackPlanEntry{}, "", map[string]string{})
		Expect(v).To(Equal(""))
	})
}
