package main

// Make1D makes the Job slices for finite differences first
// derivative force constants
func Make1D(i int) []ProtoCalc {
	scale := ANGBOHR / (2 * Conf.Deltas[i-1])
	return []ProtoCalc{
		{Name: HashName(), Steps: []int{i}, Index: []int{i}, Coeff: 1, Scale: scale},
		{Name: HashName(), Steps: []int{-i}, Index: []int{i}, Coeff: -1, Scale: scale},
	}
}

// Make2D makes the Job slices for finite differences second
// derivative force constants
func Make2D(i, j int) []ProtoCalc {
	scale := ANGBOHR * ANGBOHR /
		(4 * Conf.Deltas[i-1] *
			Conf.Deltas[j-1])
	switch {
	case i == j:
		// E(+i+i) - 2*E(0) + E(-i-i) / (2d)^2
		return []ProtoCalc{
			{Name: HashName(), Steps: []int{i, i}, Index: []int{i, i}, Coeff: 1, Scale: scale},
			{Coeff: -2, Name: "E0", Steps: []int{}, Index: []int{i, i}, Scale: scale},
			{Name: HashName(), Steps: []int{-i, -i}, Index: []int{i, i}, Coeff: 1, Scale: scale},
		}
	case i != j:
		// E(+i+j) - E(+i-j) - E(-i+j) + E(-i-j) / (2d)^2
		return []ProtoCalc{
			{Name: HashName(), Steps: []int{i, j}, Index: []int{i, j}, Coeff: 1, Scale: scale},
			{Name: HashName(), Steps: []int{i, -j}, Index: []int{i, j}, Coeff: -1, Scale: scale},
			{Name: HashName(), Steps: []int{-i, j}, Index: []int{i, j}, Coeff: -1, Scale: scale},
			{Name: HashName(), Steps: []int{-i, -j}, Index: []int{i, j}, Coeff: 1, Scale: scale},
		}
	default:
		panic("No cases matched")
	}
}

// Make3D_2_1 handles the case where i == j and i != k
func Make3D_2_1(i, _, k int, scale float64) []ProtoCalc {
	// E(+i+i+k) - 2*E(+k) + E(-i-i+k) - E(+i+i-k) + 2*E(-k) - E(-i-i-k) / (2d)^3
	return []ProtoCalc{
		{Name: HashName(), Steps: []int{i, i, k}, Index: []int{i, i, k}, Coeff: 1, Scale: scale},
		{Name: HashName(), Steps: []int{k}, Index: []int{i, i, k}, Coeff: -2, Scale: scale},
		{Name: HashName(), Steps: []int{-i, -i, k}, Index: []int{i, i, k}, Coeff: 1, Scale: scale},
		{Name: HashName(), Steps: []int{i, i, -k}, Index: []int{i, i, k}, Coeff: -1, Scale: scale},
		{Name: HashName(), Steps: []int{-k}, Index: []int{i, i, k}, Coeff: 2, Scale: scale},
		{Name: HashName(), Steps: []int{-i, -i, -k}, Index: []int{i, i, k}, Coeff: -1, Scale: scale},
	}
}

// Make3D makes the ProtoCalc slices for finite differences third
// derivative force constants
func Make3D(i, j, k int) []ProtoCalc {
	scale := ANGBOHR * ANGBOHR * ANGBOHR /
		(8 * Conf.Deltas[i-1] *
			Conf.Deltas[j-1] *
			Conf.Deltas[k-1])
	switch {
	// all same
	case i == j && i == k:
		// E(+i+i+i) - 3*E(+i) + 3*E(-i) - E(-i-i-i) / (2d)^3
		return []ProtoCalc{
			{Name: HashName(), Steps: []int{i, i, i}, Index: []int{i, i, i}, Coeff: 1, Scale: scale},
			{Name: HashName(), Steps: []int{i}, Index: []int{i, i, i}, Coeff: -3, Scale: scale},
			{Name: HashName(), Steps: []int{-i}, Index: []int{i, i, i}, Coeff: 3, Scale: scale},
			{Name: HashName(), Steps: []int{-i, -i, -i}, Index: []int{i, i, i}, Coeff: -1, Scale: scale},
		}
	// 2 and 1
	case i == j && i != k:
		return Make3D_2_1(i, j, k, scale)
	case i == k && i != j:
		return Make3D_2_1(i, k, j, scale)
	case j == k && i != j:
		return Make3D_2_1(j, k, i, scale)
	// all different
	case i != j && i != k && j != k:
		return []ProtoCalc{
			{Name: HashName(), Steps: []int{i, j, k}, Index: []int{i, j, k}, Coeff: 1, Scale: scale},
			{Name: HashName(), Steps: []int{i, -j, k}, Index: []int{i, j, k}, Coeff: -1, Scale: scale},
			{Name: HashName(), Steps: []int{-i, j, k}, Index: []int{i, j, k}, Coeff: -1, Scale: scale},
			{Name: HashName(), Steps: []int{-i, -j, k}, Index: []int{i, j, k}, Coeff: 1, Scale: scale},
			{Name: HashName(), Steps: []int{i, j, -k}, Index: []int{i, j, k}, Coeff: -1, Scale: scale},
			{Name: HashName(), Steps: []int{i, -j, -k}, Index: []int{i, j, k}, Coeff: 1, Scale: scale},
			{Name: HashName(), Steps: []int{-i, j, -k}, Index: []int{i, j, k}, Coeff: 1, Scale: scale},
			{Name: HashName(), Steps: []int{-i, -j, -k}, Index: []int{i, j, k}, Coeff: -1, Scale: scale},
		}
	default:
		panic("No cases matched")
	}
}
