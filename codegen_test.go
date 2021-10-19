package codegen_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/lestrrat-go/codegen"
	"github.com/stretchr/testify/assert"
)

func TestCodegen(t *testing.T) {
	t.Run("FormatCode", func(t *testing.T) {
		var dst, src bytes.Buffer

		o := codegen.NewOutput(&src)
		o.R("package main")
		o.LL("func main(){")
		o.L("}")

		if !assert.NoError(t, o.Write(&dst, codegen.WithFormatCode(true)), `codegen.Write should succeed`) {
			return
		}

		const expected = `package main

func main() {
}
`

		if !assert.Equal(t, expected, dst.String(), `output should match`) {
			return
		}
	})
	t.Run("InvalidCode", func(t *testing.T) {
		var dst, src bytes.Buffer

		fmt.Fprintf(&src, "package main func main(){")
		fmt.Fprintf(&src, "\n}")

		codegenErr := codegen.Write(&dst, &src, codegen.WithFormatCode(true))
		if !assert.Error(t, codegenErr, `codegen.Write should fail`) {
			return
		}

		cfe, ok := codegenErr.(codegen.CodeFormatError)
		if !assert.True(t, ok, `codegenErr should be codegen.CodeFormatError (got %T)`, codegenErr) {
			return
		}

		const expected = `1: package main func main(){
2: }
`
		if !assert.Equal(t, expected, cfe.Source(), `cfe.Source should match`) {
			return
		}
	})
}

func TestObject(t *testing.T) {
	var _ codegen.Field = &codegen.ConstantField{}

	testcases := []struct {
		Name   string
		Input  string
		Verify func(*testing.T, *codegen.ConstantField)
	}{
		{
			Name:  `Simple`,
			Input: `{"name": "Field1"}`,
			Verify: func(t *testing.T, f *codegen.ConstantField) {
				if !assert.Equal(t, "Field1", f.Name(true), `name should match`) {
					return
				}

				if !assert.Equal(t, "field1", f.Name(false), `unexported name should match`) {
					return
				}

				if !assert.Equal(t, "string", f.Type(), `type should match`) {
					return
				}
			},
		},
		{
			Name:  `separate name, exported name, and unexported name`,
			Input: `{"name": "Field1", "exported_name": "Foo", "unexported_name": "bar"}`,
			Verify: func(t *testing.T, f *codegen.ConstantField) {
				if !assert.Equal(t, "Foo", f.Name(true), `name should match`) {
					return
				}

				if !assert.Equal(t, "bar", f.Name(false), `unexported name should match`) {
					return
				}

				if !assert.Equal(t, "string", f.Type(), `type should match`) {
					return
				}
			},
		},
	}
	for _, tc := range testcases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			var field codegen.ConstantField
			if !assert.NoError(t, json.Unmarshal([]byte(tc.Input), &field), `json.Unmarshal should succeed`) {
				return
			}
			field.Organize()
			tc.Verify(t, &field)
		})
	}
}
