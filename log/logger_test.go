/*
 * Copyright 2018-2025 the original author or authors.
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

package log_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak/v2/log"
)

func testLogLevel(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	context("ParseLogLevel", func() {
		it("parses 'none' to LogLevelNone", func() {
			Expect(log.ParseLogLevel("none")).To(Equal(log.LogLevelNone))
		})

		it("parses 'error' to LogLevelError", func() {
			Expect(log.ParseLogLevel("error")).To(Equal(log.LogLevelError))
		})

		it("parses 'info' to LogLevelInfo", func() {
			Expect(log.ParseLogLevel("info")).To(Equal(log.LogLevelInfo))
		})

		it("parses 'debug' to LogLevelDebug", func() {
			Expect(log.ParseLogLevel("debug")).To(Equal(log.LogLevelDebug))
		})

		it("is case-insensitive", func() {
			Expect(log.ParseLogLevel("ERROR")).To(Equal(log.LogLevelError))
			Expect(log.ParseLogLevel("Debug")).To(Equal(log.LogLevelDebug))
		})

		it("defaults unknown values to info", func() {
			Expect(log.ParseLogLevel("foobar")).To(Equal(log.LogLevelInfo))
			Expect(log.ParseLogLevel("")).To(Equal(log.LogLevelInfo))
		})
	})

	context("GetLogLevel", func() {
		it.After(func() {
			Expect(os.Unsetenv("BP_DEBUG")).To(Succeed())
			Expect(os.Unsetenv("BP_LOG_LEVEL")).To(Succeed())
		})

		it("returns debug when BP_DEBUG is set", func() {
			t.Setenv("BP_DEBUG", "true")
			Expect(log.GetLogLevel()).To(Equal(log.LogLevelDebug))
		})

		it("returns error when BP_LOG_LEVEL=error", func() {
			t.Setenv("BP_LOG_LEVEL", "error")
			Expect(log.GetLogLevel()).To(Equal(log.LogLevelError))
		})

		it("returns none when BP_LOG_LEVEL=none", func() {
			t.Setenv("BP_LOG_LEVEL", "none")
			Expect(log.GetLogLevel()).To(Equal(log.LogLevelNone))
		})

		it("returns info for unknown BP_LOG_LEVEL", func() {
			t.Setenv("BP_LOG_LEVEL", "foobar")
			Expect(log.GetLogLevel()).To(Equal(log.LogLevelInfo))
		})

		it("BP_DEBUG takes precedence over BP_LOG_LEVEL", func() {
			t.Setenv("BP_DEBUG", "true")
			t.Setenv("BP_LOG_LEVEL", "none")
			Expect(log.GetLogLevel()).To(Equal(log.LogLevelDebug))
		})

		it("returns info by default", func() {
			Expect(log.GetLogLevel()).To(Equal(log.LogLevelInfo))
		})
	})

	context("Logger with different log levels", func() {
		var (
			b *bytes.Buffer
			l log.Logger
		)

		it.Before(func() {
			b = bytes.NewBuffer(nil)
		})

		it.After(func() {
			Expect(os.Unsetenv("BP_LOG_LEVEL")).To(Succeed())
		})

		context("with BP_LOG_LEVEL=none", func() {
			it.Before(func() {
				t.Setenv("BP_LOG_LEVEL", "none")
				l = log.NewPaketoLogger(b)
			})

			it("disables all logging", func() {
				l.Body("body")
				l.Header("header")
				l.Title("title", "v", "url")
				l.Debug("debug")
				l.TerminalError(log.IdentifiableError{Name: "n", Description: "d", Err: fmt.Errorf("e")})
				Expect(b.String()).To(BeEmpty())
			})

			it("indicates no log levels are enabled", func() {
				Expect(l.IsBodyEnabled()).To(BeFalse())
				Expect(l.IsHeaderEnabled()).To(BeFalse())
				Expect(l.IsTitleEnabled()).To(BeFalse())
				Expect(l.IsDebugEnabled()).To(BeFalse())
				Expect(l.IsTerminalErrorEnabled()).To(BeFalse())
			})
		})

		context("with BP_LOG_LEVEL=error", func() {
			it.Before(func() {
				t.Setenv("BP_LOG_LEVEL", "error")
				l = log.NewPaketoLogger(b)
			})

			it("enables only error logging", func() {
				l.Body("body")
				l.Header("header")
				l.Title("title", "v", "url")
				l.Debug("debug")
				l.TerminalError(log.IdentifiableError{Name: "test-error", Description: "d", Err: fmt.Errorf("e")})
				Expect(b.String()).To(ContainSubstring("test-error"))
				Expect(b.String()).NotTo(ContainSubstring("body"))
			})

			it("indicates only terminal error is enabled", func() {
				Expect(l.IsBodyEnabled()).To(BeFalse())
				Expect(l.IsHeaderEnabled()).To(BeFalse())
				Expect(l.IsTitleEnabled()).To(BeFalse())
				Expect(l.IsDebugEnabled()).To(BeFalse())
				Expect(l.IsTerminalErrorEnabled()).To(BeTrue())
			})
		})

		context("with BP_LOG_LEVEL=info", func() {
			it.Before(func() {
				t.Setenv("BP_LOG_LEVEL", "info")
				l = log.NewPaketoLogger(b)
			})

			it("enables info and error logging", func() {
				l.Body("body")
				l.Header("header")
				l.Title("title", "v", "url")
				l.Debug("debug")
				Expect(b.String()).To(ContainSubstring("body"))
				Expect(b.String()).To(ContainSubstring("header"))
				Expect(b.String()).To(ContainSubstring("title"))
				Expect(b.String()).NotTo(ContainSubstring("debug"))
			})

			it("indicates info and error are enabled but not debug", func() {
				Expect(l.IsBodyEnabled()).To(BeTrue())
				Expect(l.IsHeaderEnabled()).To(BeTrue())
				Expect(l.IsTitleEnabled()).To(BeTrue())
				Expect(l.IsDebugEnabled()).To(BeFalse())
				Expect(l.IsTerminalErrorEnabled()).To(BeTrue())
			})
		})

		context("with BP_LOG_LEVEL=debug", func() {
			it.Before(func() {
				t.Setenv("BP_LOG_LEVEL", "debug")
				l = log.NewPaketoLogger(b)
			})

			it("enables all logging", func() {
				l.Debug("debug")
				Expect(b.String()).To(ContainSubstring("debug"))
			})

			it("indicates all log levels are enabled", func() {
				Expect(l.IsBodyEnabled()).To(BeTrue())
				Expect(l.IsHeaderEnabled()).To(BeTrue())
				Expect(l.IsTitleEnabled()).To(BeTrue())
				Expect(l.IsDebugEnabled()).To(BeTrue())
				Expect(l.IsTerminalErrorEnabled()).To(BeTrue())
			})
		})
	})
}

func testLogger(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		b *bytes.Buffer
		l log.Logger
	)

	it.Before(func() {
		b = bytes.NewBuffer(nil)
	})

	context("without BP_DEBUG", func() {
		it.Before(func() {
			l = log.NewPaketoLogger(b)
		})

		it("does not configure debug", func() {
			Expect(l.IsDebugEnabled()).To(BeFalse())
		})
	})

	context("with BP_DEBUG", func() {
		it.Before(func() {
			// libcnb defines BP_DEBUG as enabled if it has _any_ value
			// this does not include empty string as previously tested here.
			t.Setenv("BP_DEBUG", "true")
			l = log.NewPaketoLogger(b)
		})

		it("configures debug", func() {
			Expect(l.IsDebugEnabled()).To(BeTrue())
		})
	})

	context("with BP_LOG_LEVEL set to DEBUG", func() {
		it.Before(func() {
			t.Setenv("BP_LOG_LEVEL", "DEBUG")
			l = log.NewPaketoLogger(b)
		})

		it("configures debug", func() {
			Expect(l.IsDebugEnabled()).To(BeTrue())
		})
	})

	context("with debug disabled", func() {
		it.Before(func() {
			Expect(os.Unsetenv("BP_LOG_LEVEL")).To(Succeed())
			l = log.NewPaketoLoggerWithOptions(b)
		})

		it("does not write debug log", func() {
			l.Debug("test-message")
			Expect(b.String()).To(Equal(""))
		})

		it("does not write debug formatted log", func() {
			l.Debugf("test-%s", "message")
			Expect(b.String()).To(Equal(""))
		})

		it("indicates that debug is not enabled", func() {
			Expect(l.IsDebugEnabled()).To(BeFalse())
		})
	})

	context("with debug enabled", func() {
		it.Before(func() {
			t.Setenv("BP_LOG_LEVEL", "debug")
			l = log.NewPaketoLogger(b)
		})

		it("writes body log", func() {
			l.Body("test-message-1\ntest-message-2")
			Expect(b.String()).To(Equal("\x1b[2m    test-message-1\x1b[0m\n\x1b[2m    test-message-2\x1b[0m\n"))
		})

		it("writes body formatted log", func() {
			l.Bodyf("test-%s\ntest-%s", "message-1", "message-2")
			Expect(b.String()).To(Equal("\x1b[2m    test-message-1\x1b[0m\n\x1b[2m    test-message-2\x1b[0m\n"))
		})

		it("returns body writer", func() {
			Expect(l.BodyWriter()).NotTo(BeNil())
		})

		it("indicates that body is enabled", func() {
			Expect(l.IsBodyEnabled()).To(BeTrue())
		})

		it("writes debug log", func() {
			l.Debug("test-message")
			Expect(b.String()).To(Equal("\x1b[46mtest-message\x1b[0m\n"))
		})

		it("writes debug formatted log", func() {
			l.Debugf("test-%s", "message")
			Expect(b.String()).To(Equal("\x1b[46mtest-message\x1b[0m\n"))
		})

		it("returns debug writer", func() {
			Expect(l.DebugWriter()).NotTo(BeNil())
		})

		it("indicates that debug is enabled", func() {
			Expect(l.IsDebugEnabled()).To(BeTrue())
		})

		it("writes header log", func() {
			l.Header("test-message-1\ntest-message-2")
			Expect(b.String()).To(Equal("  test-message-1\n  test-message-2\n"))
		})

		it("writes header formatted log", func() {
			l.Headerf("test-%s\ntest-%s", "message-1", "message-2")
			Expect(b.String()).To(Equal("  test-message-1\n  test-message-2\n"))
		})

		it("returns header writer", func() {
			Expect(l.HeaderWriter()).NotTo(BeNil())
		})

		it("writes terminal error", func() {
			l.TerminalError(log.IdentifiableError{Name: "test-name", Description: "test-description", Err: fmt.Errorf("test-error")})
			Expect(b.String()).To(Equal("\x1b[31m\x1b[0m\n\x1b[31m\x1b[1mtest-name\x1b[0m\x1b[31m test-description\x1b[0m\n\x1b[31;1m  test-error\x1b[0m\n"))
		})

		it("returns terminal error writer", func() {
			Expect(l.TerminalErrorWriter()).NotTo(BeNil())
		})

		it("indicates that terminal error is enabled", func() {
			Expect(l.IsTerminalErrorEnabled()).To(BeTrue())
		})

		it("writes title log", func() {
			l.Title("test-name", "test-version", "test-homepage")
			Expect(b.String()).To(Equal("\x1b[34m\x1b[0m\n\x1b[34m\x1b[1mtest-name\x1b[0m\x1b[34m test-version\x1b[0m\n  \x1b[34;2;3mtest-homepage\x1b[0m\n"))
		})

		it("returns title writer", func() {
			Expect(l.TitleWriter()).NotTo(BeNil())
		})

		it("indicates that title is enabled", func() {
			Expect(l.IsTitleEnabled()).To(BeTrue())
		})
	})
}
