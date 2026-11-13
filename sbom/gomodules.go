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

package sbom

import (
	"debug/buildinfo"
	"fmt"
	"runtime/debug"
	"strings"
)

// GoModuleComponents lists the standard library and the modules a Go binary is built from, read from its
// embedded build information. owner keeps bom-refs unique across binaries embedding the same modules.
func GoModuleComponents(path string, owner string) ([]CycloneDXComponent, error) {
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read Go build information from %s\n%w", path, err)
	}

	component := func(name string, version string, purl string) CycloneDXComponent {
		return CycloneDXComponent{
			Type:    "library",
			BOMRef:  fmt.Sprintf("%s?package-id=%s", purl, owner),
			Name:    name,
			Version: version,
			PURL:    purl,
		}
	}

	var components []CycloneDXComponent

	// the same name, version, purl and CPE as Syft gives the standard library, which is what scanners match on
	if version := strings.TrimPrefix(info.GoVersion, "go"); version != "" {
		stdlib := component("stdlib", info.GoVersion, fmt.Sprintf("pkg:golang/stdlib@%s", version))
		stdlib.CPE = fmt.Sprintf("cpe:2.3:a:golang:go:%s:-:*:*:*:*:*:*", version)
		components = append(components, stdlib)
	}

	// a binary built from a checkout has no version for its main module, one built from a released module does
	modules := info.Deps
	if info.Main.Path != "" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		main := info.Main
		modules = append([]*debug.Module{&main}, modules...)
	}

	for _, dep := range modules {
		if dep.Replace != nil {
			dep = dep.Replace
		}
		components = append(components, component(dep.Path, dep.Version, fmt.Sprintf("pkg:golang/%s@%s", dep.Path, dep.Version)))
	}

	return components, nil
}
