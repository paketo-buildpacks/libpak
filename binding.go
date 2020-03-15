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

package libpak

import (
	"fmt"
	"strings"

	"github.com/buildpacks/libcnb"
)

// BindingResolver provides functionality for resolving a binding given a collection of constraints.
type BindingResolver struct {

	// Bindings are the bindings to resolve against.
	Bindings libcnb.Bindings
}

// NoValidBindingError is returned when the resolver cannot find any valid binding given the constraints.
type NoValidBindingError struct {
	// Message is the error message
	Message string
}

func (n NoValidBindingError) Error() string {
	return n.Message
}

// IsNoValidBinding indicates whether an error is a NoValidBindingError.
func IsNoValidBinding(err error) bool {
	_, ok := err.(NoValidBindingError)
	return ok
}

// Resolve returns the matching binding within the collection of Bindings.  The candidate set is filtered by the
// constraints.
func (b *BindingResolver) Resolve(kind string, provider string, tags ...string) (libcnb.Binding, error) {
	m := make([]libcnb.Binding, 0)
	for _, binding := range b.Bindings {
		if b.matches(binding, kind, provider, tags) {
			m = append(m, binding)
		}
	}

	if len(m) != 1 {
		return libcnb.Binding{}, NoValidBindingError{
			Message: fmt.Sprintf("no valid binding for %s, %s, and %s in %s",
				kind, provider, tags, b.Bindings),
		}
	}

	return m[0], nil
}

// Any indicates whether the collection of Bindings has any binding that satisfies the constraints.
func (b *BindingResolver) Any(kind string, provider string, tags ...string) bool {
	_, err := b.Resolve(kind, provider, tags...)
	return err == nil
}

func (BindingResolver) contains(candidates []string, value string) bool {
	for _, c := range candidates {
		if c == value {
			return true
		}
	}

	return false
}

func (b BindingResolver) matches(binding libcnb.Binding, kind string, provider string, tags []string) bool {
	if kind != "" && kind != binding.Metadata[libcnb.BindingKind] {
		return false
	}

	if provider != "" && provider != binding.Metadata[libcnb.BindingProvider] {
		return false
	}

	candidates := strings.Split(binding.Metadata[libcnb.BindingTags], "\n")
	for _, t := range tags {
		if !b.contains(candidates, t) {
			return false
		}
	}

	return true
}
