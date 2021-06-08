// Code generated - DO NOT EDIT.

package model

import (
	"reflect"
	"unsafe"

	"github.com/DataDog/datadog-agent/pkg/security/secl/eval"
)

// suppress unused package warning
var (
	_ *unsafe.Pointer
)

func (m *Model) GetIterator(field eval.Field) (eval.Iterator, error) {
	switch field {

	}

	return nil, &eval.ErrIteratorNotSupported{Field: field}
}

func (m *Model) GetEventTypes() []eval.EventType {
	return []eval.EventType{}
}

func (m *Model) GetEvaluator(field eval.Field, regID eval.RegisterID) (eval.Evaluator, error) {
	switch field {

	}

	return nil, &eval.ErrFieldNotFound{Field: field}
}

func (e *Event) GetFields() []eval.Field {
	return []eval.Field{}
}

func (e *Event) GetFieldValue(field eval.Field) (interface{}, error) {
	switch field {

	}

	return nil, &eval.ErrFieldNotFound{Field: field}
}

func (e *Event) GetFieldEventType(field eval.Field) (eval.EventType, error) {
	switch field {

	}

	return "", &eval.ErrFieldNotFound{Field: field}
}

func (e *Event) GetFieldType(field eval.Field) (reflect.Kind, error) {
	switch field {

	}

	return reflect.Invalid, &eval.ErrFieldNotFound{Field: field}
}

func (e *Event) SetFieldValue(field eval.Field, value interface{}) error {
	switch field {

	}

	return &eval.ErrFieldNotFound{Field: field}
}
