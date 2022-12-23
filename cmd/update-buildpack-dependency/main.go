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
	b := carton.BuildpackDependency{}

	flagSet := pflag.NewFlagSet("Update Buildpack Dependency", pflag.ExitOnError)
	flagSet.StringVar(&b.BuildpackPath, "buildpack-toml", "", "path to buildpack.toml")
	flagSet.StringVar(&b.ID, "id", "", "the id of the dependency")
	flagSet.StringVar(&b.SHA256, "sha256", "", "the new sha256 of the dependency")
	flagSet.StringVar(&b.URI, "uri", "", "the new uri of the dependency")
	flagSet.StringVar(&b.Version, "version", "", "the new version of the dependency")
	flagSet.StringVar(&b.VersionPattern, "version-pattern", "", "the version pattern of the dependency")
	flagSet.StringVar(&b.PURL, "purl", "", "the new purl version of the dependency, if not set defaults to version")
	flagSet.StringVar(&b.PURLPattern, "purl-pattern", "", "the purl version pattern of the dependency, if not set defaults to version-pattern")
	flagSet.StringVar(&b.CPE, "cpe", "", "the new version use in all CPEs, if not set defaults to version")
	flagSet.StringVar(&b.CPEPattern, "cpe-pattern", "", "the cpe version pattern of the dependency, if not set defaults to version-pattern")
	flagSet.StringVar(&b.Source, "source", "", "the new uri of the dependency source")
	flagSet.StringVar(&b.SourceSHA256, "source-sha256", "", "the new sha256 of the dependency source")

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		log.Fatal(fmt.Errorf("unable to parse flags\n%w", err))
	}

	if b.BuildpackPath == "" {
		log.Fatal("buildpack-toml must be set")
	}

	if b.ID == "" {
		log.Fatal("id must be set")
	}

	if b.SHA256 == "" {
		log.Fatal("sha256 must be set")
	}

	if b.URI == "" {
		log.Fatal("uri must be set")
	}

	if b.Version == "" {
		log.Fatal("version must be set")
	}

	if b.VersionPattern == "" {
		log.Fatal("version-pattern must be set")
	}

	if b.PURL == "" {
		b.PURL = b.Version
	}

	if b.PURLPattern == "" {
		b.PURLPattern = b.VersionPattern
	}

	if b.CPE == "" {
		b.CPE = b.Version
	}

	if b.CPEPattern == "" {
		b.CPEPattern = b.VersionPattern
	}

	if b.Source == "" && b.SourceSHA256 == "" {
		b.Source = b.URI
		b.SourceSHA256 = b.SHA256
	}

	b.Update()
}
