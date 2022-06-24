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

const (
	// BionicStackID is the ID for the Cloud Native Buildpacks bionic stack.
	BionicStackID = "io.buildpacks.stacks.bionic"

	// BionicTinyStackID is the ID for the Paketo Buildpacks bionic tiny stack.
	BionicTinyStackID = "io.paketo.stacks.tiny"

	// TinyStackID is the ID for the Paketo Buildpacks bionic tiny stack.
	//
	// Deprecated: use BionicTinyStackID instead
	TinyStackID = "io.paketo.stacks.tiny"

	// JammyStackID is the ID for the Cloud Native Buildpacks jammy stack.
	JammyStackID = "io.buildpacks.stacks.jammy"

	// JammyTinyStackID is the ID for the Cloud Native Buildpacks jammy tiny stack.
	JammyTinyStackID = "io.buildpacks.stacks.jammy.tiny"
)
