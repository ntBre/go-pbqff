package main

import (
	symm "github.com/ntBre/chemutils/symmetry"
)

// Make1D makes the Job slices for finite differences first
// derivative force constants
func Make1D(mol symm.Molecule, i int) []ProtoCalc {
	scale := angbohr / (2 * Conf.FlSlice(Deltas)[i-1])
	return []ProtoCalc{
		{Name: HashName(), Steps: []int{i}, Index: []int{i}, Coeff: 1, Scale: scale},
		{Name: HashName(), Steps: []int{-i}, Index: []int{i}, Coeff: -1, Scale: scale},
	}
}

// Make2D makes the Job slices for finite differences second
// derivative force constants
func Make2D(mol symm.Molecule, i, j int) []ProtoCalc {
	scale := angbohr * angbohr /
		(4 * Conf.FlSlice(Deltas)[i-1] *
			Conf.FlSlice(Deltas)[j-1])
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
func Make3D_2_1(i, _, k int, scale float64, mol symm.Molecule) []ProtoCalc {
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
func Make3D(mol symm.Molecule, i, j, k int) []ProtoCalc {
	scale := angbohr * angbohr * angbohr /
		(8 * Conf.FlSlice(Deltas)[i-1] *
			Conf.FlSlice(Deltas)[j-1] *
			Conf.FlSlice(Deltas)[k-1])
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
		return Make3D_2_1(i, j, k, scale, mol)
	case i == k && i != j:
		return Make3D_2_1(i, k, j, scale, mol)
	case j == k && i != j:
		return Make3D_2_1(j, k, i, scale, mol)
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

func Make4D_3_1(i, _, _, l int, scale float64, mol symm.Molecule) []ProtoCalc {
	return []ProtoCalc{
		{Name: HashName(), Steps: []int{i, i, i, l}, Index: []int{i, i, i, l}, Coeff: 1, Scale: scale},
		{Name: HashName(), Steps: []int{i, l}, Index: []int{i, i, i, l}, Coeff: -3, Scale: scale},
		{Name: HashName(), Steps: []int{-i, l}, Index: []int{i, i, i, l}, Coeff: 3, Scale: scale},
		{Name: HashName(), Steps: []int{-i, -i, -i, l}, Index: []int{i, i, i, l}, Coeff: -1, Scale: scale},
		{Name: HashName(), Steps: []int{i, i, i, -l}, Index: []int{i, i, i, l}, Coeff: -1, Scale: scale},
		{Name: HashName(), Steps: []int{i, -l}, Index: []int{i, i, i, l}, Coeff: 3, Scale: scale},
		{Name: HashName(), Steps: []int{-i, -l}, Index: []int{i, i, i, l}, Coeff: -3, Scale: scale},
		{Name: HashName(), Steps: []int{-i, -i, -i, -l}, Index: []int{i, i, i, l}, Coeff: 1, Scale: scale},
	}
}

func Make4D_2_1_1(i, _, k, l int, scale float64, mol symm.Molecule) []ProtoCalc {
	return []ProtoCalc{
		{Name: HashName(), Steps: []int{i, i, k, l}, Index: []int{i, i, k, l}, Coeff: 1, Scale: scale},
		{Name: HashName(), Steps: []int{k, l}, Index: []int{i, i, k, l}, Coeff: -2, Scale: scale},
		{Name: HashName(), Steps: []int{-i, -i, k, l}, Index: []int{i, i, k, l}, Coeff: 1, Scale: scale},
		{Name: HashName(), Steps: []int{i, i, -k, l}, Index: []int{i, i, k, l}, Coeff: -1, Scale: scale},
		{Name: HashName(), Steps: []int{-k, l}, Index: []int{i, i, k, l}, Coeff: 2, Scale: scale},
		{Name: HashName(), Steps: []int{-i, -i, -k, l}, Index: []int{i, i, k, l}, Coeff: -1, Scale: scale},
		{Name: HashName(), Steps: []int{i, i, k, -l}, Index: []int{i, i, k, l}, Coeff: -1, Scale: scale},
		{Name: HashName(), Steps: []int{k, -l}, Index: []int{i, i, k, l}, Coeff: 2, Scale: scale},
		{Name: HashName(), Steps: []int{-i, -i, k, -l}, Index: []int{i, i, k, l}, Coeff: -1, Scale: scale},
		{Name: HashName(), Steps: []int{i, i, -k, -l}, Index: []int{i, i, k, l}, Coeff: 1, Scale: scale},
		{Name: HashName(), Steps: []int{-k, -l}, Index: []int{i, i, k, l}, Coeff: -2, Scale: scale},
		{Name: HashName(), Steps: []int{-i, -i, -k, -l}, Index: []int{i, i, k, l}, Coeff: 1, Scale: scale},
	}
}

func Make4D_2_2(i, j, k, l int, scale float64, mol symm.Molecule) []ProtoCalc {
	return []ProtoCalc{
		{Name: HashName(), Steps: []int{i, i, k, k}, Index: []int{i, i, k, k}, Coeff: 1, Scale: scale},
		{Name: HashName(), Steps: []int{-i, -i, -k, -k}, Index: []int{i, i, k, k}, Coeff: 1, Scale: scale},
		{Name: HashName(), Steps: []int{-i, -i, k, k}, Index: []int{i, i, k, k}, Coeff: 1, Scale: scale},
		{Name: HashName(), Steps: []int{i, i, -k, -k}, Index: []int{i, i, k, k}, Coeff: 1, Scale: scale},
		{Name: HashName(), Steps: []int{i, i}, Index: []int{i, i, k, k}, Coeff: -2, Scale: scale},
		{Name: HashName(), Steps: []int{k, k}, Index: []int{i, i, k, k}, Coeff: -2, Scale: scale},
		{Name: HashName(), Steps: []int{-i, -i}, Index: []int{i, i, k, k}, Coeff: -2, Scale: scale},
		{Name: HashName(), Steps: []int{-k, -k}, Index: []int{i, i, k, k}, Coeff: -2, Scale: scale},
		{Coeff: 4, Name: "E0", Steps: []int{}, Index: []int{i, i, k, k}, Scale: scale},
	}
}

func Make4D_1_1_1_1(i, j, k, l int, scale float64) []ProtoCalc {
	return []ProtoCalc{
		{Name: HashName(), Steps: []int{i, j, k, l}, Index: []int{i, j, k, l}, Coeff: 2, Scale: scale},
		{Name: HashName(), Steps: []int{i, -j, -k, l}, Index: []int{i, j, k, l}, Coeff: 2, Scale: scale},
		{Name: HashName(), Steps: []int{i, -j, k, -l}, Index: []int{i, j, k, l}, Coeff: 2, Scale: scale},
		{Name: HashName(), Steps: []int{i, j, -k, -l}, Index: []int{i, j, k, l}, Coeff: 2, Scale: scale},
		{Name: HashName(), Steps: []int{i, -j, k, l}, Index: []int{i, j, k, l}, Coeff: -2, Scale: scale},
		{Name: HashName(), Steps: []int{i, j, -k, l}, Index: []int{i, j, k, l}, Coeff: -2, Scale: scale},
		{Name: HashName(), Steps: []int{i, j, k, -l}, Index: []int{i, j, k, l}, Coeff: -2, Scale: scale},
		{Name: HashName(), Steps: []int{i, -j, -k, -l}, Index: []int{i, j, k, l}, Coeff: -2, Scale: scale},
	}
}

// Make4D makes the ProtoCalc slices for finite differences fourth
// derivative force constants
func Make4D(mol symm.Molecule, i, j, k, l int) []ProtoCalc {
	scale := angbohr * angbohr * angbohr * angbohr /
		(16 * Conf.FlSlice(Deltas)[i-1] *
			Conf.FlSlice(Deltas)[j-1] *
			Conf.FlSlice(Deltas)[k-1] *
			Conf.FlSlice(Deltas)[l-1])
	switch {
	// all the same
	case i == j && i == k && i == l:
		return []ProtoCalc{
			{Name: HashName(), Steps: []int{i, i, i, i}, Index: []int{i, i, i, i}, Coeff: 1, Scale: scale},
			{Name: HashName(), Steps: []int{i, i}, Index: []int{i, i, i, i}, Coeff: -4, Scale: scale},
			{Coeff: 6, Name: "E0", Steps: []int{}, Index: []int{i, i, i, i}, Scale: scale},
			{Name: HashName(), Steps: []int{-i, -i}, Index: []int{i, i, i, i}, Coeff: -4, Scale: scale},
			{Name: HashName(), Steps: []int{-i, -i, -i, -i}, Index: []int{i, i, i, i}, Coeff: 1, Scale: scale},
		}

	// 3 and 1
	case i == j && i == k && i != l:
		return Make4D_3_1(i, j, k, l, scale, mol)
	case i == j && i == l && i != k:
		return Make4D_3_1(i, j, l, k, scale, mol)
	case i == k && i == l && i != j:
		return Make4D_3_1(i, k, l, j, scale, mol)
	case j == k && j == l && j != i:
		return Make4D_3_1(j, k, l, i, scale, mol)

	// 2 and 1 and 1
	case i == j && i != k && i != l && k != l:
		return Make4D_2_1_1(i, j, k, l, scale, mol)
	case i == k && i != j && i != l && j != l:
		return Make4D_2_1_1(i, k, j, l, scale, mol)
	case i == l && i != j && i != k && j != k:
		return Make4D_2_1_1(i, l, j, k, scale, mol)
	case j == k && j != i && j != l && i != l:
		return Make4D_2_1_1(j, k, i, l, scale, mol)
	case j == l && j != i && j != k && i != k:
		return Make4D_2_1_1(j, l, i, k, scale, mol)
	case k == l && k != i && k != j && i != j:
		return Make4D_2_1_1(k, l, i, j, scale, mol)

	// 2 and 2
	case i == j && k == l && i != k:
		return Make4D_2_2(i, j, k, l, scale, mol)
	case i == k && j == l && i != j:
		return Make4D_2_2(i, k, j, l, scale, mol)
	case i == l && j == k && i != j:
		return Make4D_2_2(i, l, j, k, scale, mol)

	// all different
	case i != j && i != k && i != l && j != k && j != l && k != l:
		return []ProtoCalc{
			{Name: HashName(), Steps: []int{i, j, k, l}, Index: []int{i, j, k, l}, Coeff: 1, Scale: scale},
			{Name: HashName(), Steps: []int{i, -j, k, l}, Index: []int{i, j, k, l}, Coeff: -1, Scale: scale},
			{Name: HashName(), Steps: []int{-i, j, k, l}, Index: []int{i, j, k, l}, Coeff: -1, Scale: scale},
			{Name: HashName(), Steps: []int{-i, -j, k, l}, Index: []int{i, j, k, l}, Coeff: 1, Scale: scale},
			{Name: HashName(), Steps: []int{i, j, -k, l}, Index: []int{i, j, k, l}, Coeff: -1, Scale: scale},
			{Name: HashName(), Steps: []int{i, -j, -k, l}, Index: []int{i, j, k, l}, Coeff: 1, Scale: scale},
			{Name: HashName(), Steps: []int{-i, j, -k, l}, Index: []int{i, j, k, l}, Coeff: 1, Scale: scale},
			{Name: HashName(), Steps: []int{-i, -j, -k, l}, Index: []int{i, j, k, l}, Coeff: -1, Scale: scale},
			{Name: HashName(), Steps: []int{i, j, k, -l}, Index: []int{i, j, k, l}, Coeff: -1, Scale: scale},
			{Name: HashName(), Steps: []int{i, -j, k, -l}, Index: []int{i, j, k, l}, Coeff: 1, Scale: scale},
			{Name: HashName(), Steps: []int{-i, j, k, -l}, Index: []int{i, j, k, l}, Coeff: 1, Scale: scale},
			{Name: HashName(), Steps: []int{-i, -j, k, -l}, Index: []int{i, j, k, l}, Coeff: -1, Scale: scale},
			{Name: HashName(), Steps: []int{i, j, -k, -l}, Index: []int{i, j, k, l}, Coeff: 1, Scale: scale},
			{Name: HashName(), Steps: []int{i, -j, -k, -l}, Index: []int{i, j, k, l}, Coeff: -1, Scale: scale},
			{Name: HashName(), Steps: []int{-i, j, -k, -l}, Index: []int{i, j, k, l}, Coeff: -1, Scale: scale},
			{Name: HashName(), Steps: []int{-i, -j, -k, -l}, Index: []int{i, j, k, l}, Coeff: 1, Scale: scale},
		}
	default:
		panic("No cases matched")
	}
}
