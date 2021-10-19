package codegen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/lestrrat-go/xstrings"
)

var zerovals = map[string]string{
	"string": `""`,
}

func RegisterZeroVal(typ, val string) {
	zerovals[typ] = val
}

func ZeroVal(typ string) string {
	if v, ok := zerovals[typ]; ok {
		return v
	}
	return `nil`
}

type base struct {
	name           string
	exportedName   string
	unexportedName string
}

func (b *base) handleJSONField(dec *json.Decoder, field string) (bool, error) {
	var fref interface{}
	switch field {
	case "name":
		fref = &b.name
	case "unexported_name":
		fref = &b.unexportedName
	case "exported_name":
		fref = &b.exportedName
	default:
		return false, nil
	}
	if err := dec.Decode(fref); err != nil {
		return true, fmt.Errorf(`failed to decode field %q: %w`, field, err)
	}
	return true, nil
}

func (b *base) Name(exported bool) string {
	if exported {
		if v := b.exportedName; v != "" {
			return v
		}

		if strings.ToUpper(b.name) == b.name {
			b.exportedName = b.name
		} else {
			b.exportedName = xstrings.Camel(b.name)
		}
		return b.exportedName
	}

	if v := b.unexportedName; v != "" {
		return v
	}

	v := xstrings.Camel(b.name)
	if strings.ToUpper(v) == v {
		b.unexportedName = strings.ToLower(v)
	} else {
		b.unexportedName = xstrings.LcFirst(v)
	}
	return b.unexportedName
}

type Object struct {
	base
	fields   []Field
	arrayOf  string
	objectOf string
}

func (o *Object) Organize() {
	for _, field := range o.fields {
		field.Organize()
	}
	sort.Slice(o.fields, func(i, j int) bool {
		return o.fields[i].Name(true) < o.fields[j].Name(true)
	})
}

func (o *Object) ArrayOf() string {
	return o.arrayOf
}

func (o *Object) ObjectOf() string {
	return o.objectOf
}

func (o *Object) Fields() []Field {
	return o.fields
}

func (o *Object) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf(`failed to read next token: %w`, err)
	}

	switch tok := tok.(type) {
	case json.Delim:
		if tok != '{' {
			return fmt.Errorf(`expected '{', got %#v`, tok)
		}
	default:
		return fmt.Errorf(`expected '{', got %#v`, tok)
	}

OUTER:
	for {
		log.Printf("OUTER")
		tok, err := dec.Token()
		if err != nil {
			return fmt.Errorf(`failed to read next token: %w`, err)
		}

		switch tok := tok.(type) {
		case json.Delim:
			if tok == '}' {
				break OUTER
			}
			return fmt.Errorf(`unexpected delimiter %#v`, tok)
		case string:
			handled, err := o.handleJSONField(dec, tok)
			if err != nil {
				return err
			}

			if handled {
				continue OUTER
			}

			var fref interface{}
			switch tok {
			case "object_of":
				fref = &o.objectOf
			case "array_of":
				fref = &o.arrayOf
			case "fields":
				var list []json.RawMessage
				if err := dec.Decode(&list); err != nil {
					return fmt.Errorf(`failed to decode "fields" to list of messages`)
				}

				o.fields = make([]Field, 0, len(list))
				// if the message contains `constant`, it's a constant
				for i, raw := range list {
					var probe struct {
						Constant json.RawMessage `json:"constant"`
					}
					if err := json.Unmarshal(raw, &probe); err != nil {
						return fmt.Errorf(`failed to decode probe for field %d: %w`, i+1, err)
					}

					var f Field
					if len(probe.Constant) > 0 {
						var c ConstantField
						if err := json.Unmarshal(raw, &c); err != nil {
							return fmt.Errorf(`failed to decode constant field %d: %w`, i+1, err)
						}
						f = &c
					} else {
						var s stdField
						if err := json.Unmarshal(raw, &s); err != nil {
							return fmt.Errorf(`failed to decode field %d: %w`, i+1, err)
						}
						f = &s
					}

					o.fields = append(o.fields, f)
				}
				log.Printf("parsed %d fields", len(o.fields))
				continue OUTER
			default:
				return fmt.Errorf(`invalid object key %#v`, tok)
			}

			if err := dec.Decode(fref); err != nil {
				return fmt.Errorf(`failed to decode field %q: %w`, tok, err)
			}
		}
	}
	return nil
}

type Field interface {
	Organize()

	SkipMethod() bool

	// Name of the field. Can be exported or not exported
	Name(bool) string
	// The Go type
	Type() string

	// The JSON key used
	JSON() string

	GetterMethod(bool) string
}

type stdField struct {
	base
	skipMethod   bool
	typ          string
	jsonName     string
	getterMethod string
	required     bool
}

func (f *stdField) Organize() {
	if f.typ == "" {
		f.typ = "string"
	}
}

func (f *stdField) SkipMethod() bool {
	return f.skipMethod
}

func (f *stdField) Type() string {
	return f.typ
}

func (f *stdField) JSON() string {
	if v := f.jsonName; v != "" {
		return v
	}
	return f.Name(false)
}

func (f *stdField) GetterMethod(exported bool) string {
	if v := f.getterMethod; v != "" {
		return v
	}

	return xstrings.Camel(f.name)
}

func (f *stdField) handleJSONField(dec *json.Decoder, field string) (bool, error) {
	ok, err := f.base.handleJSONField(dec, field)
	if err != nil {
		return false, err
	}

	if ok {
		return ok, nil
	}

	var fref interface{}
	switch field {
	case "type":
		fref = &f.typ
	case "json":
		fref = &f.jsonName
	case "getter":
		fref = &f.getterMethod
	case "skip_method":
		fref = &f.skipMethod
	case "required":
		fref = &f.required
	default:
		return false, nil
	}
	if err := dec.Decode(fref); err != nil {
		return true, fmt.Errorf(`failed to decode field %q: %w`, field, err)
	}
	return true, nil

}

func (f *stdField) UnmarshalJSON(data []byte) error {
	f.name = ""
	f.typ = ""
	f.jsonName = ""

	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf(`failed to read next token: %w`, err)
	}

	switch tok := tok.(type) {
	case json.Delim:
		if tok != '{' {
			return fmt.Errorf(`expected '{', got %#v`, tok)
		}
	default:
		return fmt.Errorf(`expected '{', got %#v`, tok)
	}

OUTER:
	for {
		tok, err := dec.Token()
		if err != nil {
			return fmt.Errorf(`failed to read next token: %w`, err)
		}

		switch tok := tok.(type) {
		case json.Delim:
			if tok == '}' {
				break OUTER
			}
			return fmt.Errorf(`unexpected delimiter %#v`, tok)
		case string:
			handled, err := f.handleJSONField(dec, tok)
			if err != nil {
				return err
			}

			if handled {
				continue OUTER
			}
		default:
			return fmt.Errorf(`invalid field key %#v`, tok)
		}
	}
	return nil
}

type ConstantField struct {
	stdField
	value interface{}
}

func (f *ConstantField) UnmarshalJSON(data []byte) error {
	f.name = ""
	f.typ = ""
	f.jsonName = ""
	f.value = nil

	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf(`failed to read next token: %w`, err)
	}

	switch tok := tok.(type) {
	case json.Delim:
		if tok != '{' {
			return fmt.Errorf(`expected '{', got %#v`, tok)
		}
	default:
		return fmt.Errorf(`expected '{', got %#v`, tok)
	}

OUTER:
	for {
		tok, err := dec.Token()
		if err != nil {
			return fmt.Errorf(`failed to read next token: %w`, err)
		}

		switch tok := tok.(type) {
		case json.Delim:
			if tok == '}' {
				break OUTER
			}
			return fmt.Errorf(`unexpected delimiter %#v`, tok)
		case string:
			handled, err := f.handleJSONField(dec, tok)
			if err != nil {
				return err
			}

			if handled {
				continue OUTER
			}

			var fref interface{}
			switch tok {
			case "constant":
				fref = &f.value
			default:
				return fmt.Errorf(`invalid constant field key %#v`, tok)
			}

			if err := dec.Decode(fref); err != nil {
				return fmt.Errorf(`failed to decode field %q: %w`, tok, err)
			}
		}
	}

	return nil
}

func (f *ConstantField) Value() interface{} {
	return f.value
}
