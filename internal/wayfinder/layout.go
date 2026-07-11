package wayfinder

// Layers groups the tickets into columns by dependency depth. A ticket's rank
// is one past the deepest ticket it is blocked by, so every blocker sits in a
// column to the left of what it blocks and the graph reads as left-to-right
// dependency flow. Within a rank, tickets keep the ticket-number order Load
// already sorted them into.
//
// The result has one entry per rank, densely packed from rank 0; an effort with
// no tickets yields a single empty layer so callers need not special-case nil.
func (e *Effort) Layers() [][]*Ticket {
	rank := make(map[int]int, len(e.Tickets))

	// Longest path from a root, memoised. The seen set guards against a
	// blocked_by cycle: lint forbids one, but serve renders whatever is on disk,
	// including a map caught mid-edit, and must not spin. A back-edge simply
	// contributes no depth.
	var depth func(t *Ticket, seen map[int]bool) int
	depth = func(t *Ticket, seen map[int]bool) int {
		if r, ok := rank[t.Num]; ok {
			return r
		}
		if seen[t.Num] {
			return 0
		}
		seen[t.Num] = true
		r := 0
		for _, b := range t.BlockedBy {
			dep := e.ByNum(b)
			if dep == nil {
				continue
			}
			if d := depth(dep, seen) + 1; d > r {
				r = d
			}
		}
		delete(seen, t.Num)
		rank[t.Num] = r
		return r
	}

	maxRank := 0
	for _, t := range e.Tickets {
		if r := depth(t, map[int]bool{}); r > maxRank {
			maxRank = r
		}
	}

	layers := make([][]*Ticket, maxRank+1)
	for _, t := range e.Tickets {
		layers[rank[t.Num]] = append(layers[rank[t.Num]], t)
	}
	return layers
}
