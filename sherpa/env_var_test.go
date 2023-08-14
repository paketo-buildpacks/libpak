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

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak/v2/sherpa"
)

func testEnvVar(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	context("AppendToEnvVar", func() {

		context("No Existing", func() {

			it("append one", func() {
				Expect(sherpa.AppendToEnvVar("TEST_KEY", "|", "test-value-2")).
					To(Equal("test-value-2"))
			})

			it("appends multiple", func() {
				Expect(sherpa.AppendToEnvVar("TEST_KEY", "|", "test-value-2", "test-value-3")).
					To(Equal("test-value-2|test-value-3"))
			})
		})

		context("With Existing", func() {
			it.Before(func() {
				Expect(os.Setenv("TEST_KEY", "test-value-1")).To(Succeed())
			})

			it.After(func() {
				Expect(os.Unsetenv("TEST_KEY")).To(Succeed())
			})

			it("append one", func() {
				Expect(sherpa.AppendToEnvVar("TEST_KEY", "|", "test-value-2")).
					To(Equal("test-value-1|test-value-2"))
			})

			it("appends multiple", func() {
				Expect(sherpa.AppendToEnvVar("TEST_KEY", "|", "test-value-2", "test-value-3")).
					To(Equal("test-value-1|test-value-2|test-value-3"))
			})
		})
	})

	context("GetEnvRequired", func() {
		it.Before(func() {
			Expect(os.Setenv("TEST_KEY", "test-value")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("TEST_KEY")).To(Succeed())
		})

		it("returns value if set", func() {
			Expect(sherpa.GetEnvRequired("TEST_KEY")).To(Equal("test-value"))
		})

		it("returns error if not set", func() {
			_, err := sherpa.GetEnvRequired("ANOTHER_KEY")
			Expect(err).To(MatchError("$ANOTHER_KEY must be set"))
		})
	})

	context("GetEnvWithDefault", func() {
		it.Before(func() {
			Expect(os.Setenv("TEST_KEY", "test-value")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("TEST_KEY")).To(Succeed())
		})

		it("returns value if set", func() {
			Expect(sherpa.GetEnvWithDefault("TEST_KEY", "default-value")).To(Equal("test-value"))
		})

		it("returns default value if not set", func() {
			Expect(sherpa.GetEnvWithDefault("ANOTHER_KEY", "default-value")).To(Equal("default-value"))
		})
	})

	context("ResolveBoolErr", func() {
		context("variable not set", func() {
			it("returns false if not set", func() {
				boolean, err := sherpa.ResolveBoolErr("TEST_KEY")
				Expect(boolean).To(BeFalse())
				Expect(err).ToNot(HaveOccurred())
			})
		})

		context("variable is set to true value", func() {
			it.After(func() {
				Expect(os.Unsetenv("TEST_KEY")).To(Succeed())
			})

			it("returns true", func() {
				for _, form := range []string{"1", "t", "T", "TRUE", "true", "True", "\t1\n"} {
					Expect(os.Setenv("TEST_KEY", form))
					boolean, err := sherpa.ResolveBoolErr("TEST_KEY")
					Expect(boolean).To(BeTrue())
					Expect(err).ToNot(HaveOccurred())
					Expect(os.Unsetenv("TEST_KEY")).To(Succeed())
				}
			})
		})

		context("variable is set to non-true value", func() {
			it.After(func() {
				Expect(os.Unsetenv("TEST_KEY")).To(Succeed())
			})

			it("returns false", func() {
				for _, form := range []string{"0", "f", "F", "FALSE", "false", "False", "\tF\n"} {
					Expect(os.Setenv("TEST_KEY", form))
					boolean, err := sherpa.ResolveBoolErr("TEST_KEY")
					Expect(boolean).To(BeFalse())
					Expect(err).ToNot(HaveOccurred())
					Expect(os.Unsetenv("TEST_KEY")).To(Succeed())
				}
			})
		})

		context("variable is set to an invalid value", func() {
			it.After(func() {
				Expect(os.Unsetenv("TEST_KEY")).To(Succeed())
			})

			it("returns false", func() {
				Expect(os.Setenv("TEST_KEY", "foo"))
				boolean, err := sherpa.ResolveBoolErr("TEST_KEY")
				Expect(boolean).To(BeFalse())
				Expect(err).To(MatchError("invalid value 'foo' for key 'TEST_KEY': " +
					"expected one of [1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False]"))
				Expect(os.Unsetenv("TEST_KEY")).To(Succeed())
			})
		})

	})

	context("ResolveBool", func() {
		context("variable is set to an invalid value", func() {
			it.After(func() {
				Expect(os.Unsetenv("TEST_KEY")).To(Succeed())
			})

			it("returns false", func() {
				Expect(os.Setenv("TEST_KEY", "foo"))
				Expect(sherpa.ResolveBool("TEST_KEY")).To(BeFalse())
				Expect(os.Unsetenv("TEST_KEY")).To(Succeed())
			})
		})
	})
}
