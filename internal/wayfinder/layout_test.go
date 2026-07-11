package wayfinder

import "testing"

func TestLayersRanksByDependencyDepth(t *testing.T) {
	// 01, 02 are roots; 03 waits on both; 04 waits on 03. 09 is an island.
	e := effort(okMap,
		tk(1, StatusResolved),
		tk(2, StatusResolved),
		tk(3, StatusOpen, 1, 2),
		tk(4, StatusOpen, 3),
		tk(9, StatusOpen),
	)
	layers := e.Layers()
	if len(layers) != 3 {
		t.Fatalf("layers = %d, want 3", len(layers))
	}

	nums := func(ts []*Ticket) []int {
		var out []int
		for _, t := range ts {
			out = append(out, t.Num)
		}
		return out
	}
	// Rank 0: the roots and the island, in ticket-number order.
	if got := nums(layers[0]); len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 9 {
		t.Errorf("rank 0 = %v, want [1 2 9]", got)
	}
	if got := nums(layers[1]); len(got) != 1 || got[0] != 3 {
		t.Errorf("rank 1 = %v, want [3]", got)
	}
	if got := nums(layers[2]); len(got) != 1 || got[0] != 4 {
		t.Errorf("rank 2 = %v, want [4]", got)
	}
}

// A blocked_by cycle is invalid, but serve renders whatever is on disk. Layers
// must terminate rather than spin, and place every ticket somewhere.
func TestLayersToleratesCycle(t *testing.T) {
	e := effort(okMap, tk(1, StatusOpen, 2), tk(2, StatusOpen, 1))
	layers := e.Layers()
	seen := 0
	for _, l := range layers {
		seen += len(l)
	}
	if seen != 2 {
		t.Errorf("placed %d tickets, want 2 — a cycle dropped one", seen)
	}
}
