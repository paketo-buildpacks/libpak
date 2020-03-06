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
	"testing"

	"github.com/buildpacks/libcnb/mocks"
	. "github.com/onsi/gomega"
	"github.com/paketoio/libpak/sherpa"
	"github.com/sclevine/spec"
	"github.com/stretchr/testify/mock"
)

func testSherpa(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		exitHandler *mocks.ExitHandler
	)

	it.Before(func() {
		exitHandler = &mocks.ExitHandler{}
		exitHandler.On("Error", mock.Anything)
	})

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
}
