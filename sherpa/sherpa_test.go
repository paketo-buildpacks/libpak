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
	"bytes"
	"fmt"
	"testing"

	mocks1 "github.com/buildpacks/libcnb/mocks"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"

	"github.com/paketo-buildpacks/libpak/sherpa"
	mocks2 "github.com/paketo-buildpacks/libpak/sherpa/mocks"
)

func testSherpa(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		exitHandler *mocks1.ExitHandler
	)

	it.Before(func() {
		exitHandler = &mocks1.ExitHandler{}
		exitHandler.On("Error", mock.Anything)
	})

	context("Execute", func() {

		it("exits with error", func() {
			sherpa.Execute(func() error {
				return fmt.Errorf("test-error")
			}, sherpa.WithExitHandler(exitHandler))

			Expect(exitHandler.Calls).To(HaveLen(1))
		})

		it("exits without error", func() {
			sherpa.Execute(func() error {
				return nil
			}, sherpa.WithExitHandler(exitHandler))

			Expect(exitHandler.Calls).To(HaveLen(0))
		})

	})

	context("Helpers", func() {

		var (
			execd *mocks2.ExecD
		)

		it.Before(func() {
			execd = &mocks2.ExecD{}
		})

		it("returns error with no command name", func() {
			err := sherpa.Helpers(map[string]sherpa.ExecD{}, sherpa.WithArguments([]string{}))

			Expect(err).To(MatchError("expected command name"))
		})

		it("returns error with unsupported command name", func() {
			err := sherpa.Helpers(map[string]sherpa.ExecD{}, sherpa.WithArguments([]string{"test"}))

			Expect(err).To(MatchError("unsupported command test"))
		})

		it("returns error from helper", func() {
			execd.On("Execute").Return(nil, fmt.Errorf("test-error"))

			err := sherpa.Helpers(map[string]sherpa.ExecD{"test": execd}, sherpa.WithArguments([]string{"test"}))

			Expect(err).To(MatchError("test-error"))
		})

		it("handles nil environment map", func() {
			execd.On("Execute").Return(nil, nil)
			b := &bytes.Buffer{}

			err := sherpa.Helpers(map[string]sherpa.ExecD{"test": execd},
				sherpa.WithArguments([]string{"test"}),
				sherpa.WithExecdWriter(b),
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(b.String()).To(Equal(""))
		})

		it("writes output to fd/3", func() {
			execd.On("Execute").Return(map[string]string{"TEST_KEY": "test-value"}, nil)
			b := &bytes.Buffer{}

			err := sherpa.Helpers(map[string]sherpa.ExecD{"test": execd},
				sherpa.WithArguments([]string{"test"}),
				sherpa.WithExecdWriter(b),
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(b.String()).To(Equal("TEST_KEY=\"test-value\"\n"))
		})
	})
}
