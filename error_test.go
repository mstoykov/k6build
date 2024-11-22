package k6build

import (
	"errors"
	"fmt"
	"testing"
)

func Test_APIError(t *testing.T) {
	t.Parallel()

	var (
		err    = errors.New("error")
		reason = errors.New("reason")
	)

	testCases := []struct {
		title        string
		err          error
		reason       error
		expectError  error
		expectReason error
	}{
		{
			title:        "error and reason",
			err:          err,
			reason:       reason,
			expectError:  err,
			expectReason: reason,
		},
		{
			title:        "error not reason",
			err:          err,
			reason:       nil,
			expectError:  err,
			expectReason: ErrReasonUnknown,
		},
		{
			title:        "wrapped err",
			err:          fmt.Errorf("wrapped %w", err),
			reason:       reason,
			expectError:  err,
			expectReason: reason,
		},
		{
			title:        "wrapped reason",
			err:          errors.New("another error"),
			reason:       fmt.Errorf("wrapped %w", reason),
			expectError:  reason,
			expectReason: reason,
		},
		{
			title:        "wrapped err in target",
			err:          err,
			reason:       reason,
			expectError:  fmt.Errorf("wrapped %w", err),
			expectReason: reason,
		},
		{
			title:        "wrapped reason in target",
			err:          err,
			reason:       reason,
			expectError:  fmt.Errorf("wrapped %w", reason),
			expectReason: reason,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			t.Parallel()

			apiError := NewError(tc.err, tc.reason)

			if !errors.Is(apiError, tc.expectError) {
				t.Fatalf("expected %v got %v", tc.expectError, apiError)
			}

			if !errors.Is(errors.Unwrap(apiError), tc.expectReason) {
				t.Fatalf("expected %v got %v", tc.expectError, apiError)
			}
		})
	}
}
