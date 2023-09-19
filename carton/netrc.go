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

package carton

import (
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

type Netrc []NetrcLine

type NetrcLine struct {
	Machine  string
	Login    string
	Password string
}

func (n Netrc) BasicAuth(request *http.Request) (*http.Request, error) {
	for _, l := range n {
		if l.Machine != request.Host && l.Machine != "default" {
			continue
		}

		request.SetBasicAuth(l.Login, l.Password)
		break
	}

	return request, nil
}

func ParseNetrc(path string) (Netrc, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("unable to open %s\n%w", path, err)
	}

	var (
		n Netrc
		l NetrcLine
		m = false
	)

	for _, line := range strings.Split(string(b), "\n") {
		if m {
			if line == "" {
				m = false
			}
			continue
		}

		f := strings.Fields(line)
		for i := 0; i < len(f); {
			switch f[i] {
			case "machine":
				l = NetrcLine{Machine: f[i+1]}
				i += 2
			case "default":
				l = NetrcLine{Machine: "default"}
				i += 1
			case "login":
				l.Login = f[i+1]
				i += 2
			case "password":
				l.Password = f[i+1]
				i += 2
			case "macdef":
				m = true
				i += 2
			}

			if l.Machine != "" && l.Login != "" && l.Password != "" {
				n = append(n, l)

				if l.Machine == "default" {
					return n, nil
				}

				l = NetrcLine{}
			}
		}
	}

	return n, nil
}

func NetrcPath() (string, error) {
	if s, ok := os.LookupEnv("NETRC"); ok {
		return s, nil
	}

	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("unable to determine user home directory\n%w", err)
	}

	return filepath.Join(u.HomeDir, ".netrc"), nil
}
