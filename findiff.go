package main

// Make1D makes the Job slices for finite differences first
// derivative force constants
func Make1D(i int) []ProtoCalc {
	return []ProtoCalc{
		{1, HashName(), []int{i}, []int{i}},
		{-1, HashName(), []int{-i}, []int{i}},
	}
}

// Make2D makes the Job slices for finite differences second
// derivative force constants
func Make2D(i, j int) []ProtoCalc {
	switch {
	case i == j:
		// E(+i+i) - 2*E(0) + E(-i-i) / (2d)^2
		return []ProtoCalc{
			{1, HashName(), []int{i, i}, []int{i, i}},
			{-2, "E0", []int{}, []int{i, i}},
			{1, HashName(), []int{-i, -i}, []int{i, i}},
		}
	case i != j:
		// E(+i+j) - E(+i-j) - E(-i+j) + E(-i-j) / (2d)^2
		return []ProtoCalc{
			{1, HashName(), []int{i, j}, []int{i, j}},
			{-1, HashName(), []int{i, -j}, []int{i, j}},
			{-1, HashName(), []int{-i, j}, []int{i, j}},
			{1, HashName(), []int{-i, -j}, []int{i, j}},
		}
	default:
		panic("No cases matched")
	}
}

// Make3D makes the ProtoCalc slices for finite differences third derivative
// force constants
func Make3D(i, j, k int) []ProtoCalc {
	switch {
	case i == j && i == k:
		// E(+i+i+i) - 3*E(i) + 3*E(-i) -E(-i-i-i) / (2d)^3
		return []ProtoCalc{
			{1, HashName(), []int{i, i, i}, []int{i, i, i}},
			{-3, HashName(), []int{i}, []int{i, i, i}},
			{3, HashName(), []int{-i}, []int{i, i, i}},
			{-1, HashName(), []int{-i, -i, -i}, []int{i, i, i}},
		}
	case i == j && i != k:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, k}, []int{i, i, k}},
			{-2, HashName(), []int{k}, []int{i, i, k}},
			{1, HashName(), []int{-i, -i, k}, []int{i, i, k}},
			{-1, HashName(), []int{i, i, -k}, []int{i, i, k}},
			{2, HashName(), []int{-k}, []int{i, i, k}},
			{-1, HashName(), []int{-i, -i, -k}, []int{i, i, k}},
		}
	case i == k && i != j:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, j}, []int{i, i, j}},
			{-2, HashName(), []int{j}, []int{i, i, j}},
			{1, HashName(), []int{-i, -i, j}, []int{i, i, j}},
			{-1, HashName(), []int{i, i, -j}, []int{i, i, j}},
			{2, HashName(), []int{-j}, []int{i, i, j}},
			{-1, HashName(), []int{-i, -i, -j}, []int{i, i, j}},
		}
	case j == k && i != j:
		return []ProtoCalc{
			{1, HashName(), []int{j, j, i}, []int{j, j, i}},
			{-2, HashName(), []int{i}, []int{j, j, i}},
			{1, HashName(), []int{-j, -j, i}, []int{j, j, i}},
			{-1, HashName(), []int{j, j, -i}, []int{j, j, i}},
			{2, HashName(), []int{-i}, []int{j, j, i}},
			{-1, HashName(), []int{-j, -j, -i}, []int{j, j, i}},
		}
	case i != j && i != k && j != k:
		return []ProtoCalc{
			{1, HashName(), []int{i, j, k}, []int{i, j, k}},
			{-1, HashName(), []int{i, -j, k}, []int{i, j, k}},
			{-1, HashName(), []int{-i, j, k}, []int{i, j, k}},
			{1, HashName(), []int{-i, -j, k}, []int{i, j, k}},
			{-1, HashName(), []int{i, j, -k}, []int{i, j, k}},
			{1, HashName(), []int{i, -j, -k}, []int{i, j, k}},
			{1, HashName(), []int{-i, j, -k}, []int{i, j, k}},
			{-1, HashName(), []int{-i, -j, -k}, []int{i, j, k}},
		}
	default:
		panic("No cases matched")
	}
}

// Make4D makes the ProtoCalc slices for finite differences fourth
// derivative force constants
func Make4D(i, j, k, l int) []ProtoCalc {
	switch {
	// all the same
	case i == j && i == k && i == l:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, i, i}, []int{i, i, i, i}},
			{-4, HashName(), []int{i, i}, []int{i, i, i, i}},
			{6, "E0", []int{}, []int{i, i, i, i}},
			{-4, HashName(), []int{-i, -i}, []int{i, i, i, i}},
			{1, HashName(), []int{-i, -i, -i, -i}, []int{i, i, i, i}},
		}
	// 3 and 1
	case i == j && i == k && i != l:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, i, l}, []int{i, i, i, l}},
			{-3, HashName(), []int{i, l}, []int{i, i, i, l}},
			{3, HashName(), []int{-i, l}, []int{i, i, i, l}},
			{-1, HashName(), []int{-i, -i, -i, l}, []int{i, i, i, l}},
			{-1, HashName(), []int{i, i, i, -l}, []int{i, i, i, l}},
			{3, HashName(), []int{i, -l}, []int{i, i, i, l}},
			{-3, HashName(), []int{-i, -l}, []int{i, i, i, l}},
			{1, HashName(), []int{-i, -i, -i, -l}, []int{i, i, i, l}},
		}
	case i == j && i == l && i != k:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, i, k}, []int{i, i, i, k}},
			{-3, HashName(), []int{i, k}, []int{i, i, i, k}},
			{3, HashName(), []int{-i, k}, []int{i, i, i, k}},
			{-1, HashName(), []int{-i, -i, -i, k}, []int{i, i, i, k}},
			{-1, HashName(), []int{i, i, i, -k}, []int{i, i, i, k}},
			{3, HashName(), []int{i, -k}, []int{i, i, i, k}},
			{-3, HashName(), []int{-i, -k}, []int{i, i, i, k}},
			{1, HashName(), []int{-i, -i, -i, -k}, []int{i, i, i, k}},
		}
	case i == k && i == l && i != j:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, i, j}, []int{i, i, i, j}},
			{-3, HashName(), []int{i, j}, []int{i, i, i, j}},
			{3, HashName(), []int{-i, j}, []int{i, i, i, j}},
			{-1, HashName(), []int{-i, -i, -i, j}, []int{i, i, i, j}},
			{-1, HashName(), []int{i, i, i, -j}, []int{i, i, i, j}},
			{3, HashName(), []int{i, -j}, []int{i, i, i, j}},
			{-3, HashName(), []int{-i, -j}, []int{i, i, i, j}},
			{1, HashName(), []int{-i, -i, -i, -j}, []int{i, i, i, j}},
		}
	case j == k && j == l && j != i:
		return []ProtoCalc{
			{1, HashName(), []int{j, j, j, i}, []int{j, j, j, i}},
			{-3, HashName(), []int{j, i}, []int{j, j, j, i}},
			{3, HashName(), []int{-j, i}, []int{j, j, j, i}},
			{-1, HashName(), []int{-j, -j, -j, i}, []int{j, j, j, i}},
			{-1, HashName(), []int{j, j, j, -i}, []int{j, j, j, i}},
			{3, HashName(), []int{j, -i}, []int{j, j, j, i}},
			{-3, HashName(), []int{-j, -i}, []int{j, j, j, i}},
			{1, HashName(), []int{-j, -j, -j, -i}, []int{j, j, j, i}},
		}
	// 2 and 1 and 1
	case i == j && i != k && i != l && k != l:
		// x -> i, y -> k, z -> l
		return []ProtoCalc{
			{1, HashName(), []int{i, i, k, l}, []int{i, i, k, l}},
			{-2, HashName(), []int{k, l}, []int{i, i, k, l}},
			{1, HashName(), []int{-i, -i, k, l}, []int{i, i, k, l}},
			{-1, HashName(), []int{i, i, -k, l}, []int{i, i, k, l}},
			{2, HashName(), []int{-k, l}, []int{i, i, k, l}},
			{-1, HashName(), []int{-i, -i, -k, l}, []int{i, i, k, l}},
			{-1, HashName(), []int{i, i, k, -l}, []int{i, i, k, l}},
			{2, HashName(), []int{k, -l}, []int{i, i, k, l}},
			{-1, HashName(), []int{-i, -i, k, -l}, []int{i, i, k, l}},
			{1, HashName(), []int{i, i, -k, -l}, []int{i, i, k, l}},
			{-2, HashName(), []int{-k, -l}, []int{i, i, k, l}},
			{1, HashName(), []int{-i, -i, -k, -l}, []int{i, i, k, l}},
		}
	case i == k && i != j && i != l && j != l:
		// x -> i, y -> j, z -> l
		return []ProtoCalc{
			{1, HashName(), []int{i, i, j, l}, []int{i, i, j, l}},
			{-2, HashName(), []int{j, l}, []int{i, i, j, l}},
			{1, HashName(), []int{-i, -i, j, l}, []int{i, i, j, l}},
			{-1, HashName(), []int{i, i, -j, l}, []int{i, i, j, l}},
			{2, HashName(), []int{-j, l}, []int{i, i, j, l}},
			{-1, HashName(), []int{-i, -i, -j, l}, []int{i, i, j, l}},
			{-1, HashName(), []int{i, i, j, -l}, []int{i, i, j, l}},
			{2, HashName(), []int{j, -l}, []int{i, i, j, l}},
			{-1, HashName(), []int{-i, -i, j, -l}, []int{i, i, j, l}},
			{1, HashName(), []int{i, i, -j, -l}, []int{i, i, j, l}},
			{-2, HashName(), []int{-j, -l}, []int{i, i, j, l}},
			{1, HashName(), []int{-i, -i, -j, -l}, []int{i, i, j, l}},
		}
	case i == l && i != j && i != k && j != k:
		// x -> i, y -> k, z -> j
		return []ProtoCalc{
			{1, HashName(), []int{i, i, k, j}, []int{i, i, k, j}},
			{-2, HashName(), []int{k, j}, []int{i, i, k, j}},
			{1, HashName(), []int{-i, -i, k, j}, []int{i, i, k, j}},
			{-1, HashName(), []int{i, i, -k, j}, []int{i, i, k, j}},
			{2, HashName(), []int{-k, j}, []int{i, i, k, j}},
			{-1, HashName(), []int{-i, -i, -k, j}, []int{i, i, k, j}},
			{-1, HashName(), []int{i, i, k, -j}, []int{i, i, k, j}},
			{2, HashName(), []int{k, -j}, []int{i, i, k, j}},
			{-1, HashName(), []int{-i, -i, k, -j}, []int{i, i, k, j}},
			{1, HashName(), []int{i, i, -k, -j}, []int{i, i, k, j}},
			{-2, HashName(), []int{-k, -j}, []int{i, i, k, j}},
			{1, HashName(), []int{-i, -i, -k, -j}, []int{i, i, k, j}},
		}
	case j == k && j != i && j != l && i != l:
		// x -> j, y -> i, z -> l
		return []ProtoCalc{
			{1, HashName(), []int{j, j, i, l}, []int{j, j, i, l}},
			{-2, HashName(), []int{i, l}, []int{j, j, i, l}},
			{1, HashName(), []int{-j, -j, i, l}, []int{j, j, i, l}},
			{-1, HashName(), []int{j, j, -i, l}, []int{j, j, i, l}},
			{2, HashName(), []int{-i, l}, []int{j, j, i, l}},
			{-1, HashName(), []int{-j, -j, -i, l}, []int{j, j, i, l}},
			{-1, HashName(), []int{j, j, i, -l}, []int{j, j, i, l}},
			{2, HashName(), []int{i, -l}, []int{j, j, i, l}},
			{-1, HashName(), []int{-j, -j, i, -l}, []int{j, j, i, l}},
			{1, HashName(), []int{j, j, -i, -l}, []int{j, j, i, l}},
			{-2, HashName(), []int{-i, -l}, []int{j, j, i, l}},
			{1, HashName(), []int{-j, -j, -i, -l}, []int{j, j, i, l}},
		}
	case j == l && j != i && j != k && i != k:
		// x -> j, y -> i, z -> k
		return []ProtoCalc{
			{1, HashName(), []int{j, j, i, k}, []int{j, j, i, k}},
			{-2, HashName(), []int{i, k}, []int{j, j, i, k}},
			{1, HashName(), []int{-j, -j, i, k}, []int{j, j, i, k}},
			{-1, HashName(), []int{j, j, -i, k}, []int{j, j, i, k}},
			{2, HashName(), []int{-i, k}, []int{j, j, i, k}},
			{-1, HashName(), []int{-j, -j, -i, k}, []int{j, j, i, k}},
			{-1, HashName(), []int{j, j, i, -k}, []int{j, j, i, k}},
			{2, HashName(), []int{i, -k}, []int{j, j, i, k}},
			{-1, HashName(), []int{-j, -j, i, -k}, []int{j, j, i, k}},
			{1, HashName(), []int{j, j, -i, -k}, []int{j, j, i, k}},
			{-2, HashName(), []int{-i, -k}, []int{j, j, i, k}},
			{1, HashName(), []int{-j, -j, -i, -k}, []int{j, j, i, k}},
		}
	case k == l && k != i && k != j && i != j:
		// x -> k, y -> i, z -> j
		return []ProtoCalc{
			{1, HashName(), []int{k, k, i, j}, []int{k, k, i, j}},
			{-2, HashName(), []int{i, j}, []int{k, k, i, j}},
			{1, HashName(), []int{-k, -k, i, j}, []int{k, k, i, j}},
			{-1, HashName(), []int{k, k, -i, j}, []int{k, k, i, j}},
			{2, HashName(), []int{-i, j}, []int{k, k, i, j}},
			{-1, HashName(), []int{-k, -k, -i, j}, []int{k, k, i, j}},
			{-1, HashName(), []int{k, k, i, -j}, []int{k, k, i, j}},
			{2, HashName(), []int{i, -j}, []int{k, k, i, j}},
			{-1, HashName(), []int{-k, -k, i, -j}, []int{k, k, i, j}},
			{1, HashName(), []int{k, k, -i, -j}, []int{k, k, i, j}},
			{-2, HashName(), []int{-i, -j}, []int{k, k, i, j}},
			{1, HashName(), []int{-k, -k, -i, -j}, []int{k, k, i, j}},
		}
	// 2 and 2
	case i == j && k == l && i != k:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, k, k}, []int{i, i, k, k}},
			{1, HashName(), []int{-i, -i, -k, -k}, []int{i, i, k, k}},
			{1, HashName(), []int{-i, -i, k, k}, []int{i, i, k, k}},
			{1, HashName(), []int{i, i, -k, -k}, []int{i, i, k, k}},
			{-2, HashName(), []int{i, i}, []int{i, i, k, k}},
			{-2, HashName(), []int{k, k}, []int{i, i, k, k}},
			{-2, HashName(), []int{-i, -i}, []int{i, i, k, k}},
			{-2, HashName(), []int{-k, -k}, []int{i, i, k, k}},
			{4, "E0", []int{}, []int{i, i, k, k}},
		}
	case i == k && j == l && i != j:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, j, j}, []int{i, i, j, j}},
			{1, HashName(), []int{-i, -i, -j, -j}, []int{i, i, j, j}},
			{1, HashName(), []int{-i, -i, j, j}, []int{i, i, j, j}},
			{1, HashName(), []int{i, i, -j, -j}, []int{i, i, j, j}},
			{-2, HashName(), []int{i, i}, []int{i, i, j, j}},
			{-2, HashName(), []int{j, j}, []int{i, i, j, j}},
			{-2, HashName(), []int{-i, -i}, []int{i, i, j, j}},
			{-2, HashName(), []int{-j, -j}, []int{i, i, j, j}},
			{4, "E0", []int{}, []int{i, i, j, j}},
		}
	case i == l && j == k && i != j:
		return []ProtoCalc{
			{1, HashName(), []int{i, i, j, j}, []int{i, i, j, j}},
			{1, HashName(), []int{-i, -i, -j, -j}, []int{i, i, j, j}},
			{1, HashName(), []int{-i, -i, j, j}, []int{i, i, j, j}},
			{1, HashName(), []int{i, i, -j, -j}, []int{i, i, j, j}},
			{-2, HashName(), []int{i, i}, []int{i, i, j, j}},
			{-2, HashName(), []int{j, j}, []int{i, i, j, j}},
			{-2, HashName(), []int{-i, -i}, []int{i, i, j, j}},
			{-2, HashName(), []int{-j, -j}, []int{i, i, j, j}},
			{4, "E0", []int{}, []int{i, i, j, j}},
		}
	// all different
	case i != j && i != k && i != l && j != k && j != l && k != l:
		return []ProtoCalc{
			{1, HashName(), []int{i, j, k, l}, []int{i, j, k, l}},
			{-1, HashName(), []int{i, -j, k, l}, []int{i, j, k, l}},
			{-1, HashName(), []int{-i, j, k, l}, []int{i, j, k, l}},
			{1, HashName(), []int{-i, -j, k, l}, []int{i, j, k, l}},
			{-1, HashName(), []int{i, j, -k, l}, []int{i, j, k, l}},
			{1, HashName(), []int{i, -j, -k, l}, []int{i, j, k, l}},
			{1, HashName(), []int{-i, j, -k, l}, []int{i, j, k, l}},
			{-1, HashName(), []int{-i, -j, -k, l}, []int{i, j, k, l}},
			{-1, HashName(), []int{i, j, k, -l}, []int{i, j, k, l}},
			{1, HashName(), []int{i, -j, k, -l}, []int{i, j, k, l}},
			{1, HashName(), []int{-i, j, k, -l}, []int{i, j, k, l}},
			{-1, HashName(), []int{-i, -j, k, -l}, []int{i, j, k, l}},
			{1, HashName(), []int{i, j, -k, -l}, []int{i, j, k, l}},
			{-1, HashName(), []int{i, -j, -k, -l}, []int{i, j, k, l}},
			{-1, HashName(), []int{-i, j, -k, -l}, []int{i, j, k, l}},
			{1, HashName(), []int{-i, -j, -k, -l}, []int{i, j, k, l}},
		}
	default:
		panic("No cases matched")
	}
}
