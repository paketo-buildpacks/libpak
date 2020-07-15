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

package libpak

import (
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// MappingsFile defines dependency mappings for a set of buildpacks
type MappingsFile struct {
	BuildpackMappings []BuildpackMappings `toml:"buildpacks"`
}

type BuildpackMappings struct {
	BuildpackID string              `toml:"id"`
	Mappings    []DependencyMapping `toml:"mappings"`
}

// DependencyMapping defines a new URI for a dependency with the given ID and Version
type DependencyMapping struct {
	ID      string `toml:"id"`
	Version string `toml:"version"`
	URI     string `toml:"uri"`
}

// ReadMappingsFile read MappingsFile from path
func ReadMappingsFile(path string) (MappingsFile, error) {
	mappingsFile := MappingsFile{}
	_, err := toml.DecodeFile(path, &mappingsFile)
	if err != nil {
		return MappingsFile{}, err
	}
	return mappingsFile, nil
}

func DefaultMappingsFilePath(platformDir string) string {
	return filepath.Join(platformDir, "dependencies", "mappings.toml")
}
