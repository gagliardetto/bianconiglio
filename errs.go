// Package bianconiglio provides an error wrapper where you can add arbitrary fields
// to provide more context to the error.
// bianconiglio Error type is completely marshalable into json, and contains stack
// trace by default.
package bianconiglio

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gagliardetto/treeout"
)

type marshalableError struct {
	Err     error  `json:"err"`
	Context Fields `json:"context"`
	Stack   Fields `json:"stack"`

	mu *sync.RWMutex
}

// Error is the error interface of bianconiglio
type Error interface {
	error
	json.Marshaler
	toMap() map[string]interface{}
	tree(parent treeout.Branches) treeout.Branches

	Timestamp() time.Time
	CauseInterface
}

func newMarshalableError() *marshalableError {
	return &marshalableError{
		Context: make(Fields),
		Stack:   make(Fields),
		mu:      &sync.RWMutex{},
	}
}

type CauseInterface interface {
	Cause() error
}

// toMap converts Error to map[string]interface{}
func (e *marshalableError) toMap() map[string]interface{} {
	result := make(map[string]interface{})
	ctx := make(map[string]interface{})

	switch v := e.Err.(type) {
	case Error:
		result["err"] = v.toMap()
	case error:
		result["err"] = v.Error()
	}

	for k, i := range e.Context {

		switch v := i.(type) {
		case Error:
			ctx[k] = v.toMap()
		case error:
			ctx[k] = v.Error()
		default:
			ctx[k] = v
		}
	}
	result["context"] = ctx
	result["stack"] = e.Stack

	return result
}

// MarshalJSON marshals Error into json
func (e *marshalableError) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.toMap())
}

// Error formats the Error into a tree rapresentation of the error
func (e *marshalableError) Error() string {
	return e.tree(treeout.New("").Child("")).String()
}

func (e *marshalableError) Cause() error {
	if cause, ok := e.Err.(CauseInterface); ok {
		return cause.Cause()
	}
	return e.Err
}

// tree adds branches to parent tree
func (e *marshalableError) tree(parent treeout.Branches) treeout.Branches {
	errt := parent.Child("ERR")

	switch v := e.Err.(type) {
	case Error:
		errt = v.tree(errt)
	case error:
		errt.Child(fmt.Sprint(v.Error()))
	default:
		errt.Child(fmt.Sprint(v))
	}

	ctxT := parent.Child("CTX")

	for k, i := range e.Context {
		switch v := i.(type) {
		case Error:
			ctxT = v.tree(ctxT)
		case error:
			ctxT.Child(fmt.Sprintf("%v: %v", k, v.Error()))
		default:
			ctxT.Child(fmt.Sprintf("%v: %v", k, fmt.Sprint(v)))
		}
	}

	stack := parent.Child("STACK")

	for k, i := range e.Stack {
		switch v := i.(type) {
		case Error:
			stack = v.tree(stack)
		case error:
			stack.Child(fmt.Sprintf("%v: %v", k, v.Error()))
		default:
			stack.Child(fmt.Sprintf("%v: %v", k, fmt.Sprint(v)))
		}
	}

	return parent
}

// Timestamp returns the timestamp (if any) of the error;
// otherwise, a zero time is returned.
func (e *marshalableError) Timestamp() time.Time {
	if v, ok := e.Context["timestamp"]; ok {

		// the timestamp is in the context as string
		if ts, ok := v.(string); ok {
			t, err := time.Parse(time.RFC3339, ts)
			if err == nil {
				return t
			}
		}
	}

	return time.Time{}
}

// Fields is just map[string]interface{}
type Fields map[string]interface{}

// Contextualize lets you add context fields to the error in the style of Log15
func Contextualize(err error, keyVals ...interface{}) Error {
	er := newMarshalableError()
	er.mu.Lock()
	defer er.mu.Unlock()

	// add timestamp; the timestamp can be modified by user if they decide
	// to set a different timestamp.
	er.Context["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	for k, v := range parseKeyVals(keyVals...) {
		er.Context[k] = v
	}

	er.Err = err
	er.SetLocation(1)
	return er
}

func parseKeyVals(keyvals ...interface{}) map[string]interface{} {
	if len(keyvals) == 0 {
		return nil
	}
	meta := make(map[string]interface{}, (len(keyvals)+1)/2)
	for i := 0; i < len(keyvals); i += 2 {
		k := keyvals[i]
		var v interface{} = "MISSING"
		if i+1 < len(keyvals) {
			v = keyvals[i+1]
		}
		meta[fmt.Sprint(k)] = v
	}
	return meta
}
