package internal

import (
	"bytes"

	"github.com/BurntSushi/toml"
)

func Marshal(v interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := toml.NewEncoder(buf).Encode(v)
	return buf.Bytes(), err
}
