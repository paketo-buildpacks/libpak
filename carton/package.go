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

package carton

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/libcnb/v2"
	"github.com/heroku/color"

	"github.com/paketo-buildpacks/libpak/v2"
	"github.com/paketo-buildpacks/libpak/v2/effect"
	"github.com/paketo-buildpacks/libpak/v2/internal"
	"github.com/paketo-buildpacks/libpak/v2/log"
)

// Package is an object that contains the configuration for building a package.
type Package struct {

	// CacheLocation is the location to cache downloaded dependencies.
	CacheLocation string

	// DependencyFilters indicates which filters should be applied to exclude dependencies
	DependencyFilters []string

	// StrictDependencyFilters indicates that a filter must match both the ID and version, otherwise it must only match one of the two
	StrictDependencyFilters bool

	// IncludeDependencies indicates whether to include dependencies in build package.
	IncludeDependencies bool

	// Destination is the directory to create the build package in.
	Destination string

	// Source is the source directory of the buildpack.
	Source string

	// Version is a version to substitute into an existing buildpack.toml.
	Version string
}

// Create creates a package.
func (p Package) Create(options ...Option) {
	config := Config{
		entryWriter: internal.EntryWriter{},
		executor:    effect.NewExecutor(),
		exitHandler: internal.NewExitHandler(),
	}

	for _, option := range options {
		config = option(config)
	}

	var (
		err  error
		file string
	)

	logger := log.NewPaketoLogger(os.Stdout)

	// Is this a buildpack or an extension?
	bpfile := filepath.Join(p.Source, "buildpack.toml")
	extnfile := filepath.Join(p.Source, "extension.toml")
	var metadataMap map[string]interface{}
	var id string
	var name string
	var version string
	var homepage string
	extension := false
	if _, err := os.Stat(bpfile); err == nil {
		s, err := os.ReadFile(bpfile)
		if err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to read buildpack.toml %s\n%w", bpfile, err))
			return
		}
		var b libcnb.Buildpack
		if err := toml.Unmarshal(s, &b); err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to decode %s\n%w", bpfile, err))
			return
		}
		metadataMap = b.Metadata
		id = b.Info.ID
		name = b.Info.Name
		version = b.Info.Version
		homepage = b.Info.Homepage
		logger.Debugf("Buildpack: %+v", b)
	} else if _, err := os.Stat(extnfile); err == nil {
		s, err := os.ReadFile(extnfile)
		if err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to read extension.toml %s\n%w", extnfile, err))
			return
		}
		var e libcnb.Extension
		if err := toml.Unmarshal(s, &e); err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to decode %s\n%w", extnfile, err))
			return
		}
		metadataMap = e.Metadata
		id = e.Info.ID
		name = e.Info.Name
		version = e.Info.Version
		homepage = e.Info.Homepage
		extension = true
		logger.Debugf("Extension: %+v", e)
	} else {
		config.exitHandler.Error(fmt.Errorf("unable to read buildpack/extension.toml at %s", p.Source))
		return
	}

	metadata, err := libpak.NewBuildModuleMetadata(metadataMap)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to decode metadata %s\n%w", metadataMap, err))
		return
	}

	entries := map[string]string{}

	for _, i := range metadata.IncludeFiles {
		entries[i] = filepath.Join(p.Source, i)
	}
	logger.Debugf("Include files: %+v", entries)

	if p.Version != "" {
		version = p.Version

		tomlName := ""
		if extension {
			tomlName = "extension"
		} else {
			tomlName = "buildpack"
		}

		file = filepath.Join(p.Source, tomlName+".toml")
		t, err := template.ParseFiles(file)
		if err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to parse template %s\n%w", file, err))
			return
		}

		out, err := os.CreateTemp("", tomlName+"-*.toml")
		if err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to open temporary "+tomlName+".toml file\n%w", err))
		}
		defer out.Close()

		if err = t.Execute(out, map[string]interface{}{"version": p.Version}); err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to execute template %s with version %s\n%w", file, p.Version, err))
			return
		}

		entries[tomlName+".toml"] = out.Name()
	}

	logger.Title(name, version, homepage)
	logger.Headerf("Creating package in %s", p.Destination)

	if err = os.RemoveAll(p.Destination); err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to remove destination path %s\n%w", p.Destination, err))
		return
	}

	file = metadata.PrePackage
	if file != "" {
		logger.Headerf("Pre-package with %s", file)
		execution := effect.Execution{
			Command: file,
			Dir:     p.Source,
			Stdout:  logger.BodyWriter(),
			Stderr:  logger.BodyWriter(),
		}

		if err = config.executor.Execute(execution); err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to execute pre-package script %s\n%w", file, err))
		}
	}

	if p.IncludeDependencies {
		cache := libpak.DependencyCache{
			Logger:    logger,
			UserAgent: fmt.Sprintf("%s/%s", id, version),
		}

		if p.CacheLocation != "" {
			cache.DownloadPath = p.CacheLocation
		} else {
			cache.DownloadPath = filepath.Join(p.Source, "dependencies")
		}

		np, err := NetrcPath()
		if err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to determine netrc path\n%w", err))
			return
		}

		n, err := ParseNetrc(np)
		if err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to read %s as netrc\n%w", np, err))
			return
		}

		for _, dep := range metadata.Dependencies {
			if !p.matchDependency(dep) {
				logger.Bodyf("Skipping [%s or %s] which matched a filter", dep.ID, dep.Version)
				continue
			}

			logger.Headerf("Caching %s", color.BlueString("%s %s", dep.Name, dep.Version))

			f, err := cache.Artifact(dep, n.BasicAuth)
			if err != nil {
				config.exitHandler.Error(fmt.Errorf("unable to download %s\n%w", dep.URI, err))
				return
			}
			if err = f.Close(); err != nil {
				config.exitHandler.Error(fmt.Errorf("unable to close %s\n%w", f.Name(), err))
				return
			}

			entries[fmt.Sprintf("dependencies/%s/%s", dep.SHA256, filepath.Base(f.Name()))] = f.Name()
			entries[fmt.Sprintf("dependencies/%s.toml", dep.SHA256)] = fmt.Sprintf("%s.toml", filepath.Dir(f.Name()))
		}
	}

	var files []string
	for d := range entries {
		files = append(files, d)
	}
	sort.Strings(files)
	for _, d := range files {
		logger.Bodyf("Adding %s", d)
		file = filepath.Join(p.Destination, d)
		if err = config.entryWriter.Write(entries[d], file); err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to write file %s to %s\n%w", entries[d], file, err))
			return
		}
	}
}

// matchDependency checks all filters against dependency and returns true if there is a match (or no filters) and false if there is no match
// There is a match if a regular expression matches against the ID or Version
func (p Package) matchDependency(dep libpak.BuildModuleDependency) bool {
	if len(p.DependencyFilters) == 0 {
		return true
	}

	for _, rawFilter := range p.DependencyFilters {
		filter := regexp.MustCompile(rawFilter)

		if (p.StrictDependencyFilters && filter.MatchString(dep.ID) && filter.MatchString(dep.Version)) ||
			(!p.StrictDependencyFilters && (filter.MatchString(dep.ID) || filter.MatchString(dep.Version))) {
			return true
		}
	}

	return false
}
