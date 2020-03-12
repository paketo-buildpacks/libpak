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

package bard

import (
	"fmt"

	"github.com/heroku/color"
)

// FormatIdentity formats a name and an optional description in the form '<b>name</b>[ description]'.
func FormatIdentity(name string, description string) string {
	s := color.New(color.Bold).Sprint(name)

	if description != "" {
		s += fmt.Sprintf(" %s", description)
	}

	return s
}

// FormatLaunchConfig formats a name and default value, and an optional reason in the form
// 'Set $name to configure[ reason]. Default <i>def</i>.
func FormatLaunchConfig(name string, reason string, def string) string {
	s := fmt.Sprintf("Set $%s to configure", name)

	if reason != "" {
		s += fmt.Sprintf(" %s", reason)
	}

	s += fmt.Sprintf(". Default %s.", color.New(color.Italic).Sprint(def))

	return s
}
