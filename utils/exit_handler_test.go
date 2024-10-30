/*
 * Copyright 2018-2024 the original author or authors.
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

package utils_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak/v2/log"
	"github.com/paketo-buildpacks/libpak/v2/utils"
)

func testExitHandler(t *testing.T, _ spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		b        *bytes.Buffer
		exitCode int
		handler  utils.ExitHandler
	)

	it.Before(func() {
		b = bytes.NewBuffer([]byte{})

		handler = utils.NewExitHandler(
			utils.WithExitHandlerExitFunc(func(c int) { exitCode = c }),
			utils.WithExitHandlerLogger(log.NewPaketoLogger(b)),
			utils.WithExitHandlerWriter(b),
		)
	})

	it("exits with code 0 when passing", func() {
		handler.Pass()
		Expect(exitCode).To(Equal(0))
	})

	it("exits with code 100 when failing", func() {
		handler.Fail()
		Expect(exitCode).To(Equal(100))
	})

	it("exits with code 1 when the error is non-nil", func() {
		handler.Error(errors.New("failed"))
		Expect(exitCode).To(Equal(1))
	})

	it("writes the error message", func() {
		handler.Error(errors.New("test-message"))
		Expect(b).To(ContainSubstring("test-message"))
	})

	it("writes terminal error", func() {
		handler.Error(log.IdentifiableError{Name: "test-name", Description: "test-description", Err: fmt.Errorf("test-error")})
		Expect(b).To(ContainSubstring("\x1b[31m\x1b[0m\n\x1b[31m\x1b[1mtest-name\x1b[0m\x1b[31m test-description\x1b[0m\n\x1b[31;1m  test-error\x1b[0m\n"))
	})
}
