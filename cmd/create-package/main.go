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

	"github.com/spf13/pflag"

	"github.com/paketo-buildpacks/libpak/carton"
)

func main() {
	p := carton.Package{}

	flagSet := pflag.NewFlagSet("Create Package", pflag.ExitOnError)
	flagSet.StringVar(&p.CacheLocation, "cache-location", "", "path to cache downloaded dependencies (default: $PWD/dependencies)")
	flagSet.StringVar(&p.Destination, "destination", "", "path to the build package destination directory")
	flagSet.BoolVar(&p.IncludeDependencies, "include-dependencies", false, "whether to include dependencies (default: false)")
	flagSet.StringSliceVar(&p.DependencyFilters, "dependency-filter", []string{}, "one or more filters that are applied to exclude dependencies")
	flagSet.BoolVar(&p.StrictDependencyFilters, "strict-filters", false, "require filter to match all data or just some data (default: false)")
	flagSet.StringVar(&p.Source, "source", defaultSource(), "path to build package source directory (default: $PWD)")
	flagSet.StringVar(&p.Version, "version", "", "version to substitute into buildpack.toml")
	flagSet.StringVar(&p.TargetArch, "target-arch", carton.DefaultTargetArch, "target architecture for the package (default: all)")

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		log.Fatal(fmt.Errorf("unable to parse flags\n%w", err))
	}

	if p.Destination == "" {
		log.Fatal("destination must be set")
	}

	p.Create()
}

func defaultSource() string {
	s, err := os.Getwd()
	if err != nil {
		log.Fatal(fmt.Errorf("unable to get working directory\n%w", err))
	}

	return s
}
