package codegen

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"golang.org/x/tools/imports"
)

func Write(dst io.Writer, src io.Reader, options ...Option) error {
	var formatCode bool
	var lineNumber bool
	for _, option := range options {
		switch option.Name() {
		case optKeyFormatCode:
			formatCode = option.Value().(bool)
		case optKeyLineNumber:
			lineNumber = option.Value().(bool)
		}
	}

	if formatCode {
		buf, err := ioutil.ReadAll(src)
		if err != nil {
			return errors.Wrap(err, `failed to read from source`)
		}

		formatted, err := imports.Process("", buf, nil)
		if err != nil {
			return codeFormatError(err, buf)
		}

		src = bytes.NewReader(formatted)
	}

	if lineNumber {
		// Count the number of lines, so we know how many digits to use
		buf, err := ioutil.ReadAll(src)
		if err != nil {
			return errors.Wrap(err, `failed to read from source`)
		}

		digits := int(math.Log10(float64(bytes.Count(buf, []byte{'\n'})))) + 1
		dstFmt := fmt.Sprintf("%%0%dd %%s\n", digits)
		var dst bytes.Buffer
		for len(buf) > 0 {
			l := bytes.Index(buf, []byte{'\n'})
			if l == -1 {
				l = len(buf)
			}
			fmt.Fprintf(&dst, dstFmt, buf[:l])
			if l == len(buf) {
				buf = nil
			} else {
				buf = buf[l+1:]
			}
		}

		src = &dst
	}

	_, err := io.Copy(dst, src)
	return err
}

func WriteFile(filename string, src io.Reader, options ...Option) error {
	if dir := filepath.Dir(filename); dir != "." {
		if _, err := os.Stat(dir); err != nil {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return errors.Wrapf(err, `failed to create directory %s`, dir)
			}
		}
	}

	dst, err := os.Create(filename)
	if err != nil {
		return errors.Wrapf(err, `failed to open %s.go`, filename)
	}
	defer dst.Close()

	return Write(dst, src, options...)
}
