package codegen

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"golang.org/x/tools/imports"
)

func Write(dst io.Writer, src io.Reader, options ...Option) error {
	var formatCode bool

	for _, option := range options {
		switch option.Name() {
		case optKeyFormatCode:
			formatCode = option.Value().(bool)
		}
	}

	if formatCode {
		buf, err := ioutil.ReadAll(src)
		if err != nil {
			return errors.Wrap(err, `failed to read from source`)
		}

		formatted, err := imports.Process("", buf, nil)
		if err != nil {
			return CodeFormatError{
				src: buf,
			}
		}

		src = bytes.NewReader(formatted)
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
