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

// BindingConstraint is the collection of constraints to use during resolution.
type BindingConstraint struct {

	// Name is the name of the binding.
	Name string

	// Kind is the kind of the binding.
	Kind string

	// Provider is the provider of the binding.
	Provider string

	// Tags are the tags of the binding.  The contents of Tags must be a strict subset of the binding's tags.
	Tags []string
}

// Resolve returns the matching binding within the collection of Bindings.  The candidate set is filtered by the
// constraints.
func (b *BindingResolver) Resolve(constraint BindingConstraint) (libcnb.Binding, bool, error) {
	var m []libcnb.Binding
	for name, binding := range b.Bindings {
		if b.matches(name, binding, constraint) {
			m = append(m, binding)
		}
	}

	if len(m) < 1 {
		return libcnb.Binding{}, false, nil
	} else if len(m) > 1 {
		return libcnb.Binding{}, false, fmt.Errorf("multiple bindings found for %+v in %+v", constraint, b.Bindings)
	}

	return m[0], true, nil
}

func (BindingResolver) contains(candidates []string, value string) bool {
	for _, c := range candidates {
		if c == value {
			return true
		}
	}

	return false
}

func (b BindingResolver) matches(name string, binding libcnb.Binding, constraint BindingConstraint) bool {
	if constraint.Name != "" && constraint.Name != name {
		return false
	}

	if constraint.Kind != "" && constraint.Kind != binding.Metadata[libcnb.BindingKind] {
		return false
	}

	if constraint.Provider != "" && constraint.Provider != binding.Metadata[libcnb.BindingProvider] {
		return false
	}

	tags := strings.Split(binding.Metadata[libcnb.BindingTags], "\n")
	for _, t := range constraint.Tags {
		if !b.contains(tags, t) {
			return false
		}
	}

	return true
}
