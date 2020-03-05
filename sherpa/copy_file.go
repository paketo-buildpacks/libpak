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
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyFile copies a file from the source to the destination. It ensures that the parent directory is created and
// matches the source and destination permissions.
func CopyFile(source *os.File, destination string) error {
	s, err := source.Stat()
	if err != nil {
		return fmt.Errorf("unable to stat %s: %w", source.Name(), err)
	}

	file := filepath.Dir(destination)
	if err := os.MkdirAll(file, 0755); err != nil {
		return fmt.Errorf("unable to create directory %s: %w", file, err)
	}

	out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, s.Mode())
	if err != nil {
		return fmt.Errorf("unable to open %s: %w", destination, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, source); err != nil {
		return fmt.Errorf("unable to copy from %s to %s: %w", source.Name(), destination, err)
	}

	return nil
}
