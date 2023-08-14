package internal_test

import (
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/libpak/v2/internal"
	"github.com/sclevine/spec"
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

func testTOMLMarshal(t *testing.T, context spec.G, it spec.S) {
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

		dataBytes, err := internal.Marshal(someData)
		Expect(err).ToNot(HaveOccurred())
		Expect(dataBytes).ToNot(BeEmpty())

		output := Stuff{}
		err = toml.Unmarshal(dataBytes, &output)
		Expect(err).ToNot(HaveOccurred())
		Expect(someData).To(Equal(output))
		Expect(someData.APointer).To(BeNil())
	})
}
