package main

import "sort"

// E2dIndex converts n to an index in E2d
// water example:
// options for steps:
// 1 2 3 4 5 6 7 8 9 -1 -2 -3 -4 -5 -6 -7 -8 -9
// indices:
// 0 1 2 3 4 5 6 7 8  9 10 11 12 13 14 15 16 17
// grid is then 18x18 = 2ncoords x 2ncoords
func E2dIndex(ncoords int, ns ...int) []int {
	out := make([]int, 0)
	for _, n := range ns {
		if n < 0 {
			out = append(out, IntAbs(n)+ncoords)
		} else {
			out = append(out, n)
		}
	}
	return Index(2*ncoords, false, out...)
}

// Index returns the 1-dimensional array index of force constants in
// 2,3,4-D arrays
func Index(ncoords int, nosort bool, id ...int) []int {
	if !nosort {
		sort.Ints(id)
	}
	switch len(id) {
	case 2:
		if id[0] == id[1] {
			return []int{ncoords*(id[0]-1) + id[1] - 1}
		}
		return []int{
			ncoords*(id[0]-1) + id[1] - 1,
			ncoords*(id[1]-1) + id[0] - 1,
		}
	case 3:
		return []int{
			id[0] + (id[1]-1)*id[1]/2 +
				(id[2]-1)*id[2]*(id[2]+1)/6 - 1,
		}
	case 4:
		return []int{
			id[0] + (id[1]-1)*id[1]/2 +
				(id[2]-1)*id[2]*(id[2]+1)/6 +
				(id[3]-1)*id[3]*(id[3]+1)*(id[3]+2)/24 - 1,
		}
	}
	panic("wrong number of indices in call to Index")
}
