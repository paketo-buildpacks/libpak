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
	"strings"

	"github.com/heroku/color"
)

// Identity is the formatter for an identifiable object.
type IdentityFormatter struct {

	// Name is the name of the identified object.
	Name string

	// Description is the description of the identified object.
	Description string
}

func (i IdentityFormatter) String() string {
	var sb strings.Builder

	_, _ = sb.WriteString(color.New(color.Bold).Sprint(i.Name))

	if i.Description != "" {
		_, _ = sb.WriteString(fmt.Sprintf(" %s", i.Description))
	}

	return sb.String()
}

var launchConfigDefault = color.New(color.Italic).SprintFunc()

// LaunchConfigFormatter is the logging formatter for a user configuration and its default value.
type LaunchConfigFormatter struct {

	// Name is the name of the launch configuration environment variable.
	Name string

	// Default is the default value of the the configuration.
	Default string
}

func (l LaunchConfigFormatter) String() string {
	return fmt.Sprintf("Set $%s to configure. Default %s.", l.Name, launchConfigDefault(l.Default))
}
