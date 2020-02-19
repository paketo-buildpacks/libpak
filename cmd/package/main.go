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

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/paketoio/libpak/bard"
	"github.com/paketoio/libpak/carton"
	"github.com/spf13/pflag"
)

func main() {
	context := carton.Context{}

	flagSet := pflag.NewFlagSet("Build Package", pflag.ExitOnError)
	flagSet.StringVarP(&context.CacheLocation, "cache-location", "c", defaultCacheLocation(), "path to cache downloaded dependencies (default: $PWD/dependencies")
	flagSet.StringVarP(&context.Destination, "destination", "d", "", "path to the build package destination directory")
	flagSet.BoolVarP(&context.IncludeDependencies, "include-dependencies", "i", true, "whether to include dependencies (default: true)")
	flagSet.StringVarP(&context.Source, "source", "s", defaultSource(), "path to build package source directory (default: $PWD)")
	flagSet.StringVarP(&context.Version, "version", "v", "", "version to substitute into buildpack.toml")

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		log.Fatal(fmt.Errorf("unable to parse flags: %w", err))
	}

	if context.Destination == "" {
		log.Fatal("destination must be set")
	}

	b := carton.Build{Logger: bard.NewLogger(os.Stdout)}
	b.Build(context)
}

func defaultCacheLocation() string {
	s, err := os.Getwd()
	if err != nil {
		log.Fatal(fmt.Errorf("unable to get working directory: %w", err))
	}

	return filepath.Join(s, "dependencies")
}

func defaultSource() string {
	s, err := os.Getwd()
	if err != nil {
		log.Fatal(fmt.Errorf("unable to get working directory: %w", err))
	}

	return s
}
