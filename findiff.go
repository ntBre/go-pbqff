package main

// Make1D makes the Job slices for finite differences first
// derivative force constants
func Make1D(i int) []ProtoCalc {
	// Not sure about this one
	scale := angbohr / (2 * Conf.FlSlice(Deltas)[i-1])
	return []ProtoCalc{
		{1, HashName(), []int{i}, []int{i}, scale},
		{-1, HashName(), []int{-i}, []int{i}, scale},
	}
}

// Make2D makes the Job slices for finite differences second
// derivative force constants
func Make2D(i, j int) []ProtoCalc {
	scale := angbohr * angbohr / (4 * Conf.FlSlice(Deltas)[i-1] * Conf.FlSlice(Deltas)[j-1])
	switch {
	case i == j:
		// E(+i+i) - 2*E(0) + E(-i-i) / (2d)^2
		return []ProtoCalc{
			{1, HashName(), []int{i, i}, []int{i, i}, scale},
			{-2, "E0", []int{}, []int{i, i}, scale},
			{1, HashName(), []int{-i, -i}, []int{i, i}, scale},
		}
	case i != j:
		// E(+i+j) - E(+i-j) - E(-i+j) + E(-i-j) / (2d)^2
		return []ProtoCalc{
			{1, HashName(), []int{i, j}, []int{i, j}, scale},
			{-1, HashName(), []int{i, -j}, []int{i, j}, scale},
			{-1, HashName(), []int{-i, j}, []int{i, j}, scale},
			{1, HashName(), []int{-i, -j}, []int{i, j}, scale},
		}
	default:
		panic("No cases matched")
	}
}

// Make3D makes the ProtoCalc slices for finite differences third derivative
// force constants
func Make3D(i, j, k int) []ProtoCalc {
	scale := angbohr * angbohr * angbohr / (8 * Conf.FlSlice(Deltas)[i-1] * Conf.FlSlice(Deltas)[j-1] * Conf.FlSlice(Deltas)[k-1])
	switch {
	case i == j && i == k:
		// E(+i+i+i) - 3*E(i) + 3*E(-i) -E(-i-i-i) / (2d)^3
		return []ProtoCalc{
			{1, HashName(), []int{i, i, i}, []int{i, i, i}, scale},
			{-3, HashName(), []int{i}, []int{i, i, i}, scale},
			{3, HashName(), []int{-i}, []int{i, i, i}, scale},
			{-1, HashName(), []int{-i, -i, -i}, []int{i, i, i}, scale},
		}
	case i == j && i != k:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, k}, []int{i, i, k}, scale},
			{-2, HashName(), []int{k}, []int{i, i, k}, scale},
			{1, HashName(), []int{-i, -i, k}, []int{i, i, k}, scale},
			{-1, HashName(), []int{i, i, -k}, []int{i, i, k}, scale},
			{2, HashName(), []int{-k}, []int{i, i, k}, scale},
			{-1, HashName(), []int{-i, -i, -k}, []int{i, i, k}, scale},
		}
	case i == k && i != j:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, j}, []int{i, i, j}, scale},
			{-2, HashName(), []int{j}, []int{i, i, j}, scale},
			{1, HashName(), []int{-i, -i, j}, []int{i, i, j}, scale},
			{-1, HashName(), []int{i, i, -j}, []int{i, i, j}, scale},
			{2, HashName(), []int{-j}, []int{i, i, j}, scale},
			{-1, HashName(), []int{-i, -i, -j}, []int{i, i, j}, scale},
		}
	case j == k && i != j:
		return []ProtoCalc{
			{1, HashName(), []int{j, j, i}, []int{j, j, i}, scale},
			{-2, HashName(), []int{i}, []int{j, j, i}, scale},
			{1, HashName(), []int{-j, -j, i}, []int{j, j, i}, scale},
			{-1, HashName(), []int{j, j, -i}, []int{j, j, i}, scale},
			{2, HashName(), []int{-i}, []int{j, j, i}, scale},
			{-1, HashName(), []int{-j, -j, -i}, []int{j, j, i}, scale},
		}
	case i != j && i != k && j != k:
		return []ProtoCalc{
			{1, HashName(), []int{i, j, k}, []int{i, j, k}, scale},
			{-1, HashName(), []int{i, -j, k}, []int{i, j, k}, scale},
			{-1, HashName(), []int{-i, j, k}, []int{i, j, k}, scale},
			{1, HashName(), []int{-i, -j, k}, []int{i, j, k}, scale},
			{-1, HashName(), []int{i, j, -k}, []int{i, j, k}, scale},
			{1, HashName(), []int{i, -j, -k}, []int{i, j, k}, scale},
			{1, HashName(), []int{-i, j, -k}, []int{i, j, k}, scale},
			{-1, HashName(), []int{-i, -j, -k}, []int{i, j, k}, scale},
		}
	default:
		panic("No cases matched")
	}
}

// Make4D makes the ProtoCalc slices for finite differences fourth
// derivative force constants
func Make4D(i, j, k, l int) []ProtoCalc {
	scale := angbohr * angbohr * angbohr * angbohr /
		(16 * Conf.FlSlice(Deltas)[i-1] *
			Conf.FlSlice(Deltas)[j-1] *
			Conf.FlSlice(Deltas)[k-1] *
			Conf.FlSlice(Deltas)[l-1])
	switch {
	// all the same
	case i == j && i == k && i == l:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, i, i}, []int{i, i, i, i}, scale},
			{-4, HashName(), []int{i, i}, []int{i, i, i, i}, scale},
			{6, "E0", []int{}, []int{i, i, i, i}, scale},
			{-4, HashName(), []int{-i, -i}, []int{i, i, i, i}, scale},
			{1, HashName(), []int{-i, -i, -i, -i}, []int{i, i, i, i}, scale},
		}
	// 3 and 1
	case i == j && i == k && i != l:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, i, l}, []int{i, i, i, l}, scale},
			{-3, HashName(), []int{i, l}, []int{i, i, i, l}, scale},
			{3, HashName(), []int{-i, l}, []int{i, i, i, l}, scale},
			{-1, HashName(), []int{-i, -i, -i, l}, []int{i, i, i, l}, scale},
			{-1, HashName(), []int{i, i, i, -l}, []int{i, i, i, l}, scale},
			{3, HashName(), []int{i, -l}, []int{i, i, i, l}, scale},
			{-3, HashName(), []int{-i, -l}, []int{i, i, i, l}, scale},
			{1, HashName(), []int{-i, -i, -i, -l}, []int{i, i, i, l}, scale},
		}
	case i == j && i == l && i != k:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, i, k}, []int{i, i, i, k}, scale},
			{-3, HashName(), []int{i, k}, []int{i, i, i, k}, scale},
			{3, HashName(), []int{-i, k}, []int{i, i, i, k}, scale},
			{-1, HashName(), []int{-i, -i, -i, k}, []int{i, i, i, k}, scale},
			{-1, HashName(), []int{i, i, i, -k}, []int{i, i, i, k}, scale},
			{3, HashName(), []int{i, -k}, []int{i, i, i, k}, scale},
			{-3, HashName(), []int{-i, -k}, []int{i, i, i, k}, scale},
			{1, HashName(), []int{-i, -i, -i, -k}, []int{i, i, i, k}, scale},
		}
	case i == k && i == l && i != j:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, i, j}, []int{i, i, i, j}, scale},
			{-3, HashName(), []int{i, j}, []int{i, i, i, j}, scale},
			{3, HashName(), []int{-i, j}, []int{i, i, i, j}, scale},
			{-1, HashName(), []int{-i, -i, -i, j}, []int{i, i, i, j}, scale},
			{-1, HashName(), []int{i, i, i, -j}, []int{i, i, i, j}, scale},
			{3, HashName(), []int{i, -j}, []int{i, i, i, j}, scale},
			{-3, HashName(), []int{-i, -j}, []int{i, i, i, j}, scale},
			{1, HashName(), []int{-i, -i, -i, -j}, []int{i, i, i, j}, scale},
		}
	case j == k && j == l && j != i:
		return []ProtoCalc{
			{1, HashName(), []int{j, j, j, i}, []int{j, j, j, i}, scale},
			{-3, HashName(), []int{j, i}, []int{j, j, j, i}, scale},
			{3, HashName(), []int{-j, i}, []int{j, j, j, i}, scale},
			{-1, HashName(), []int{-j, -j, -j, i}, []int{j, j, j, i}, scale},
			{-1, HashName(), []int{j, j, j, -i}, []int{j, j, j, i}, scale},
			{3, HashName(), []int{j, -i}, []int{j, j, j, i}, scale},
			{-3, HashName(), []int{-j, -i}, []int{j, j, j, i}, scale},
			{1, HashName(), []int{-j, -j, -j, -i}, []int{j, j, j, i}, scale},
		}
	// 2 and 1 and 1
	case i == j && i != k && i != l && k != l:
		// x -> i, y -> k, z -> l
		return []ProtoCalc{
			{1, HashName(), []int{i, i, k, l}, []int{i, i, k, l}, scale},
			{-2, HashName(), []int{k, l}, []int{i, i, k, l}, scale},
			{1, HashName(), []int{-i, -i, k, l}, []int{i, i, k, l}, scale},
			{-1, HashName(), []int{i, i, -k, l}, []int{i, i, k, l}, scale},
			{2, HashName(), []int{-k, l}, []int{i, i, k, l}, scale},
			{-1, HashName(), []int{-i, -i, -k, l}, []int{i, i, k, l}, scale},
			{-1, HashName(), []int{i, i, k, -l}, []int{i, i, k, l}, scale},
			{2, HashName(), []int{k, -l}, []int{i, i, k, l}, scale},
			{-1, HashName(), []int{-i, -i, k, -l}, []int{i, i, k, l}, scale},
			{1, HashName(), []int{i, i, -k, -l}, []int{i, i, k, l}, scale},
			{-2, HashName(), []int{-k, -l}, []int{i, i, k, l}, scale},
			{1, HashName(), []int{-i, -i, -k, -l}, []int{i, i, k, l}, scale},
		}
	case i == k && i != j && i != l && j != l:
		// x -> i, y -> j, z -> l
		return []ProtoCalc{
			{1, HashName(), []int{i, i, j, l}, []int{i, i, j, l}, scale},
			{-2, HashName(), []int{j, l}, []int{i, i, j, l}, scale},
			{1, HashName(), []int{-i, -i, j, l}, []int{i, i, j, l}, scale},
			{-1, HashName(), []int{i, i, -j, l}, []int{i, i, j, l}, scale},
			{2, HashName(), []int{-j, l}, []int{i, i, j, l}, scale},
			{-1, HashName(), []int{-i, -i, -j, l}, []int{i, i, j, l}, scale},
			{-1, HashName(), []int{i, i, j, -l}, []int{i, i, j, l}, scale},
			{2, HashName(), []int{j, -l}, []int{i, i, j, l}, scale},
			{-1, HashName(), []int{-i, -i, j, -l}, []int{i, i, j, l}, scale},
			{1, HashName(), []int{i, i, -j, -l}, []int{i, i, j, l}, scale},
			{-2, HashName(), []int{-j, -l}, []int{i, i, j, l}, scale},
			{1, HashName(), []int{-i, -i, -j, -l}, []int{i, i, j, l}, scale},
		}
	case i == l && i != j && i != k && j != k:
		// x -> i, y -> k, z -> j
		return []ProtoCalc{
			{1, HashName(), []int{i, i, k, j}, []int{i, i, k, j}, scale},
			{-2, HashName(), []int{k, j}, []int{i, i, k, j}, scale},
			{1, HashName(), []int{-i, -i, k, j}, []int{i, i, k, j}, scale},
			{-1, HashName(), []int{i, i, -k, j}, []int{i, i, k, j}, scale},
			{2, HashName(), []int{-k, j}, []int{i, i, k, j}, scale},
			{-1, HashName(), []int{-i, -i, -k, j}, []int{i, i, k, j}, scale},
			{-1, HashName(), []int{i, i, k, -j}, []int{i, i, k, j}, scale},
			{2, HashName(), []int{k, -j}, []int{i, i, k, j}, scale},
			{-1, HashName(), []int{-i, -i, k, -j}, []int{i, i, k, j}, scale},
			{1, HashName(), []int{i, i, -k, -j}, []int{i, i, k, j}, scale},
			{-2, HashName(), []int{-k, -j}, []int{i, i, k, j}, scale},
			{1, HashName(), []int{-i, -i, -k, -j}, []int{i, i, k, j}, scale},
		}
	case j == k && j != i && j != l && i != l:
		// x -> j, y -> i, z -> l
		return []ProtoCalc{
			{1, HashName(), []int{j, j, i, l}, []int{j, j, i, l}, scale},
			{-2, HashName(), []int{i, l}, []int{j, j, i, l}, scale},
			{1, HashName(), []int{-j, -j, i, l}, []int{j, j, i, l}, scale},
			{-1, HashName(), []int{j, j, -i, l}, []int{j, j, i, l}, scale},
			{2, HashName(), []int{-i, l}, []int{j, j, i, l}, scale},
			{-1, HashName(), []int{-j, -j, -i, l}, []int{j, j, i, l}, scale},
			{-1, HashName(), []int{j, j, i, -l}, []int{j, j, i, l}, scale},
			{2, HashName(), []int{i, -l}, []int{j, j, i, l}, scale},
			{-1, HashName(), []int{-j, -j, i, -l}, []int{j, j, i, l}, scale},
			{1, HashName(), []int{j, j, -i, -l}, []int{j, j, i, l}, scale},
			{-2, HashName(), []int{-i, -l}, []int{j, j, i, l}, scale},
			{1, HashName(), []int{-j, -j, -i, -l}, []int{j, j, i, l}, scale},
		}
	case j == l && j != i && j != k && i != k:
		// x -> j, y -> i, z -> k
		return []ProtoCalc{
			{1, HashName(), []int{j, j, i, k}, []int{j, j, i, k}, scale},
			{-2, HashName(), []int{i, k}, []int{j, j, i, k}, scale},
			{1, HashName(), []int{-j, -j, i, k}, []int{j, j, i, k}, scale},
			{-1, HashName(), []int{j, j, -i, k}, []int{j, j, i, k}, scale},
			{2, HashName(), []int{-i, k}, []int{j, j, i, k}, scale},
			{-1, HashName(), []int{-j, -j, -i, k}, []int{j, j, i, k}, scale},
			{-1, HashName(), []int{j, j, i, -k}, []int{j, j, i, k}, scale},
			{2, HashName(), []int{i, -k}, []int{j, j, i, k}, scale},
			{-1, HashName(), []int{-j, -j, i, -k}, []int{j, j, i, k}, scale},
			{1, HashName(), []int{j, j, -i, -k}, []int{j, j, i, k}, scale},
			{-2, HashName(), []int{-i, -k}, []int{j, j, i, k}, scale},
			{1, HashName(), []int{-j, -j, -i, -k}, []int{j, j, i, k}, scale},
		}
	case k == l && k != i && k != j && i != j:
		// x -> k, y -> i, z -> j
		return []ProtoCalc{
			{1, HashName(), []int{k, k, i, j}, []int{k, k, i, j}, scale},
			{-2, HashName(), []int{i, j}, []int{k, k, i, j}, scale},
			{1, HashName(), []int{-k, -k, i, j}, []int{k, k, i, j}, scale},
			{-1, HashName(), []int{k, k, -i, j}, []int{k, k, i, j}, scale},
			{2, HashName(), []int{-i, j}, []int{k, k, i, j}, scale},
			{-1, HashName(), []int{-k, -k, -i, j}, []int{k, k, i, j}, scale},
			{-1, HashName(), []int{k, k, i, -j}, []int{k, k, i, j}, scale},
			{2, HashName(), []int{i, -j}, []int{k, k, i, j}, scale},
			{-1, HashName(), []int{-k, -k, i, -j}, []int{k, k, i, j}, scale},
			{1, HashName(), []int{k, k, -i, -j}, []int{k, k, i, j}, scale},
			{-2, HashName(), []int{-i, -j}, []int{k, k, i, j}, scale},
			{1, HashName(), []int{-k, -k, -i, -j}, []int{k, k, i, j}, scale},
		}
	// 2 and 2
	case i == j && k == l && i != k:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, k, k}, []int{i, i, k, k}, scale},
			{1, HashName(), []int{-i, -i, -k, -k}, []int{i, i, k, k}, scale},
			{1, HashName(), []int{-i, -i, k, k}, []int{i, i, k, k}, scale},
			{1, HashName(), []int{i, i, -k, -k}, []int{i, i, k, k}, scale},
			{-2, HashName(), []int{i, i}, []int{i, i, k, k}, scale},
			{-2, HashName(), []int{k, k}, []int{i, i, k, k}, scale},
			{-2, HashName(), []int{-i, -i}, []int{i, i, k, k}, scale},
			{-2, HashName(), []int{-k, -k}, []int{i, i, k, k}, scale},
			{4, "E0", []int{}, []int{i, i, k, k}, scale},
		}
	case i == k && j == l && i != j:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, j, j}, []int{i, i, j, j}, scale},
			{1, HashName(), []int{-i, -i, -j, -j}, []int{i, i, j, j}, scale},
			{1, HashName(), []int{-i, -i, j, j}, []int{i, i, j, j}, scale},
			{1, HashName(), []int{i, i, -j, -j}, []int{i, i, j, j}, scale},
			{-2, HashName(), []int{i, i}, []int{i, i, j, j}, scale},
			{-2, HashName(), []int{j, j}, []int{i, i, j, j}, scale},
			{-2, HashName(), []int{-i, -i}, []int{i, i, j, j}, scale},
			{-2, HashName(), []int{-j, -j}, []int{i, i, j, j}, scale},
			{4, "E0", []int{}, []int{i, i, j, j}, scale},
		}
	case i == l && j == k && i != j:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, j, j}, []int{i, i, j, j}, scale},
			{1, HashName(), []int{-i, -i, -j, -j}, []int{i, i, j, j}, scale},
			{1, HashName(), []int{-i, -i, j, j}, []int{i, i, j, j}, scale},
			{1, HashName(), []int{i, i, -j, -j}, []int{i, i, j, j}, scale},
			{-2, HashName(), []int{i, i}, []int{i, i, j, j}, scale},
			{-2, HashName(), []int{j, j}, []int{i, i, j, j}, scale},
			{-2, HashName(), []int{-i, -i}, []int{i, i, j, j}, scale},
			{-2, HashName(), []int{-j, -j}, []int{i, i, j, j}, scale},
			{4, "E0", []int{}, []int{i, i, j, j}, scale},
		}
	// all different
	case i != j && i != k && i != l && j != k && j != l && k != l:
		return []ProtoCalc{
			{1, HashName(), []int{i, j, k, l}, []int{i, j, k, l}, scale},
			{-1, HashName(), []int{i, -j, k, l}, []int{i, j, k, l}, scale},
			{-1, HashName(), []int{-i, j, k, l}, []int{i, j, k, l}, scale},
			{1, HashName(), []int{-i, -j, k, l}, []int{i, j, k, l}, scale},
			{-1, HashName(), []int{i, j, -k, l}, []int{i, j, k, l}, scale},
			{1, HashName(), []int{i, -j, -k, l}, []int{i, j, k, l}, scale},
			{1, HashName(), []int{-i, j, -k, l}, []int{i, j, k, l}, scale},
			{-1, HashName(), []int{-i, -j, -k, l}, []int{i, j, k, l}, scale},
			{-1, HashName(), []int{i, j, k, -l}, []int{i, j, k, l}, scale},
			{1, HashName(), []int{i, -j, k, -l}, []int{i, j, k, l}, scale},
			{1, HashName(), []int{-i, j, k, -l}, []int{i, j, k, l}, scale},
			{-1, HashName(), []int{-i, -j, k, -l}, []int{i, j, k, l}, scale},
			{1, HashName(), []int{i, j, -k, -l}, []int{i, j, k, l}, scale},
			{-1, HashName(), []int{i, -j, -k, -l}, []int{i, j, k, l}, scale},
			{-1, HashName(), []int{-i, j, -k, -l}, []int{i, j, k, l}, scale},
			{1, HashName(), []int{-i, -j, -k, -l}, []int{i, j, k, l}, scale},
		}
	default:
		panic("No cases matched")
	}
}
