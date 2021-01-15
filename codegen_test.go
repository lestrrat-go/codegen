package codegen_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/lestrrat-go/codegen"
	"github.com/stretchr/testify/assert"
)

func TestCodegen(t *testing.T) {

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
}
