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

package libpak_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnit(t *testing.T) {
	suite := spec.New("libpak", spec.Report(report.Terminal{}))
	suite("Build", testBuild)
	suite("Buildpack", testBuildpack)
	suite("BuildpackPlan", testBuildpackPlan)
	suite("Detect", testDetect)
	suite("DependencyCache", testDependencyCache)
	suite("Formatter", testFormatter)
	suite("Layer", testLayer)
	suite("Main", testMain)
	suite("Stack", testStack)
	suite.Run(t)
}
