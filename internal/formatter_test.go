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
	"fmt"
	"testing"

	"github.com/buildpacks/libcnb"
	"github.com/heroku/color"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak/internal"
	"github.com/sclevine/spec"
)

func testFormatter(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	context("LaunchFormatter", func() {

		it("is empty without slices or processes", func() {
			launch := libcnb.Launch{}

			Expect(internal.LaunchFormatter(launch).String()).
				To(Equal(""))
		})

		it("formats contents with slices", func() {
			launch := libcnb.Launch{
				Slices: []libcnb.Slice{{}, {}},
			}

			Expect(internal.LaunchFormatter(launch).String()).
				To(Equal("2 application slices"))
		})

		it("formats contents with processes", func() {
			launch := libcnb.Launch{
				Processes: []libcnb.Process{
					{
						Type:    "test-type",
						Command: "test-command",
					},
				},
			}

			Expect(internal.LaunchFormatter(launch).String()).
				To(Equal("Process types:\n  \x1b[36mtest-type\x1b[0m: test-command"))
		})

		it("formats contents with slices and processes", func() {
			launch := libcnb.Launch{
				Slices: []libcnb.Slice{{}, {}},
				Processes: []libcnb.Process{
					{
						Type:    "test-type",
						Command: "test-command",
					},
				},
			}

			Expect(internal.LaunchFormatter(launch).String()).
				To(Equal("2 application slices\nProcess types:\n  \x1b[36mtest-type\x1b[0m: test-command"))
		})
	})

	context("ProcessesFormatter", func() {

		it("aligns process types", func() {
			Expect(internal.ProcessesFormatter([]libcnb.Process{
				{Type: "short", Command: "test-command-1"},
				{Type: "a-very-long-type", Command: "test-command-2"},
			}).String()).To(Equal(
				fmt.Sprintf("  %s: test-command-2\n  %s:            test-command-1",
					color.CyanString("a-very-long-type"), color.CyanString("short"))))
		})

		it("appends arguments", func() {
			Expect(internal.ProcessesFormatter([]libcnb.Process{
				{Type: "test-type", Command: "test-command", Arguments: []string{"test-arg-1", "test-arg-2"}},
			}).String()).To(Equal(
				fmt.Sprintf("  %s: test-command test-arg-1 test-arg-2", color.CyanString("test-type"))))
		})

		it("indicates direct", func() {
			Expect(internal.ProcessesFormatter([]libcnb.Process{
				{Type: "test-type", Command: "test-command", Direct: true},
			}).String()).To(Equal(
				fmt.Sprintf("  %s: test-command (direct)", color.CyanString("test-type"))))
		})
	})

	context("SlicesFormatter", func() {

		it("formats contents", func() {
			slices := []libcnb.Slice{{}, {}}

			Expect(internal.SlicesFormatter(slices).String()).
				To(Equal("2 application slices"))
		})
	})

	context("StoreFormatter", func() {

		it("formats contents", func() {
			store := libcnb.Store{
				Metadata: map[string]interface{}{
					"test-key-1": "test-value-1",
					"test-key-2": "test-value-2",
				},
			}

			Expect(internal.StoreFormatter(store).String()).
				To(Equal("Writing persistent metadata: test-key-1, test-key-2"))
		})

	})
}
