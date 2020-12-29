package snmp

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

type MyStruct struct {
	SomeIds StringArray `yaml:"my_field"`
}

func TestStringArray_UnmarshalYAML_array(t *testing.T) {
	myStruct := MyStruct{}
	expected := MyStruct{SomeIds: StringArray{"aaa", "bbb"}}

	yaml.Unmarshal([]byte(`
my_field:
 - aaa
 - bbb
`), &myStruct)

	assert.Equal(t, expected, myStruct)
}

func TestStringArray_UnmarshalYAML_string(t *testing.T) {
	myStruct := MyStruct{}
	expected := MyStruct{SomeIds: StringArray{"aaa"}}

	yaml.Unmarshal([]byte(`
my_field: aaa
`), &myStruct)

	assert.Equal(t, expected, myStruct)
}
