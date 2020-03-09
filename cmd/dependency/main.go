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

	"github.com/packeto-buildpacks/libpak/carton"
	"github.com/spf13/pflag"
)

func main() {
	d := carton.Dependency{}

	flagSet := pflag.NewFlagSet("Update Dependency", pflag.ExitOnError)
	flagSet.StringVar(&d.BuildpackPath, "buildpack-toml", "", "path to buildpack.toml")
	flagSet.StringVar(&d.ID, "id", "", "the id of the dependency")
	flagSet.StringVar(&d.SHA256, "sha256", "", "the new sha256 of the dependency")
	flagSet.StringVar(&d.URI, "uri", "", "the new uri of the dependency")
	flagSet.StringVar(&d.Version, "version", "", "the new version of the dependency")
	flagSet.StringVar(&d.VersionPattern, "version-pattern", "", "the version pattern of the dependency")

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		log.Fatal(fmt.Errorf("unable to parse flags: %w", err))
	}

	if d.BuildpackPath == "" {
		log.Fatal("buildpack-toml must be set")
	}

	if d.ID == "" {
		log.Fatal("id must be set")
	}

	if d.SHA256 == "" {
		log.Fatal("sha256 must be set")
	}

	if d.URI == "" {
		log.Fatal("uri must be set")
	}

	if d.Version == "" {
		log.Fatal("version must be set")
	}

	if d.VersionPattern == "" {
		log.Fatal("version-pattern must be set")
	}

	d.Build()
}
