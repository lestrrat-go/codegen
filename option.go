package codegen

import "github.com/lestrrat-go/option"

type Option = option.Interface

type identFormatCode struct{}
type identLineNumber struct{}

func WithFormatCode(b bool) Option {
	return option.New(identFormatCode{}, b)
}

func WithLineNumber(b bool) Option {
	return option.New(identLineNumber{}, b)
}
