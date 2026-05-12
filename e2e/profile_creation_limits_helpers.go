//go:build e2e
// +build e2e

package e2e

import "time"

func availableProfileSlots(cap, active, trash int) int {
	free := cap - active - trash
	if free < 0 {
		return 0
	}
	return free
}

func initialBatchProbeCandidates(freeSlots int) []int {
	if freeSlots <= 0 {
		return nil
	}

	out := []int{1}
	if freeSlots >= 10 {
		out = append(out, 10)
	}
	if freeSlots >= 11 {
		out = append(out, 11)
	}
	if freeSlots > 11 {
		out = append(out, freeSlots)
	}
	return out
}

func splitIntoCreateBatchSizes(total, batchSize int) []int {
	if total <= 0 || batchSize <= 0 {
		return nil
	}

	out := make([]int, 0, (total+batchSize-1)/batchSize)
	remaining := total
	for remaining > 0 {
		size := batchSize
		if remaining < batchSize {
			size = remaining
		}
		out = append(out, size)
		remaining -= size
	}
	return out
}

func createFiftyIntervalCandidates() []time.Duration {
	return []time.Duration{
		0,
		250 * time.Millisecond,
		500 * time.Millisecond,
		time.Second,
		1500 * time.Millisecond,
		2 * time.Second,
		3 * time.Second,
		5 * time.Second,
	}
}
