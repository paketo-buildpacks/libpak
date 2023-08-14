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

	"github.com/paketo-buildpacks/libpak/v2/carton"
)

func main() {
	i := carton.BuildImageDependency{}

	flagSet := pflag.NewFlagSet("Update Image Dependency", pflag.ExitOnError)
	flagSet.StringVar(&i.BuilderPath, "builder-toml", "", "path to builder.toml")
	flagSet.StringVar(&i.Version, "version", "", "the new version of the dependency")

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		log.Fatal(fmt.Errorf("unable to parse flags\n%w", err))
	}

	if i.BuilderPath == "" {
		log.Fatal("builder-toml must be set")
	}

	if i.Version == "" {
		log.Fatal("version must be set")
	}

	i.Update()
}
