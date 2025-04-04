package cmd

import (
	"slices"
	"testing"
)

func TestRemoveIndexFromSlice(t *testing.T) {
	t.Parallel()

	s := []string{"one", "two", "three", "four"}
	s = removeIndexFromSlice(s, 1)
	if s[1] != "three" {
		t.Fatal(s[1], s)
	}

	if cap(s) != 3 {
		t.Fatal(cap(s))
	}

	newS := []string{"one", "three", "four"}

	if slices.Compare(s, newS) != 0 {
		t.Fatalf("s (%v) and newS (%v) should be the same", s, newS)
	}
}
