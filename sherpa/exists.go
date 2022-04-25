/*
 * Copyright 2018-2022 the original author or authors.
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

import "os"

// Exists returns true if the path exists.
func Exists(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

// FileExists returns true if the path exists and is a regular file.
func FileExists(path string) (bool, error) {
	if stat, err := os.Stat(path); err == nil {
		return stat.Mode().IsRegular(), nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

// DirExists returns true if the path exists and is a directory.
func DirExists(path string) (bool, error) {
	if stat, err := os.Stat(path); err == nil {
		return stat.IsDir(), nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

// SymlinkExists returns true if the path exists and is a symlink.
func SymlinkExists(path string) (bool, error) {
	if stat, err := os.Lstat(path); err == nil {
		return stat.Mode()&os.ModeSymlink != 0, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}
