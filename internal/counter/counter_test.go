package counter

import (
	"testing"

	"github.com/ogdakke/symbolista/internal/domain"
)

func TestCharCountSorting(t *testing.T) {
	counts := domain.CharCounts{
		{Char: "a", Count: 5, Percentage: 50.0},
		{Char: "b", Count: 3, Percentage: 30.0},
		{Char: "c", Count: 2, Percentage: 20.0},
	}

	if counts[0].Count != 5 || counts[1].Count != 3 || counts[2].Count != 2 {
		t.Error("CharCounts should be sorted by count in descending order")
	}

	if counts.Less(0, 1) != true {
		t.Error("Less method should return true when first count is greater")
	}
	if counts.Less(1, 0) != false {
		t.Error("Less method should return false when first count is smaller")
	}

	if counts.Len() != 3 {
		t.Errorf("Expected length 3, got %d", counts.Len())
	}

	counts.Swap(0, 2)
	if counts[0].Char != "c" || counts[2].Char != "a" {
		t.Error("Swap method did not work correctly")
	}
}
