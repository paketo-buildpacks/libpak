/*
 * Copyright 2018-2025 the original author or authors.
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
	"os"
	"path/filepath"
)

// CopyDir copies the source directory to the destination. It ensures that the source and destination permissions match.
func CopyDir(source string, destination string) error {
	entries, err := os.ReadDir(source)
	if err != nil {
		return fmt.Errorf("unable to read dir '%s'\n%w", source, err)
	}

	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("unable to stat %s\n%w", source, err)
	}
	if err := os.MkdirAll(destination, info.Mode()); err != nil {
		return fmt.Errorf("unable to create directory %s\n%w", destination, err)
	}

	for _, entry := range entries {
		sourceEntry := filepath.Join(source, entry.Name())
		destinationEntry := filepath.Join(destination, entry.Name())
		if entry.IsDir() {
			entryInfo, err := entry.Info()
			if err != nil {
				return fmt.Errorf("unable to get directory info for %s\n%w", destinationEntry, err)
			}
			if err := os.Mkdir(destinationEntry, entryInfo.Mode().Perm()); err != nil {
				return fmt.Errorf("unable to create directory %s\n%w", destinationEntry, err)
			}
			if err := CopyDir(sourceEntry, destinationEntry); err != nil {
				return fmt.Errorf("unable to copy directory %s to %s\n%w", sourceEntry, destinationEntry, err)
			}
		} else {
			if err := copyFile(sourceEntry, destinationEntry); err != nil {
				return fmt.Errorf("unable to copy from %s to %s\n%w", sourceEntry, destinationEntry, err)
			}
		}
	}

	return nil
}

func copyFile(source string, destination string) error {
	file, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("unable to open source file %s\n%w", source, err)
	}
	defer file.Close()
	return CopyFile(file, destination)
}
