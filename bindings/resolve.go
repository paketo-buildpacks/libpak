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

package bindings

import (
	"fmt"
	"strings"

	"github.com/buildpacks/libcnb/v2"
)

// Predicate should return true if it matches a given binding.
type Predicate func(bind libcnb.Binding) bool

// OfType returns a Predicate that returns true if a given binding has Type that matches t. The comparison is
// case-insensitive.
func OfType(t string) Predicate {
	return func(bind libcnb.Binding) bool {
		return strings.EqualFold(bind.Type, t)
	}
}

// OfKeyWithValueFromSecret returns a Predicate that returns true if a given binding has Key that matches k with Value that matches v.
// The comparison is case-insensitive.
func OfKeyWithValueFromSecret(k string, v string) Predicate {
	return func(bind libcnb.Binding) bool {
		value, exists := bind.Secret[k]
		if !exists {
			return false
		}

		return strings.EqualFold(value, v)
	}
}

// OfProvider returns a Predicate that returns true if a given binding has Provider that matches p. The comparison is
// case-insensitive.
func OfProvider(p string) Predicate {
	return func(bind libcnb.Binding) bool {
		return strings.EqualFold(bind.Provider, p)
	}
}

// WithName returns a Predicate that returns true if a given binding has Name that matches n. The comparison is
// case-insensitive.
func WithName(n string) Predicate {
	return func(bind libcnb.Binding) bool {
		return strings.EqualFold(bind.Name, n)
	}
}

// Resolve returns all bindings from binds that match every Predicate in predicates.
func Resolve(binds libcnb.Bindings, predicates ...Predicate) libcnb.Bindings {
	var result libcnb.Bindings
	// deep copy
	result = append(result, binds...)

	// filter on predicates
	for _, p := range predicates {
		result = filter(result, p)
	}
	return result
}

// ResolveOne returns a single binding from bindings that match every Predicate if present. If exactly one match is found
// ResolveOne returns the binding and true. If zero matches are found, ResolveOne returns an empty binding and false.
// An error is returned if multiple matches are found.
func ResolveOne(binds libcnb.Bindings, predicates ...Predicate) (libcnb.Binding, bool, error) {
	resolved := Resolve(binds, predicates...)
	if len(resolved) == 0 {
		return libcnb.Binding{}, false, nil
	}
	if len(resolved) > 1 {
		return libcnb.Binding{}, false, errTooManyBindings(resolved)
	}
	return resolved[0], true, nil
}

func errTooManyBindings(binds libcnb.Bindings) error {
	var names []string
	for _, bind := range binds {
		names = append(names, bind.Name)
	}
	return fmt.Errorf("multiple bindings matched the given predicates %+v", names)
}

func filter(binds libcnb.Bindings, p Predicate) libcnb.Bindings {
	var result []libcnb.Binding
	for _, bind := range binds {
		if p(bind) {
			result = append(result, bind)
		}
	}
	return result
}
