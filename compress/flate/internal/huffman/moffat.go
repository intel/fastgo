// Copyright (c) 2023, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package huffman

// MoffatHuffmanCode implements In-Place Calculation of Minimum-Redundancy Codes.
// Check http://hjemmesider.diku.dk/~jyrki/Paper/WADS95.pdf .
type MoffatHuffmanCode struct{}

type litCount struct {
	lit   uint16
	count uint16
}

type decLitCounts []litCount

// Len is the number of elements in the collection.
func (c decLitCounts) Len() int {
	return len(c)
}

// Less compare two elements
func (c decLitCounts) Less(i int, j int) bool {
	return c[i].count > c[j].count
}

// Swap swaps the elements with indexes i and j.
func (c decLitCounts) Swap(i int, j int) {
	c[i], c[j] = c[j], c[i]
}

func (m *MoffatHuffmanCode) codeLens(w []uint32) uint32 {
	// phase 1
	n := len(w)
	if n == 0 {
		return 0
	}
	if n == 1 {
		w[0] = 1
		return 1
	}
	leaf := n - 1
	root := n - 1
	for next := n - 1; next >= 1; next-- {
		// find first child
		if leaf < 0 || (root > next && w[root] < w[leaf]) {
			// use internal code
			w[next] = w[root]
			w[root] = uint32(next)
			root--
		} else {
			// use leaf node
			w[next] = w[leaf]
			leaf--
		}

		// find second child
		if leaf < 0 || (root > next && w[root] < w[leaf]) {
			// use internal code
			w[next] += w[root]
			w[root] = uint32(next)
			root--
		} else {
			// use leaf node
			w[next] += w[leaf]
			leaf--
		}
	}
	// phase 2
	w[1] = 0
	for next := 2; next <= n-1; next++ {
		w[next] = w[w[next]] + 1
	}
	// phase 3
	avail := 1
	used := 0
	depth := 0
	root = 1
	next := 0
	for avail > 0 {
		//  count internal nodes used at depth depth

		for ; root < n && w[root] == uint32(depth); root++ {
			used++
		}
		// assign as leaves any nodes that are not internal
		for ; avail > used; avail-- {
			w[next] = uint32(depth)
			next = next + 1
		}
		avail = 2 * used
		depth++
		used = 0
	}
	return w[len(w)-1]
}

// Generate huffman tree from histogram and write tree codes into codes.
func (m *MoffatHuffmanCode) Generate(histogram []uint32) {
	counts := make(decLitCounts, 0, len(histogram)/2)
	for i, v := range histogram {
		if v != 0 {
			counts = append(counts, litCount{
				lit:   uint16(i),
				count: uint16(v),
			})
		}
	}
	sortDecLitCounts(counts)
	w := make([]uint32, len(counts))
	for i, v := range counts {
		w[i] = uint32(v.count)
	}

	m.codeLens(w)
	for i := range histogram {
		histogram[i] = 0
	}
	for i, v := range w {
		histogram[counts[i].lit] = v
	}
}
