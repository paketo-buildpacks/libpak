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

// Resolve returns the matching binding within the collection of Bindings.  The candidate set is filtered by the
// constraints.
func (b *BindingResolver) Resolve(bindingType string) (libcnb.Binding, bool, error) {
	m := make([]libcnb.Binding, 0)
	for _, binding := range b.Bindings {
		if strings.ToLower(binding.Type) == strings.ToLower(bindingType) {
			m = append(m, binding)
		}
	}

	if len(m) < 1 {
		return libcnb.Binding{}, false, nil
	} else if len(m) > 1 {
		return libcnb.Binding{}, false, fmt.Errorf("multiple bindings found for type %s in %+v", bindingType, b.Bindings)
	}

	return m[0], true, nil
}
