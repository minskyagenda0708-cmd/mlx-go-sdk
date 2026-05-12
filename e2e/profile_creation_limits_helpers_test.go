//go:build e2e
// +build e2e

package e2e

import (
	"reflect"
	"testing"
	"time"
)

func TestAvailableProfileSlotsUsesActiveAndTrash(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		cap    int
		active int
		trash  int
		want   int
	}{
		{name: "empty account", cap: 50, active: 0, trash: 0, want: 50},
		{name: "trash counts against cap", cap: 50, active: 40, trash: 10, want: 0},
		{name: "mixed active and trash", cap: 50, active: 12, trash: 8, want: 30},
		{name: "never negative", cap: 50, active: 45, trash: 10, want: 0},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := availableProfileSlots(tc.cap, tc.active, tc.trash); got != tc.want {
				t.Fatalf("availableProfileSlots(%d, %d, %d) = %d, want %d", tc.cap, tc.active, tc.trash, got, tc.want)
			}
		})
	}
}

func TestInitialBatchProbeCandidatesIncludeDocumentedBoundary(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		freeSlots int
		want      []int
	}{
		{name: "no free slots", freeSlots: 0, want: nil},
		{name: "single free slot", freeSlots: 1, want: []int{1}},
		{name: "exact documented boundary", freeSlots: 10, want: []int{1, 10}},
		{name: "can test boundary and overflow", freeSlots: 11, want: []int{1, 10, 11}},
		{name: "large free space", freeSlots: 50, want: []int{1, 10, 11, 50}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := initialBatchProbeCandidates(tc.freeSlots); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("initialBatchProbeCandidates(%d) = %#v, want %#v", tc.freeSlots, got, tc.want)
			}
		})
	}
}

func TestSplitIntoCreateBatchSizesBuildsTenSizedPlan(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		total     int
		batchSize int
		want      []int
	}{
		{name: "zero total", total: 0, batchSize: 10, want: nil},
		{name: "single batch", total: 10, batchSize: 10, want: []int{10}},
		{name: "fifty profiles", total: 50, batchSize: 10, want: []int{10, 10, 10, 10, 10}},
		{name: "remainder batch", total: 23, batchSize: 10, want: []int{10, 10, 3}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := splitIntoCreateBatchSizes(tc.total, tc.batchSize); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("splitIntoCreateBatchSizes(%d, %d) = %#v, want %#v", tc.total, tc.batchSize, got, tc.want)
			}
		})
	}
}

func TestCreateFiftyIntervalCandidatesStartAtZeroAndIncrease(t *testing.T) {
	t.Parallel()

	want := []time.Duration{
		0,
		250 * time.Millisecond,
		500 * time.Millisecond,
		time.Second,
		1500 * time.Millisecond,
		2 * time.Second,
		3 * time.Second,
		5 * time.Second,
	}

	if got := createFiftyIntervalCandidates(); !reflect.DeepEqual(got, want) {
		t.Fatalf("createFiftyIntervalCandidates() = %#v, want %#v", got, want)
	}
}
