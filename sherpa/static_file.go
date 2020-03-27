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

package sherpa

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"text/template"

	"github.com/rakyll/statik/fs"
)

// StaticFile returns the contents of a file in a statik filesystem.
func StaticFile(name string) (string, error) {
	fs, err := fs.New()
	if err != nil {
		return "", fmt.Errorf("unable to load static files\n%w", err)
	}

	in, err := fs.Open(name)
	if err != nil {
		return "", fmt.Errorf("unable to open %s\n%w", name, err)
	}
	defer in.Close()

	b, err := ioutil.ReadAll(in)
	if err != nil {
		return "", fmt.Errorf("unable to read %s\n%w", name, err)
	}

	return string(b), nil
}

// TemplateFile uses the contents of a file in a statik filesystem as a text/template template and returns the template as applied to the data object.
func TemplateFile(name string, data interface{}) (string, error) {
	s, err := StaticFile(name)
	if err != nil {
		return "", fmt.Errorf("unable to read template %s\n%w", name, err)
	}

	t, err := template.New(name).Parse(s)
	if err != nil {
		return "", fmt.Errorf("unable to parse template %s\n%w", name, err)
	}

	out := &bytes.Buffer{}
	if err := t.Execute(out, data); err != nil {
		return "", fmt.Errorf("unable to execute template %s with %+v\n%w", name, data, err)
	}

	return out.String(), nil
}
