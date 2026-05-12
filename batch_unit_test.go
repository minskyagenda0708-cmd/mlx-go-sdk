package mlx

import (
	"errors"
	"testing"
)

func TestBatchProfileOperationErrorAggregatesFailures(t *testing.T) {
	errOne := errors.New("first")
	errTwo := errors.New("second")
	err := &BatchProfileOperationError{
		Operation: "start",
		Failures: []ProfileBatchFailure{
			{ProfileName: "Alpha", Err: errOne},
			{ProfileName: "Beta", Err: errTwo},
		},
	}

	if !errors.Is(err, errOne) || !errors.Is(err, errTwo) {
		t.Fatal("expected aggregated error to unwrap underlying failures")
	}
	if got := err.Error(); got == "" || got == "batch profile operation failed" {
		t.Fatalf("expected informative batch error string, got %q", got)
	}
}
