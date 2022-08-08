package codegen

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/imports"
)

// R is a short hand for fmt.Fprintf(...)
func R(dst io.Writer, s string, args ...interface{}) {
	fmt.Fprintf(dst, s, args...)
}

// L writes a line, PREFIXED with a single new line ('\n') character
func L(dst io.Writer, s string, args ...interface{}) {
	R(dst, "\n"+s, args...)
}

// LL writes a line, PREFIXED with two new line ('\n') characters
func LL(dst io.Writer, s string, args ...interface{}) {
	R(dst, "\n\n"+s, args...)
}

type Output struct {
	src io.Reader
	dst io.Writer
}

func NewOutput(dst io.Writer) *Output {
	o := &Output{}
	if v, ok := dst.(io.Writer); ok {
		o.dst = v
	}

	if v, ok := dst.(io.Reader); ok {
		o.src = v
	}
	return o
}

func (o *Output) R(s string, args ...interface{}) {
	R(o.dst, s, args...)
}

func (o *Output) L(s string, args ...interface{}) {
	L(o.dst, s, args...)
}

func (o *Output) LL(s string, args ...interface{}) {
	LL(o.dst, s, args...)
}

// Comment outputs multi-line comments, prefixed with a '//` marker
// on each line.
// The first line is prefixed with an extra new line
func (o *Output) Comment(s string) {
	scanner := bufio.NewScanner(strings.NewReader(s))
	i := 0
	for scanner.Scan() {
		l := scanner.Text()

		if i == 0 {
			o.LL(`// %s`, l)
		} else {
			o.L(`// %s`, l)
		}
		i++
	}
}

func (o *Output) WritePackage(s string, args ...interface{}) {
	L(o.dst, "package ")
	R(o.dst, s, args...)
}

type ImportPkg struct {
	Alias string
	URL   string
}

func (o *Output) WriteImports(urls ...string) error {
	var pkgs []ImportPkg
	for _, url := range urls {
		pkgs = append(pkgs, ImportPkg{
			URL: url,
		})
	}
	return o.WriteImportPkgs(pkgs...)
}

func (o *Output) WriteImportPkgs(pkgs ...ImportPkg) error {
	return WriteImports(o.dst, pkgs...)
}

func (o *Output) Write(dst io.Writer, options ...Option) error {
	return Write(dst, o.src, options...)
}

func (o *Output) WriteFile(fn string, options ...Option) error {
	return WriteFile(fn, o.src, options...)
}

func WriteImports(dst io.Writer, pkgs ...ImportPkg) error {
	L(dst, "import (")
	for _, pkg := range pkgs {
		if pkg.Alias != "" {
			L(dst, "%s %q", pkg.Alias, pkg.URL)
		} else {
			L(dst, "%q", pkg.URL)
		}
	}
	L(dst, ")")
	return nil
}

func Write(dst io.Writer, src io.Reader, options ...Option) error {
	var formatCode bool
	var lineNumber bool
	for _, option := range options {
		switch option.Ident() {
		case identFormatCode{}:
			formatCode = option.Value().(bool)
		case identLineNumber{}:
			lineNumber = option.Value().(bool)
		}
	}

	if formatCode {
		buf, err := ioutil.ReadAll(src)
		if err != nil {
			return fmt.Errorf(`failed to read from source: %w`, err)
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
			return fmt.Errorf(`failed to read from source: %w`, err)
		}

		digits := int(math.Log10(float64(bytes.Count(buf, []byte{'\n'})))) + 1
		dstFmt := fmt.Sprintf("%%0%dd %%s\n", digits)
		var dst bytes.Buffer
		lineno := 1
		for len(buf) > 0 {
			l := bytes.Index(buf, []byte{'\n'})
			if l == -1 {
				l = len(buf)
			}
			fmt.Fprintf(&dst, dstFmt, lineno, buf[:l])
			if l == len(buf) {
				buf = nil
			} else {
				buf = buf[l+1:]
			}
			lineno++
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
				return fmt.Errorf(`failed to create directory %q: %w`, dir, err)
			}
		}
	}

	dst, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf(`failed to open %s.go: %w`, filename, err)
	}
	defer dst.Close()

	return Write(dst, src, options...)
}
