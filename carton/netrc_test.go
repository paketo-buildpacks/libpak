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

package carton_test

import (
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak/v2/carton"
)

func testNetrc(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		path string
	)

	it.Before(func() {
		var err error

		f, err := os.CreateTemp("", "netrc")
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).To(Succeed())
		path = f.Name()
	})

	it.After(func() {
		Expect(os.RemoveAll(path)).To(Succeed())
	})

	context("path", func() {
		context("$NETRC", func() {
			it.Before(func() {
				Expect(os.Setenv("NETRC", "test-value")).To(Succeed())
			})

			it.After(func() {
				Expect(os.Unsetenv("NETRC")).To(Succeed())
			})

			it("returns value from env var", func() {
				Expect(carton.NetrcPath()).To(Equal("test-value"))
			})
		})

		it("returns default", func() {
			u, err := user.Current()
			Expect(err).NotTo(HaveOccurred())

			Expect(carton.NetrcPath()).To(Equal(filepath.Join(u.HomeDir, ".netrc")))
		})
	})

	context("parse", func() {
		it("parses one-liner", func() {
			Expect(os.WriteFile(path, []byte(`machine test-machine login test-login password test-password`), 0600)).To(Succeed())

			Expect(carton.ParseNetrc(path)).To(Equal(carton.Netrc{
				{
					Machine:  "test-machine",
					Login:    "test-login",
					Password: "test-password",
				},
			}))
		})

		it("parses multi-liner", func() {
			Expect(os.WriteFile(path, []byte(`
machine test-machine 
login test-login 
password test-password
`), 0600)).To(Succeed())

			Expect(carton.ParseNetrc(path)).To(Equal(carton.Netrc{
				{
					Machine:  "test-machine",
					Login:    "test-login",
					Password: "test-password",
				},
			}))
		})

		it("ignores macdef", func() {
			Expect(os.WriteFile(path, []byte(`
macdef uploadtest
	cd /pub/tests
	bin
	put filename.tar.gz
	quit

machine test-machine login test-login password test-password
`), 0600)).To(Succeed())

			Expect(carton.ParseNetrc(path)).To(Equal(carton.Netrc{
				{
					Machine:  "test-machine",
					Login:    "test-login",
					Password: "test-password",
				},
			}))
		})

		it("ignores all after default", func() {
			Expect(os.WriteFile(path, []byte(`
machine test-machine-1 login test-login-1 password test-password-1

default
login test-login-2
password test-password-2

machine test-machine-3 login test-login-3 password test-password-3
`), 0600)).To(Succeed())

			Expect(carton.ParseNetrc(path)).To(Equal(carton.Netrc{
				{
					Machine:  "test-machine-1",
					Login:    "test-login-1",
					Password: "test-password-1",
				},
				{
					Machine:  "default",
					Login:    "test-login-2",
					Password: "test-password-2",
				},
			}))
		})
	})

	context("basic auth", func() {
		it("does not apply auth if no candidates", func() {
			n := carton.Netrc{
				{
					Machine:  "test-machine",
					Login:    "test-login",
					Password: "test-password",
				},
			}

			req, err := http.NewRequest("GET", "http://another-machine", nil)
			Expect(err).NotTo(HaveOccurred())

			req, err = n.BasicAuth(req)
			Expect(err).NotTo(HaveOccurred())

			_, _, ok := req.BasicAuth()
			Expect(ok).To(BeFalse())
		})

		it("applies basic auth for match", func() {
			n := carton.Netrc{
				{
					Machine:  "test-machine",
					Login:    "test-login",
					Password: "test-password",
				},
			}

			req, err := http.NewRequest("GET", "http://test-machine", nil)
			Expect(err).NotTo(HaveOccurred())

			req, err = n.BasicAuth(req)
			Expect(err).NotTo(HaveOccurred())

			u, p, ok := req.BasicAuth()
			Expect(ok).To(BeTrue())
			Expect(u).To(Equal("test-login"))
			Expect(p).To(Equal("test-password"))
		})

		it("applies basic auth for default", func() {
			n := carton.Netrc{
				{
					Machine:  "test-machine",
					Login:    "test-login",
					Password: "test-password",
				},
				{
					Machine:  "default",
					Login:    "default-login",
					Password: "default-password",
				},
			}

			req, err := http.NewRequest("GET", "http://another-machine", nil)
			Expect(err).NotTo(HaveOccurred())

			req, err = n.BasicAuth(req)
			Expect(err).NotTo(HaveOccurred())

			u, p, ok := req.BasicAuth()
			Expect(ok).To(BeTrue())
			Expect(u).To(Equal("default-login"))
			Expect(p).To(Equal("default-password"))
		})
	})
}
