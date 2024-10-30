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

package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const ModeExecutable = 0100

type EntryWriter struct{}

func (e EntryWriter) Write(source string, destination string) error {
	p := filepath.Dir(destination)
	if err := os.MkdirAll(p, 0755); err != nil {
		return fmt.Errorf("unable to create destination directory %s\n%w", p, err)
	}

	s, err := os.Lstat(source)
	if err != nil {
		return fmt.Errorf("unable to stat file %s\n%w", source, err)
	}

	if s.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(source)
		if err != nil {
			return fmt.Errorf("unable to read link %s\n%w", source, err)
		}

		if err := os.Symlink(target, destination); err != nil {
			return fmt.Errorf("unable to create link %s\n%w", destination, err)
		}

		return nil
	}

	var perm os.FileMode
	if s.Mode()&ModeExecutable == ModeExecutable {
		perm = 0755
	} else {
		perm = 0644
	}

	in, err := os.OpenFile(source, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("unable to open source file %s\n%w", source, err)
	}
	defer in.Close()

	out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("unable to open destination file %s\n%w", destination, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("unable to copy %s to %s\n%w", source, destination, err)
	}

	return nil
}
