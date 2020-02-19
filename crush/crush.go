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

package crush

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xi2/xz"
)

type Crush struct{}

// ExtractTar extracts source TAR file to a destination directory.  An arbitrary number of top-level directory
// components can be stripped from each path.
func (c *Crush) ExtractTar(source io.Reader, destination string, stripComponents int) error {
	t := tar.NewReader(source)

	for {
		f, err := t.Next()
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("unable to read TAR file: %w", err)
		}

		target := c.strippedPath(f.Name, destination, stripComponents)
		if target == "" {
			continue
		}

		info := f.FileInfo()
		if info.IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("unable to make directory %s: %w", target, err)
			}
		} else if info.Mode()&os.ModeSymlink != 0 {
			if err := c.writeSymlink(f.Linkname, target); err != nil {
				return err
			}
		} else {
			if err := c.writeFile(t, target, info.Mode()); err != nil {
				return err
			}
		}
	}

	return nil
}

// ExtractTarGz extracts source GZIP'd TAR file to a destination directory.  An arbitrary number of top-level directory
// components can be stripped from each path.
func (c *Crush) ExtractTarGz(source io.Reader, destination string, stripComponents int) error {
	gz, err := gzip.NewReader(source)
	if err != nil {
		return fmt.Errorf("unable to create GZIP reader: %w", err)
	}
	defer gz.Close()

	return c.ExtractTar(gz, destination, stripComponents)
}

// ExtractTarXZ extracts source XZ'd TAR file to a destination directory.  An arbitrary number of top-level directory
// components can be stripped from each path.
func (c *Crush) ExtractTarXZ(source io.Reader, destination string, stripComponents int) error {
	xz, err := xz.NewReader(source, 0)
	if err != nil {
		return fmt.Errorf("unable to create XZ reader: %w", err)
	}

	return c.ExtractTar(xz, destination, stripComponents)
}

// ExtractZip extracts source ZIP file to a destination directory.  An arbitrary number of top-level directory
// components can be stripped from each path.
func (c *Crush) ExtractZIP(source *os.File, destination string, stripComponents int) error {
	stat, err := source.Stat()
	if err != nil {
		return fmt.Errorf("unable to stat %s: %w", source.Name(), err)
	}

	z, err := zip.NewReader(source, stat.Size())
	if err != nil {
		return err
	}

	for _, f := range z.File {
		target := c.strippedPath(f.Name, destination, stripComponents)
		if target == "" {
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		} else {
			if err := c.writeZipEntry(f, target); err != nil {
				return err
			}
		}
	}

	return nil
}

func (Crush) strippedPath(source string, destination string, stripComponents int) string {
	components := strings.Split(source, string(filepath.Separator))

	if len(components) <= stripComponents {
		return ""
	}

	return filepath.Join(append([]string{destination}, components[stripComponents:]...)...)
}

func (Crush) writeFile(source io.Reader, path string, perm os.FileMode) error {
	file := filepath.Dir(path)
	if err := os.MkdirAll(file, 0755); err != nil {
		return fmt.Errorf("unable to create directory %s: %w", file, err)
	}

	out, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("unable to open file %s: %w", path, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, source); err != nil {
		return fmt.Errorf("unable to write data to %s: %w", path, err)
	}

	return nil
}

func (c Crush) writeZipEntry(file *zip.File, path string) error {
	in, err := file.Open()
	if err != nil {
		return fmt.Errorf("unable to open %s: %w", file.Name, err)
	}
	defer in.Close()

	return c.writeFile(in, path, file.Mode())
}

func (Crush) writeSymlink(oldName string, newName string) error {
	file := filepath.Dir(newName)
	if err := os.MkdirAll(file, 0755); err != nil {
		return fmt.Errorf("unable to create directory %s: %w", file, err)
	}

	if err := os.Symlink(oldName, newName); err != nil {
		return fmt.Errorf("unable to create '%s' as symlink to '%s': %v", newName, oldName, err)
	}

	return nil
}
