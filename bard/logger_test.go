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

package bard_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/buildpacks/libcnb"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/sclevine/spec"
)

func testLogger(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		b *bytes.Buffer
		l bard.Logger
	)

	it.Before(func() {
		b = bytes.NewBuffer(nil)
	})

	context("without BP_DEBUG", func() {
		it.Before(func() {
			l = bard.NewLogger(b)
		})

		it("does not configure debug", func() {
			Expect(l.IsDebugEnabled()).To(BeFalse())
		})
	})

	context("with BP_DEBUG", func() {
		it.Before(func() {
			Expect(os.Setenv("BP_DEBUG", "")).To(Succeed())
			l = bard.NewLogger(b)
		})

		it.After(func() {
			Expect(os.Unsetenv("BP_DEBUG")).To(Succeed())
		})

		it("configures debug", func() {
			Expect(l.IsDebugEnabled()).To(BeTrue())
		})
	})

	context("with debug disabled", func() {
		it.Before(func() {
			l = bard.NewLoggerWithOptions(b)
		})

		it("does not write debug log", func() {
			l.Debug("test-%s", "message")
			Expect(b.String()).To(Equal(""))
		})

		it("does not return debug writer", func() {
			Expect(l.DebugWriter()).To(BeNil())
		})

		it("indicates that debug is not enabled", func() {
			Expect(l.IsDebugEnabled()).To(BeFalse())
		})

		it("writes info log", func() {
			l.Info("test-%s", "message")
			Expect(b.String()).To(Equal("test-message\n"))
		})

		it("returns info writer", func() {
			Expect(l.InfoWriter()).NotTo(BeNil())
		})

		it("indicates that info is enabled", func() {
			Expect(l.IsInfoEnabled()).To(BeTrue())
		})
	})

	context("with debug enabled", func() {
		it.Before(func() {
			l = bard.NewLoggerWithOptions(b, bard.WithDebug(b))
		})

		it("writes body log", func() {
			l.Body("test-%s\ntest-%s", "message-1", "message-2")
			Expect(b.String()).To(Equal("\x1b[2m    test-message-1\x1b[0m\n\x1b[2m    test-message-2\x1b[0m\n"))
		})

		it("returns body writer", func() {
			Expect(l.BodyWriter()).NotTo(BeNil())
		})

		it("writes debug log", func() {
			l.Debug("test-%s", "message")
			Expect(b.String()).To(Equal("test-message\n"))
		})

		it("indicates that body is enabled", func() {
			Expect(l.IsBodyEnabled()).To(BeTrue())
		})

		it("returns debug writer", func() {
			Expect(l.DebugWriter()).NotTo(BeNil())
		})

		it("indicates that debug is enabled", func() {
			Expect(l.IsDebugEnabled()).To(BeTrue())
		})

		it("writes info log", func() {
			l.Info("test-%s", "message")
			Expect(b.String()).To(Equal("test-message\n"))
		})

		it("returns info writer", func() {
			Expect(l.InfoWriter()).NotTo(BeNil())
		})

		it("indicates that info is enabled", func() {
			Expect(l.IsInfoEnabled()).To(BeTrue())
		})

		it("writes terminal error", func() {
			l.TerminalError(bard.IdentifiableError{Name: "test-name", Description: "test-description", Err: fmt.Errorf("test-error")})
			Expect(b.String()).To(Equal("\x1b[31m\x1b[0m\n\x1b[31m\x1b[1mtest-name\x1b[0m\x1b[31m test-description\x1b[0m\n\x1b[31;1m  test-error\x1b[0m\n"))
		})

		it("returns terminal error writer", func() {
			Expect(l.TerminalErrorWriter()).NotTo(BeNil())
		})

		it("indicates that terminal error is enabled", func() {
			Expect(l.IsTerminalErrorEnabled()).To(BeTrue())
		})

		it("writes title log", func() {
			l.Title(libcnb.Buildpack{Info: libcnb.BuildpackInfo{Name: "test-name", Version: "test-version"}})
			Expect(b.String()).To(Equal("\x1b[34m\x1b[0m\n\x1b[34m\x1b[1mtest-name\x1b[0m\x1b[34m test-version\x1b[0m\n"))
		})

		it("returns title writer", func() {
			Expect(l.TitleWriter()).NotTo(BeNil())
		})

		it("indicates that title is enabled", func() {
			Expect(l.IsTitleEnabled()).To(BeTrue())
		})
	})
}
