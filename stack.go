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

package libpak

const (
	// JammyStackID is the ID for the Cloud Native Buildpacks jammy stack.
	JammyStackID = "io.buildpacks.stacks.jammy"

	// JammyTinyStackID is the ID for the Cloud Native Buildpacks jammy tiny stack.
	JammyTinyStackID = "io.buildpacks.stacks.jammy.tiny"

	// JammyStaticStackID is the ID for the Cloud Native Buildpacks jammy static stack.
	JammyStaticStackID = "io.buildpacks.stacks.jammy.static"

	// NobleStackID is the ID for the Cloud Native Buildpacks noble stack.
	NobleStackID = "io.buildpacks.stacks.noble"

	// NobleTinyStackID is the ID for the Cloud Native Buildpacks noble tiny stack.
	NobleTinyStackID = "io.buildpacks.stacks.noble.tiny"

	// NobleStaticStackID is the ID for the Cloud Native Buildpacks noble static stack.
	NobleStaticStackID = "io.buildpacks.stacks.noble.static"
)

// IsJammyStack returns true if the stack is one of the jammy variants
func IsJammyStack(stack string) bool {
	return JammyStackID == stack || JammyTinyStackID == stack || JammyStaticStackID == stack
}

// IsNobleStack returns true if the stack is one of the noble variants
func IsNobleStack(stack string) bool {
	return NobleStackID == stack || NobleTinyStackID == stack || NobleStaticStackID == stack
}

// IsTinyStack returns true if the stack is one of the tiny variants
func IsTinyStack(stack string) bool {
	return JammyTinyStackID == stack || NobleTinyStackID == stack
}

// IsStaticStack returns true if the stack is one of the static variants
func IsStaticStack(stack string) bool {
	return JammyStaticStackID == stack || NobleStaticStackID == stack
}

// IsShellPresentOnStack returns true if the stack is known to have a shell
func IsShellPresentOnStack(stack string) bool {
	return JammyStackID == stack || NobleStackID == stack
}
