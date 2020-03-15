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
	"reflect"
	"strings"

	"github.com/buildpacks/libcnb"
	"github.com/imdario/mergo"
)

// PlanEntryResolver provides functionality for resolving a Buildpack Plan Entry given a name.
type PlanEntryResolver struct {

	// Plan is the BuildpackPlan to resolve against.
	Plan libcnb.BuildpackPlan
}

// MergeFunc takes two BuildpackPlanEntry's and returns a merged entry.
type MergeFunc func(a, b libcnb.BuildpackPlanEntry) (libcnb.BuildpackPlanEntry, error)

// NoValidEntryError is returned when the resolver cannot find any valid entry.
type NoValidEntryError struct {
	// Message is the error message
	Message string
}

func (n NoValidEntryError) Error() string {
	return n.Message
}

// IsNoValidEntry indicates whether an error is a NoValidEntryError.
func IsNoValidEntry(err error) bool {
	_, ok := err.(NoValidEntryError)
	return ok
}

// ResolveWithMerge returns a single BuildpackPlanEntry that is a merged version of all entries that have a given name.
// A merge function is used to describe how two entries are merged together.
func (p *PlanEntryResolver) ResolveWithMerge(name string, f MergeFunc) (libcnb.BuildpackPlanEntry, error) {
	m := libcnb.BuildpackPlanEntry{}

	var err error
	for _, e := range p.Plan.Entries {
		if e.Name == name {
			if m, err = f(m, e); err != nil {
				return libcnb.BuildpackPlanEntry{}, fmt.Errorf("error merging %+v and %+v\n%w", m, e, err)
			}
		}
	}

	if reflect.DeepEqual(m, libcnb.BuildpackPlanEntry{}) {
		return libcnb.BuildpackPlanEntry{}, NoValidEntryError{
			Message: fmt.Sprintf("no valid entries for %s in %s", name, p.Plan.Entries),
		}
	}

	return m, nil
}

// ShallowMerge merges two BuildpackPlanEntry's together.  Declared versions are combined with a comma delimiter and
// metadata is combined with the values for b taking priority over the values of a when the keys are duplicated.
func ShallowMerge(a, b libcnb.BuildpackPlanEntry) (libcnb.BuildpackPlanEntry, error) {
	var v []string
	if a.Version != "" {
		v = append(v, a.Version)
	}
	if b.Version != "" {
		v = append(v, b.Version)
	}

	if err := mergo.Merge(&b, a); err != nil {
		return libcnb.BuildpackPlanEntry{}, fmt.Errorf("unable to merge %+v and %+v\n%w", a, b, err)
	}

	b.Version = strings.Join(v, ",")
	return b, nil
}

// Resolve calls ResolveWithMerge function passing in the ShallowMerge function as the merge strategy.
func (p *PlanEntryResolver) Resolve(name string) (libcnb.BuildpackPlanEntry, error) {
	return p.ResolveWithMerge(name, ShallowMerge)
}

// Any indicates whether the collection of entries has any entry with the given name.
func (p *PlanEntryResolver) Any(name string) bool {
	_, err := p.Resolve(name)
	return err == nil
}
