package k6build

import (
	"errors"
	"fmt"
	"testing"
)

func Test_Error(t *testing.T) {
	t.Parallel()

	var (
		err    = errors.New("error")
		reason = errors.New("reason")
	)

	testCases := []struct {
		title  string
		err    error
		reason error
		expect []error
	}{
		{
			title:  "error and reason",
			err:    err,
			reason: reason,
			expect: []error{err, reason},
		},
		{
			title:  "error not reason",
			err:    err,
			reason: nil,
			expect: []error{err},
		},
		{
			title:  "multiple and reasons",
			err:    err,
			reason: reason,
			expect: []error{err, reason},
		},
		{
			title:  "wrapped err",
			err:    fmt.Errorf("wrapped %w", err),
			reason: reason,
			expect: []error{err, reason},
		},
		{
			title:  "wrapped reason",
			err:    err,
			reason: fmt.Errorf("wrapped %w", reason),
			expect: []error{err, reason},
		},
		{
			title:  "wrapped err in target",
			err:    err,
			reason: reason,
			expect: []error{fmt.Errorf("wrapped %w", err)},
		},
		{
			title:  "wrapped reason in target",
			err:    err,
			reason: reason,
			expect: []error{fmt.Errorf("wrapped %w", reason)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			err := NewError(tc.err, tc.reason)
			for _, expected := range tc.expect {
				if !errors.Is(err, expected) {
					t.Fatalf("expected %v got %v", expected, err)
				}
			}
		})
	}
}
