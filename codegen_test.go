package codegen_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/lestrrat-go/codegen"
	"github.com/stretchr/testify/assert"
)

func TestCodegen(t *testing.T) {
	t.Run("FormatCode", func(t *testing.T) {
		var dst, src bytes.Buffer

		fmt.Fprintf(&src, "package main")
		fmt.Fprintf(&src, "\n\nfunc main(){")
		fmt.Fprintf(&src, "\n}")

		if !assert.NoError(t, codegen.Write(&dst, &src, codegen.WithFormatCode(true)), `codegen.Write should succeed`) {
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
