package k6build

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Error represents an error returned by the build service
// This custom error type facilitates extracting the reason of an error
// by using errors.Unwrap method.
// It also facilitates checking an error (or its reason) using errors.Is by
// comparing the error and its reason.
// This custom type has the following known limitations:
// - A nil Error 'e' will not satisfy errors.Is(e, nil)
// - Is method will not
type Error struct {
	Err    error `json:"error,omitempty"`
	Reason error `json:"reason,omitempty"`
}

// Error returns the Error as a string
func (e *Error) Error() string {
	reason := ""
	if e.Reason != nil {
		reason = fmt.Sprintf(": %s", e.Reason)
	}
	return fmt.Sprintf("%s%s", e.Err, reason)
}

// Is returns true if the target error is the same as the Error or its reason
// It attempts several strategies:
// - compare error and reason to target's Error()
// - unwrap the error and reason and compare to target's Error
// - unwrap target and compares to the error recursively
func (e *Error) Is(target error) bool {
	if target == nil {
		return false
	}

	if e.Err.Error() == target.Error() {
		return true
	}

	if e.Reason != nil && e.Reason.Error() == target.Error() {
		return true
	}

	if u := errors.Unwrap(e.Err); u != nil && u.Error() == target.Error() {
		return true
	}

	if u := errors.Unwrap(e.Reason); u != nil && u.Error() == target.Error() {
		return true
	}

	return e.Is(errors.Unwrap(target))
}

// Unwrap returns the underlying reason for the Error
func (e *Error) Unwrap() error {
	return e.Reason
}

// MarshalJSON implements the json.Marshaler interface for the Error type
func (e *Error) MarshalJSON() ([]byte, error) {
	reason := ""
	if e.Reason != nil {
		reason = e.Reason.Error()
	}
	return json.Marshal(&struct {
		Err    string `json:"error,omitempty"`
		Reason string `json:"reason,omitempty"`
	}{
		Err:    e.Err.Error(),
		Reason: reason,
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface for the Error type
func (e *Error) UnmarshalJSON(data []byte) error {
	val := struct {
		Err    string `json:"error,omitempty"`
		Reason string `json:"reason,omitempty"`
	}{}

	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}

	e.Err = errors.New(val.Err)
	e.Reason = errors.New(val.Reason)
	return nil
}

// NewError creates an Error from an error and a reason
// If the reason is nil, ErrReasonUnknown is used
func NewError(err error, reasons ...error) *Error {
	var reason error
	if len(reasons) > 0 {
		reason = NewError(reasons[0], reasons[1:]...)
	}
	return &Error{
		Err:    err,
		Reason: reason,
	}
}
