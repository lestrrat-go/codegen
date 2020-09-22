package codegen

const (
	optKeyFormatCode = `optkey-format-code`
)

type Option interface {
	Name() string
	Value() interface{}
}

type option struct {
	name  string
	value interface{}
}

func (o option) Name() string {
	return o.name
}

func (o option) Value() interface{} {
	return o.value
}

func WithFormatCode(b bool) Option {
	return &option{name: optKeyFormatCode, value: b}
}
