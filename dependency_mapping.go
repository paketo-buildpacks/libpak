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
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// DependencyMapping defines a new URI for a dependency with the given ID and Version
type DependencyMapping struct {
	ID      string `toml:"id"`
	Version string `toml:"version"`
	URI     string `toml:"uri"`
}

// ReadMappingsForBuildpack reads the mappings for the buildpack with ID buildpackID from the file at path
func ReadMappingsForBuildpack(path string, buildpackID string) ([]DependencyMapping, error) {
	mappingsFile := struct {
		Buildpacks []struct {
			ID       string              `toml:"id"`
			Mappings []DependencyMapping `toml:"mappings"`
		} `toml:"buildpacks"`
	}{}
	if _, err := toml.DecodeFile(path, &mappingsFile); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("unable to decode dependency mappings file%s\n%w", path, err)
	}
	for _, bps := range mappingsFile.Buildpacks {
		if bps.ID == buildpackID {
			return bps.Mappings, nil
		}
	}
	return nil, nil
}

// DefaultMappingsFilePath returns default path for mappings file
func DefaultMappingsFilePath(platformDir string) string {
	return filepath.Join(platformDir, "dependencies", "mappings.toml")
}
