package main

import "sort"

// Index returns the 1-dimensional array index of force constants in
// 2,3,4-D arrays
func Index(ncoords int, nosort bool, id ...int) []int {
	if !nosort {
		sort.Ints(id)
	}
	switch len(id) {
	case 0:
		return []int{}
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
