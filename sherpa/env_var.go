/*
 * Copyright 2018-2025 the original author or authors.
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

package sherpa

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// AppendToEnvVar appends a collection of values to an env var separated by a delimiter. If the env var does not already
// exist, joins the values with the delimiter and returns the result.
func AppendToEnvVar(name string, delimiter string, values ...string) string {
	var e []string
	if s, ok := os.LookupEnv(name); ok {
		e = append(e, s)
	}
	e = append(e, values...)
	return strings.Join(e, delimiter)
}

// GetEnvRequired returns the value of an environment variable if it exists, otherwise returns an error with a
// predictable message.
func GetEnvRequired(name string) (string, error) {
	if s, ok := os.LookupEnv(name); ok {
		return s, nil
	}

	return "", fmt.Errorf("$%s must be set", name)
}

// GetEnvWithWithDefault returns the value of an environment variable if it exists, otherwise returns the default.
func GetEnvWithDefault(name string, def string) string {
	if s, ok := os.LookupEnv(name); ok {
		return s
	}
	return def
}

// ResolveBool resolves a boolean value for a configuration option. Returns true for 1, t, T, TRUE, true, True. Returns
// false for all other values or unset.
func ResolveBool(name string) bool {
	resolveBool, _ := ResolveBoolErr(name)
	return resolveBool
}

// ResolveBoolErr resolves a boolean value for a configuration option.
// Returns true, nil for 1, t, T, TRUE, true, True.
// Returns false, nil for all other values or unset.
// Returns false, error is the value could not be parsed into a bool
func ResolveBoolErr(name string) (bool, error) {
	s, ok := os.LookupEnv(name)
	if !ok {
		return false, nil
	}

	p, err := strconv.ParseBool(strings.TrimSpace(s))
	if err != nil {
		return false, fmt.Errorf(
			"invalid value '%s' for key '%s': expected one of [1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False]",
			s,
			name,
		)
	}

	return p, nil
}

// ResolveBoolWithDefault resolves a boolean value for a configuration option or returns the default value
func ResolveBoolWithDefault(name string, defaultVal bool) bool {
	s, ok := os.LookupEnv(name)
	if !ok {
		return defaultVal
	}

	p, err := strconv.ParseBool(strings.TrimSpace(s))
	if err != nil {
		return false
	}

	return p
}
