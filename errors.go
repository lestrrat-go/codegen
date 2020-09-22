package codegen

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
)

type CodeFormatError struct {
	src []byte
	err error
}

func codeFormatError(err error, src []byte) error {
	if err == nil {
		panic("invalid code format error: nil error passed")
	}

	if src == nil {
		panic("invalid code format error: nil src passed")
	}

	return CodeFormatError {
		err: err,
		src: src,
	}
}

func (err CodeFormatError) Error() string {
	return err.err.Error()
}

// Returns the source code with line numbers
func (err CodeFormatError) Source() string {
	n := bytes.Count(err.src, []byte{'\n'})
	if n == 0 {
		if len(err.src) > 0 {
			n = 1
		}
	}

	lineDigits := int(math.Log10(float64(n))) + 1
	prefix := fmt.Sprintf("%%0%dd", lineDigits)

	var dst bytes.Buffer

	scanner := bufio.NewScanner(bytes.NewReader(err.src))
	lineno := 1
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintf(&dst, "%s: %s\n", prefix, line)
		lineno++
	}

	return dst.String()
}
