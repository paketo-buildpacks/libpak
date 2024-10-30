/*
 * Copyright 2018-2024 the original author or authors.
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
package utils_test

import (
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"

	"github.com/paketo-buildpacks/libpak/v2/utils"
)

type Stuff struct {
	AString  string
	AnInt    int
	AFloat   float64
	AnArray  []string
	AMap     map[string]string
	APointer *string
	BPointer *string
}

func testTOMLMarshal(t *testing.T, _ spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	it("marshals and unmarshals without change", func() {
		bString := "b_string"
		someData := Stuff{
			"a_string",
			10034, 3040.23943,
			[]string{"foo", "bar", "baz"},
			map[string]string{
				"AM":  "1",
				"bar": "2",
				"baz": "3",
			},
			nil,
			&bString,
		}

		dataBytes, err := utils.Marshal(someData)
		Expect(err).ToNot(HaveOccurred())
		Expect(dataBytes).ToNot(BeEmpty())

		output := Stuff{}
		err = toml.Unmarshal(dataBytes, &output)
		Expect(err).ToNot(HaveOccurred())
		Expect(someData).To(Equal(output))
		Expect(someData.APointer).To(BeNil())
	})
}
