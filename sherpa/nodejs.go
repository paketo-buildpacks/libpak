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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// NodeJSMainModule returns the name of the main module as defined in <path>/package.json.
func NodeJSMainModule(path string) (string, error) {
	file := filepath.Join(path, "package.json")
	in, err := os.Open(file)
	if err != nil {
		return "", fmt.Errorf("unable to open %s\n%w", file, err)
	}
	defer in.Close()

	var raw map[string]interface{}
	if err := json.NewDecoder(in).Decode(&raw); err != nil {
		return "", fmt.Errorf("unable to decode %s\n%w", file, err)
	}

	m, ok := raw["main"].(string)
	if !ok {
		return "", fmt.Errorf("no main module defined in %s: %+v", file, raw)
	}

	return m, nil
}

