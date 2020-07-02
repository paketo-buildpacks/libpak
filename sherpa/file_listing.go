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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// FileEntry is metadata about a file.
type FileEntry struct {

	// Path is the path of a file.
	Path string `toml:"path"`

	// Mode is the mode of the source file.
	Mode string `toml:"mode"`

	// SHA256 is the SHA256 hash of the source file.
	SHA256 string `toml:"sha256,omitempty"`
}

type result struct {
	err   error
	value FileEntry
}

// NewFileListing generates a listing of all entries under the roots.
func NewFileListing(roots ...string) ([]FileEntry, error) {
	entries := make(chan FileEntry)
	results := make(chan result)

	go func() {
		for _, root := range roots {
			p, err := filepath.EvalSymlinks(root)
			if os.IsNotExist(err) {
				continue
			} else if err != nil {
				results <- result{err: fmt.Errorf("unable to resolve %s\n%w", root, err)}
				return
			}

			if err := filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if path == p {
					return nil
				}

				e := FileEntry{
					Path:             path,
					Mode:             info.Mode().String(),
				}

				if info.IsDir() {
					results <- result{value: e}
					return nil
				}

				entries <- e
				return nil
			}); err != nil {
				results <- result{err: fmt.Errorf("error walking path %s\n%w", root, err)}
				return
			}
		}

		close(entries)
	}()

	go func() {
		var workers sync.WaitGroup
		for i := 0; i < 128; i++ {
			workers.Add(1)
			go worker(entries, results, &workers)
		}

		workers.Wait()
		close(results)
	}()

	var e []FileEntry
	for r := range results {
		if r.err != nil {
			return nil, fmt.Errorf("unable to create file listing\n%w", r.err)
		}
		e = append(e, r.value)
	}
	sort.Slice(e, func(i, j int) bool {
		return e[i].Path < e[j].Path
	})

	return e, nil
}

func worker(entries chan FileEntry, results chan result, wg *sync.WaitGroup) {
	for entry := range entries {
		e, err := process(entry)
		results <- result{value: e, err: err}
	}

	wg.Done()
}

func process(entry FileEntry) (FileEntry, error) {
	s := sha256.New()

	in, err := os.Open(entry.Path)
	if err != nil {
		return FileEntry{}, fmt.Errorf("unable to open file %s\n%w", entry.Path, err)
	}
	defer in.Close()

	if _, err := io.Copy(s, in); err != nil {
		return FileEntry{}, fmt.Errorf("unable to hash file %s\n%w", entry.Path, err)
	}

	entry.SHA256 = hex.EncodeToString(s.Sum(nil))
	return entry, nil
}
