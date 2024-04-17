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
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/libcnb"
	"github.com/heroku/color"

	"github.com/paketo-buildpacks/libpak"
	"github.com/paketo-buildpacks/libpak/bard"
	"github.com/paketo-buildpacks/libpak/effect"
	"github.com/paketo-buildpacks/libpak/internal"
)

const DefaultTargetArch = "all"

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

	// TargetArch is the target architecture to package. Default is "all".
	TargetArch string
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

	logger := bard.NewLogger(os.Stdout)

	buildpack := libcnb.Buildpack{}
	file = filepath.Join(p.Source, "buildpack.toml")
	b, err := os.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		config.exitHandler.Error(fmt.Errorf("unable to read %s\n%w", file, err))
		return
	}
	if err := toml.Unmarshal(b, &buildpack); err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to decode buildpack %s\n%w", file, err))
		return
	}
	logger.Debugf("Buildpack: %+v", buildpack)

	metadata, err := libpak.NewBuildpackMetadata(buildpack.Metadata)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to decode metadata %s\n%w", buildpack.Metadata, err))
		return
	}

	logger.Debugf("IncludeFiles: %+v", metadata.IncludeFiles)

	supportedTargets := []string{}
	for _, i := range metadata.IncludeFiles {
		if strings.HasPrefix(i, "linux/") {
			parts := strings.SplitN(i, "/", 3)
			if len(parts) < 3 {
				// this shouldn't happen, but if it does for some reason just ignore it
				//   this entry is not a properly formatted target
				continue
			}
			supportedTargets = append(supportedTargets, fmt.Sprintf("%s/%s", parts[0], parts[1]))
		}
	}

	oldOutputFormat := len(supportedTargets) == 0
	if oldOutputFormat {
		logger.Info("No supported targets found, defaulting to old format")
	}

	logger.Debugf("Supported targets: %+v", supportedTargets)

	entries := map[string]string{}

	for _, i := range metadata.IncludeFiles {
		if oldOutputFormat || strings.HasPrefix(i, "linux/") || i == "buildpack.toml" {
			entries[i] = filepath.Join(p.Source, i)
		} else {
			for _, target := range supportedTargets {
				entries[fmt.Sprintf("%s/%s", target, i)] = filepath.Join(p.Source, i)
			}
		}
	}
	logger.Debugf("Include files: %+v", entries)

	if p.Version != "" {
		buildpack.Info.Version = p.Version

		file = filepath.Join(p.Source, "buildpack.toml")
		t, err := template.ParseFiles(file)
		if err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to parse template %s\n%w", file, err))
			return
		}

		out, err := os.CreateTemp("", "buildpack-*.toml")
		if err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to open temporary buildpack.toml file\n%w", err))
		}
		defer out.Close()

		if err = t.Execute(out, map[string]interface{}{"version": p.Version}); err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to execute template %s with version %s\n%w", file, p.Version, err))
			return
		}

		entries["buildpack.toml"] = out.Name()
	}

	logger.Title(buildpack)
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
			UserAgent: fmt.Sprintf("%s/%s", buildpack.Info.ID, buildpack.Info.Version),
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
				logger.Debugf("fetching dependency %s failed\n%w", dep.Name, err)
				config.exitHandler.Error(fmt.Errorf("unable to download %s. see DEBUG log level", dep.Name))
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
		if p.TargetArch != DefaultTargetArch && !oldOutputFormat && strings.HasPrefix(d, "linux/") && !strings.HasPrefix(d, fmt.Sprintf("linux/%s", p.TargetArch)) {
			logger.Debugf("Skipping %s because target arch is %s", d, p.TargetArch)
			continue
		}

		targetLocation := d
		if p.TargetArch != DefaultTargetArch {
			targetLocation = strings.Replace(d, fmt.Sprintf("linux/%s/", p.TargetArch), "", 1)
		}

		logger.Bodyf("Adding %s", targetLocation)
		file = filepath.Join(p.Destination, targetLocation)
		if err = config.entryWriter.Write(entries[d], file); err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to write file %s to %s\n%w", entries[d], file, err))
			return
		}
	}
}

// matchDependency checks all filters against dependency and returns true if there is a match (or no filters) and false if there is no match
// There is a match if a regular expression matches against the ID or Version
func (p Package) matchDependency(dep libpak.BuildpackDependency) bool {
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
