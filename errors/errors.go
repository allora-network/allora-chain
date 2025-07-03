package errors

import (
	"errors"
	"fmt"

	cerrors "cosmossdk.io/errors"
)

type Error = cerrors.Error

var NewCosmos = cerrors.New
var New = errors.New
var Errorf = fmt.Errorf
var Register = cerrors.Register
var RegisterWithGRPCCode = cerrors.RegisterWithGRPCCode
var Wrap = cerrors.Wrap
var Wrapf = cerrors.Wrapf
var IsOf = cerrors.IsOf
var Recover = cerrors.Recover
var WithType = cerrors.WithType
var ABCIError = cerrors.ABCIError

const UndefinedCodespace = cerrors.UndefinedCodespace

type ErrorWithFields struct {
	fields []any
	parent error
}

func NewWithFields(msg string, fields ...any) error {
	return WithFields(New(msg), fields...)
}

func WrapWithFields(err error, msg string, fields ...any) error {
	return WithFields(Wrap(err, msg), fields...)
}

func WithFields(err error, fields ...any) error {
	return &ErrorWithFields{
		parent: err,
		fields: fields,
	}
}

func Fields(err error) []any {
	var fields []any
	for {
		var errf ErrorWithFields
		if !errors.As(err, &errf) {
			break
		}
		for k, v := range errf.fields {
			fields = append(fields, k, v)
		}
		err = errf.parent
	}
	return fields
}

func (ef *ErrorWithFields) Error() string {
	if ef.parent != nil {
		return ef.parent.Error()
	}
	return "error with fields"
}

func (ef *ErrorWithFields) Unwrap() error {
	return ef.parent
}
