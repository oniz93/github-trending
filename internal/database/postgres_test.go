package database

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lib/pq"
)

func TestIsConstraintViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "unique violation", err: &pq.Error{Code: "23505"}, want: true},
		{name: "foreign key violation", err: &pq.Error{Code: "23503"}, want: true},
		{name: "check violation", err: &pq.Error{Code: "23514"}, want: true},
		{name: "wrapped unique violation", err: fmt.Errorf("insert failed: %w", &pq.Error{Code: "23505"}), want: true},
		{name: "serialization failure", err: &pq.Error{Code: "40001"}, want: false},
		{name: "generic error", err: errors.New("connection refused"), want: false},
		{name: "nil error", err: nil, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsConstraintViolation(tc.err); got != tc.want {
				t.Fatalf("IsConstraintViolation(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
