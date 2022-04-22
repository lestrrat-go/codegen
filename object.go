package codegen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/lestrrat-go/xstrings"
)

var zerovals = map[string]string{
	`int`:       `0`,
	`int8`:      `0`,
	`int16`:     `0`,
	`int32`:     `0`,
	`int64`:     `0`,
	`uint`:      `0`,
	`uint8`:     `0`,
	`uint16`:    `0`,
	`uint32`:    `0`,
	`uint64`:    `0`,
	`float32`:   `0`,
	`float64`:   `0`,
	`string`:    `""`,
	`time.Time`: `time.Time{}`,
	`bool`:      `false`,
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
	comment        string
	extras         map[string]interface{}
}

func (b *base) Initialize() {
	b.name = ""
	b.exportedName = ""
	b.unexportedName = ""
	b.comment = ""
	b.extras = make(map[string]interface{})
}

func (b *base) Extra(s string) (interface{}, bool) {
	v, ok := b.extras[s]
	return v, ok
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
	case "comment":
		fref = &b.comment
	default:
		return false, nil
	}
	if err := dec.Decode(fref); err != nil {
		return true, fmt.Errorf(`failed to decode field %q: %w`, field, err)
	}
	return true, nil
}

func (b *base) Comment() string {
	return b.comment
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

func (o *Object) AddField(f Field) {
	o.fields = append(o.fields, f)
}

func (o *Object) UnmarshalJSON(data []byte) error {
	o.base.Initialize()
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
				var fl FieldList
				if err := dec.Decode(&fl); err != nil {
					return fmt.Errorf(`failed to decode field list: %w`, err)
				}
				o.fields = fl
				continue OUTER
			default:
				var v interface{}
				if err := dec.Decode(&v); err != nil {
					return fmt.Errorf(`failed to decode extra field %q: %w`, tok, err)
				}
				o.extras[tok] = v
				continue OUTER
			}

			if err := dec.Decode(fref); err != nil {
				return fmt.Errorf(`failed to decode field %q: %w`, tok, err)
			}
		}
	}
	return nil
}

// Returns the value of field `name` as a boolean
func (o *Object) Bool(name string) bool {
	v, _ := boolFrom(o, name, false)
	return v
}

func (o *Object) MustBool(name string) bool {
	v, _ := boolFrom(o, name, true)
	return v
}

// Returns the value of field `name` as a string. Returns empty value
// if the object stored in the field is not a string
func (o *Object) String(name string) string {
	v, _ := stringFrom(o, name, false)
	return v
}

func (o *Object) MustString(name string) string {
	v, _ := stringFrom(o, name, true)
	return v
}

type FieldList []Field

func (l *FieldList) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))

	var list []json.RawMessage
	if err := dec.Decode(&list); err != nil {
		return fmt.Errorf(`failed to decode field list to list of messages`)
	}

	*l = make([]Field, 0, len(list))
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

		*l = append(*l, f)
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

	Comment() string

	Extra(string) (interface{}, bool)

	IsRequired() bool
	IsConstant() bool

	Bool(string) bool
	MustBool(string) bool
	String(string) string
	MustString(string) string
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
	f.base.Initialize()

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
			var v interface{}
			if err := dec.Decode(&v); err != nil {
				return fmt.Errorf(`failed to decode extra field %q: %w`, tok, err)
			}
			f.extras[tok] = v
			continue OUTER
		default:
			return fmt.Errorf(`invalid token: %#v`, tok)
		}
	}
	return nil
}

func (f *stdField) MustBool(s string) bool {
	v, _ := boolFrom(f, s, true)
	return v
}

func (f *stdField) Bool(s string) bool {
	v, _ := boolFrom(f, s, false)
	return v
}

func (f *stdField) MustString(s string) string {
	v, _ := stringFrom(f, s, true)
	return v
}

func (f *stdField) String(s string) string {
	v, _ := stringFrom(f, s, false)
	return v
}

func (f *stdField) IsRequired() bool {
	return f.required
}

func (f *stdField) IsConstant() bool {
	return false
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
				var v interface{}
				if err := dec.Decode(&v); err != nil {
					return fmt.Errorf(`failed to decode extra field %q: %w`, tok, err)
				}
				f.extras[tok] = v
				continue OUTER
			}

			if err := dec.Decode(fref); err != nil {
				return fmt.Errorf(`failed to decode field %q: %w`, tok, err)
			}
		}
	}

	return nil
}

func (f *ConstantField) Bool(s string) bool {
	v, _ := boolFrom(f, s, false)
	return v
}

func (f *ConstantField) MustBool(s string) bool {
	v, _ := boolFrom(f, s, true)
	return v
}

func (f *ConstantField) String(s string) string {
	v, _ := stringFrom(f, s, false)
	return v
}

func (f *ConstantField) MustString(s string) string {
	v, _ := stringFrom(f, s, true)
	return v
}

func (f *ConstantField) Value() interface{} {
	return f.value
}

func (f *ConstantField) IsRequired() bool {
	return true
}

func (f *ConstantField) IsConstant() bool {
	return false
}

func boolFrom(src interface {
	Extra(string) (interface{}, bool)
}, field string, required bool) (bool, error) {
	v, ok := src.Extra(field)
	if !ok {
		err := fmt.Errorf("%q does not exist in %q", field, field)
		if required {
			panic(err.Error())
		}
		return false, err
	}

	b, ok := v.(bool)
	if !ok {
		err := fmt.Errorf("%q should be a bool in %q", field, field)
		if required {
			panic(err.Error())
		}
		return false, err
	}
	return b, nil
}

func stringFrom(src interface {
	Extra(string) (interface{}, bool)
}, field string, required bool) (string, error) {
	v, ok := src.Extra(field)
	if !ok {
		err := fmt.Errorf("%q does not exist in %q", field, field)
		if required {
			panic(err.Error())
		}
		return "", err
	}

	b, ok := v.(string)
	if !ok {
		err := fmt.Errorf("%q should be a string in %q", field, field)
		if required {
			panic(err.Error())
		}
		return "", err
	}
	return b, nil
}
