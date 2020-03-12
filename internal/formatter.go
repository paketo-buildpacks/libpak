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

package internal

import (
	"fmt"
	"sort"
	"strings"

	"github.com/buildpacks/libcnb"
	"github.com/heroku/color"
)

// LaunchFormatter is the formatter for a libcnb.Launch.
type LaunchFormatter libcnb.Launch

func (l LaunchFormatter) String() string {
	var s []string

	if len(l.Slices) > 0 {
		s = append(s, fmt.Sprintf("%s", SlicesFormatter(l.Slices)))
	}

	if len(l.Processes) > 0 {
		s = append(s, fmt.Sprintf("Process types:\n%s", ProcessesFormatter(l.Processes)))
	}

	return strings.Join(s, "\n")
}

// ProcessesFormatter is the formatter for a []libcnb.Process.
type ProcessesFormatter []libcnb.Process

func (p ProcessesFormatter) String() string {
	sort.Slice(p, func(i int, j int) bool {
		return p[i].Type < p[j].Type
	})

	var s []string

	max := p.maximumTypeLength()
	for _, p := range p {
		sb := strings.Builder{}
		sb.WriteString(fmt.Sprintf("  %s: ", color.CyanString(p.Type)))

		for i := 0; i < max-len(p.Type); i++ {
			sb.WriteString(" ")
		}

		sb.WriteString(p.Command)

		for _, a := range p.Arguments {
			sb.WriteString(fmt.Sprintf(" %s", a))
		}

		if p.Direct {
			sb.WriteString(" (direct)")
		}

		s = append(s, sb.String())
	}

	return strings.Join(s, "\n")
}

func (p ProcessesFormatter) maximumTypeLength() int {
	max := 0

	for _, t := range p {
		if l := len(t.Type); l > max {
			max = l
		}
	}

	return max
}

// SlicesFormatter is the formatter for a []libcnb.Slice.
type SlicesFormatter []libcnb.Slice

func (s SlicesFormatter) String() string {
	return fmt.Sprintf("%d application slices", len(s))
}

// StoreFormatter is the formatter for a libcnb.Store.
type StoreFormatter libcnb.Store

func (s StoreFormatter) String() string {
	var n []string

	for k, _ := range s.Metadata {
		n = append(n, k)
	}

	sort.Strings(n)
	return fmt.Sprintf("Writing persistent metadata: %s", strings.Join(n, ", "))
}
