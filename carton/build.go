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
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/libcnb"
	"github.com/heroku/color"
	"github.com/paketoio/libpak"
	"github.com/paketoio/libpak/bard"
	"github.com/paketoio/libpak/effect"
	"github.com/paketoio/libpak/internal"
)

// Context is an object that contains the context for execution.
type Context struct {

	// CacheLocation is the location to cache downloaded dependencies.
	CacheLocation string

	// IncludeDependencies indicates whether to include dependencies in build package.
	IncludeDependencies bool

	// Destination is the directory to create the build package in.
	Destination string

	// Source is the source directory of the buildpack.
	Source string

	// Version is a version to substitute into an existing buildpack.toml.
	Version string
}

type Build struct {
	Logger bard.Logger
}

// Build is the method called for packaging.
func (b Build) Build(context Context, options ...Option) {
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

	buildpack := libcnb.Buildpack{}
	file = filepath.Join(context.Source, "buildpack.toml")
	if _, err = toml.DecodeFile(file, &buildpack); err != nil && !os.IsNotExist(err) {
		config.exitHandler.Error(fmt.Errorf("unable to decode buildpack %s: %w", file, err))
		return
	}
	b.Logger.Debug("Buildpack: %+v", buildpack)

	metadata, err := libpak.NewBuildpackMetadata(buildpack.Metadata)
	if err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to decode metadata %s: %w", buildpack.Metadata, err))
		return
	}

	entries := map[string]string{}

	for _, i := range metadata.IncludeFiles {
		entries[i] = filepath.Join(context.Source, i)
	}
	b.Logger.Debug("Include files: %+v", entries)

	if context.Version != "" {
		buildpack.Info.Version = context.Version

		file = filepath.Join(context.Source, "buildpack.toml")
		t, err := template.ParseFiles(file)
		if err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to parse template %s: %w", file, err))
			return
		}

		out, err := ioutil.TempFile("", "buildpack-*.toml")
		if err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to open temporary buildpack.toml file: %w", err))
		}
		defer out.Close()

		if err = t.Execute(out, map[string]string{"Version": context.Version}); err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to execute template %s with version %s: %w", file, context.Version, err))
			return
		}

		entries["buildpack.toml"] = out.Name()
	}

	b.Logger.Title(buildpack)
	b.Logger.Header("Creating package in %s", context.Destination)

	if err = os.RemoveAll(context.Destination); err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to remove destination path %s: %w", context.Destination, err))
		return
	}

	file = metadata.PrePackage
	b.Logger.Header("Pre-package with %s", file)
	execution := effect.Execution{
		Command: file,
		Dir:     context.Source,
		Stdout:  b.Logger.BodyWriter(),
		Stderr:  b.Logger.BodyWriter(),
	}

	if err = config.executor.Execute(execution); err != nil {
		config.exitHandler.Error(fmt.Errorf("unable to execute pre-package script %s: %w", file, err))
	}

	if context.IncludeDependencies {
		cache := libpak.DependencyCache{
			Logger:    b.Logger,
			UserAgent: fmt.Sprintf("%s/%s", buildpack.Info.ID, buildpack.Info.Version),
		}

		if context.CacheLocation != "" {
			cache.DownloadPath = context.CacheLocation
		} else {
			cache.DownloadPath = filepath.Join(context.Source, "dependencies")
		}

		for _, dep := range metadata.Dependencies {
			b.Logger.Header("Caching %s", color.BlueString("%s %s", dep.Name, dep.Version))

			f, err := cache.Artifact(dep)
			if err != nil {
				config.exitHandler.Error(fmt.Errorf("unable to download %s: %w", dep.URI, err))
				return
			}
			if err = f.Close(); err != nil {
				config.exitHandler.Error(fmt.Errorf("unable to close %s: %w", f.Name(), err))
				return
			}

			entries[fmt.Sprintf("dependencies/%s/%s", dep.SHA256, filepath.Base(f.Name()))] = f.Name()
			entries[fmt.Sprintf("dependencies/%s.toml", dep.SHA256)] = fmt.Sprintf("%s.toml", filepath.Dir(f.Name()))
		}
	}

	var files []string
	for d, _ := range entries {
		files = append(files, d)
	}
	sort.Strings(files)
	for _, d := range files {
		b.Logger.Body("Adding %s", d)
		file = filepath.Join(context.Destination, d)
		if err = config.entryWriter.Write(entries[d], file); err != nil {
			config.exitHandler.Error(fmt.Errorf("unable to write file %s to %s: %w", entries[d], file, err))
			return
		}
	}
}
