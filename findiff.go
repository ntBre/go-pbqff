package main

import (
	symm "github.com/ntBre/chemutils/symmetry"
)

var (
	None = ProtoCalc{0, "E0", []int{}, []int{}, 1}
)

// OOP returns whether the coordinate i is out of the plane defined by
// mol.Main
func OOP(i int, mol symm.Molecule) bool {
	if *nosym {
		return false
	}
	// coordinate index of the ith coordinate
	// i%3 = 0 => X
	// i%3 = 1 => Y
	// i%3 = 2 => Z
	ix := IntAbs(i) - 1
	if len(mol.Planes) < 1 {
		return false
	}
	// axis out of the main plane
	ax := int(mol.Planes[0].Not())
	if ix%3 == ax {
		return true
	}
	return false
}

// Make1D makes the Job slices for finite differences first
// derivative force constants
func Make1D(mol symm.Molecule, i int) []ProtoCalc {
	scale := angbohr / (2 * Conf.FlSlice(Deltas)[i-1])
	if OOP(i, mol) {
		return []ProtoCalc{
			{0, "E0", []int{i}, []int{}, scale},
		}
	}
	return []ProtoCalc{
		{1, HashName(), []int{i}, []int{i}, scale},
		{-1, HashName(), []int{-i}, []int{i}, scale},
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
		if OOP(i, mol) {
			return []ProtoCalc{
				{2, HashName(), []int{i, i}, []int{i, i}, scale},
				{-2, "E0", []int{}, []int{i, i}, scale},
			}
		}
		return []ProtoCalc{
			{1, HashName(), []int{i, i}, []int{i, i}, scale},
			{-2, "E0", []int{}, []int{i, i}, scale},
			{1, HashName(), []int{-i, -i}, []int{i, i}, scale},
		}
	case i != j:
		// E(+i+j) - E(+i-j) - E(-i+j) + E(-i-j) / (2d)^2
		if OOP(i, mol) && OOP(j, mol) {
			return []ProtoCalc{
				{2, HashName(), []int{i, j}, []int{i, j}, scale},
				{-2, HashName(), []int{i, -j}, []int{i, j}, scale},
			}
		} else if OOP(i, mol) || OOP(j, mol) {
			return []ProtoCalc{
				{0, "E0", []int{i, j}, []int{}, scale},
			}
		}
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

// Make3D_2_1 handles the case where i == j and i != k
func Make3D_2_1(i, j, k int, scale float64, mol symm.Molecule) []ProtoCalc {
	// E(+i+i+k) - 2*E(+k) + E(-i-i+k) - E(+i+i-k) + 2*E(-k) - E(-i-i-k) / (2d)^3
	switch {
	case OOP(k, mol):
		return []ProtoCalc{
			None,
		}
	case OOP(i, mol):
		return []ProtoCalc{
			{2, HashName(), []int{i, i, k}, []int{i, i, k}, scale},
			{-2, HashName(), []int{k}, []int{i, i, k}, scale},
			{-2, HashName(), []int{i, i, -k}, []int{i, i, k}, scale},
			{2, HashName(), []int{-k}, []int{i, i, k}, scale},
		}
	}
	return []ProtoCalc{
		{1, HashName(), []int{i, i, k}, []int{i, i, k}, scale},
		{-2, HashName(), []int{k}, []int{i, i, k}, scale},
		{1, HashName(), []int{-i, -i, k}, []int{i, i, k}, scale},
		{-1, HashName(), []int{i, i, -k}, []int{i, i, k}, scale},
		{2, HashName(), []int{-k}, []int{i, i, k}, scale},
		{-1, HashName(), []int{-i, -i, -k}, []int{i, i, k}, scale},
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
	case i == j && i == k:
		// E(+i+i+i) - 3*E(+i) + 3*E(-i) - E(-i-i-i) / (2d)^3
		if OOP(i, mol) {
			return []ProtoCalc{
				None,
			}
		}
		return []ProtoCalc{
			{1, HashName(), []int{i, i, i}, []int{i, i, i}, scale},
			{-3, HashName(), []int{i}, []int{i, i, i}, scale},
			{3, HashName(), []int{-i}, []int{i, i, i}, scale},
			{-1, HashName(), []int{-i, -i, -i}, []int{i, i, i}, scale},
		}
	case i == j && i != k:
		return Make3D_2_1(i, j, k, scale, mol)
	case i == k && i != j:
		return Make3D_2_1(i, k, j, scale, mol)
	case j == k && i != j:
		return Make3D_2_1(j, k, i, scale, mol)
	case i != j && i != k && j != k:
		switch {
		case OOP(i, mol) && OOP(j, mol) && OOP(k, mol):
			return []ProtoCalc{
				None,
			}
		case OOP(i, mol) && OOP(j, mol):
			fallthrough
		case OOP(i, mol) && OOP(k, mol):
			return []ProtoCalc{
				{2, HashName(), []int{i, j, k}, []int{i, j, k}, scale},
				{-2, HashName(), []int{i, -j, k}, []int{i, j, k}, scale},
				{-2, HashName(), []int{i, j, -k}, []int{i, j, k}, scale},
				{2, HashName(), []int{i, -j, -k}, []int{i, j, k}, scale},
			}
		case OOP(j, mol) && OOP(k, mol):
			return []ProtoCalc{
				{2, HashName(), []int{i, j, k}, []int{i, j, k}, scale},
				{-2, HashName(), []int{i, -j, k}, []int{i, j, k}, scale},
				{2, HashName(), []int{-i, -j, k}, []int{i, j, k}, scale},
				{-2, HashName(), []int{-i, j, k}, []int{i, j, k}, scale},
			}
		case OOP(i, mol):
			fallthrough
		case OOP(j, mol):
			fallthrough
		case OOP(k, mol):
			return []ProtoCalc{
				None,
			}
		}
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

func Make4D_3_1(i, j, k, l int, scale float64, mol symm.Molecule) []ProtoCalc {
	switch {
	case OOP(i, mol) && OOP(l, mol):
		return []ProtoCalc{
			{2, HashName(), []int{i, i, i, l}, []int{i, i, i, l}, scale},
			{-6, HashName(), []int{i, l}, []int{i, i, i, l}, scale},
			{-2, HashName(), []int{-i, -i, -i, l}, []int{i, i, i, l}, scale},
			{6, HashName(), []int{i, -l}, []int{i, i, i, l}, scale},
		}
	case OOP(i, mol):
		fallthrough
	case OOP(l, mol):
		return []ProtoCalc{
			None,
		}
	}
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
}

func Make4D_2_1_1(i, j, k, l int, scale float64, mol symm.Molecule) []ProtoCalc {
	switch {
	case OOP(i, mol) && OOP(l, mol) && OOP(k, mol):
		return []ProtoCalc{
			{2, HashName(), []int{i, i, k, l}, []int{i, i, k, l}, scale},
			{-4, HashName(), []int{k, l}, []int{i, i, k, l}, scale},
			{2, HashName(), []int{-i, -i, k, l}, []int{i, i, k, l}, scale},
			{-2, HashName(), []int{-i, -i, -k, l}, []int{i, i, k, l}, scale},
			{-2, HashName(), []int{i, i, -k, l}, []int{i, i, k, l}, scale},
			{4, HashName(), []int{k, -l}, []int{i, i, k, l}, scale},
		}
	case OOP(l, mol) && OOP(k, mol):
		return []ProtoCalc{
			{2, HashName(), []int{i, i, k, l}, []int{i, i, k, l}, scale},
			{-4, HashName(), []int{k, l}, []int{i, i, k, l}, scale},
			{2, HashName(), []int{-i, -i, k, l}, []int{i, i, k, l}, scale},
			{-2, HashName(), []int{i, i, -k, l}, []int{i, i, k, l}, scale},
			{4, HashName(), []int{-k, l}, []int{i, i, k, l}, scale},
			{-2, HashName(), []int{-i, -i, k, -l}, []int{i, i, k, l}, scale},
		}
	case OOP(l, mol) || OOP(k, mol):
		return []ProtoCalc{
			None,
		}
	case OOP(i, mol):
		return []ProtoCalc{
			{2, HashName(), []int{i, i, k, l}, []int{i, i, k, l}, scale},
			{-2, HashName(), []int{i, i, -k, l}, []int{i, i, k, l}, scale},
			{-2, HashName(), []int{i, i, k, -l}, []int{i, i, k, l}, scale},
			{2, HashName(), []int{i, i, -k, -l}, []int{i, i, k, l}, scale},
			{-2, HashName(), []int{k, l}, []int{i, i, k, l}, scale},
			{2, HashName(), []int{-k, l}, []int{i, i, k, l}, scale},
			{2, HashName(), []int{k, -l}, []int{i, i, k, l}, scale},
			{-2, HashName(), []int{-k, -l}, []int{i, i, k, l}, scale},
		}
	}
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
}

func Make4D_2_2(i, j, k, l int, scale float64, mol symm.Molecule) []ProtoCalc {
	switch {
	case OOP(i, mol) && OOP(k, mol):
		return []ProtoCalc{
			{2, HashName(), []int{i, i, k, k}, []int{i, i, k, k}, scale},
			{2, HashName(), []int{-i, -i, k, k}, []int{i, i, k, k}, scale},
			{-4, HashName(), []int{i, i}, []int{i, i, k, k}, scale},
			{-4, HashName(), []int{k, k}, []int{i, i, k, k}, scale},
			{4, "E0", []int{}, []int{i, i, k, k}, scale},
		}
	case OOP(i, mol):
		return []ProtoCalc{
			{2, HashName(), []int{i, i, k, k}, []int{i, i, k, k}, scale},
			{2, HashName(), []int{i, i, -k, -k}, []int{i, i, k, k}, scale},
			{-4, HashName(), []int{i, i}, []int{i, i, k, k}, scale},
			{-2, HashName(), []int{k, k}, []int{i, i, k, k}, scale},
			{-2, HashName(), []int{-k, -k}, []int{i, i, k, k}, scale},
			{4, "E0", []int{}, []int{i, i, k, k}, scale},
		}
	case OOP(k, mol):
		return []ProtoCalc{
			{2, HashName(), []int{i, i, k, k}, []int{i, i, k, k}, scale},
			{2, HashName(), []int{-i, -i, k, k}, []int{i, i, k, k}, scale},
			{-4, HashName(), []int{k, k}, []int{i, i, k, k}, scale},
			{-2, HashName(), []int{i, i}, []int{i, i, k, k}, scale},
			{-2, HashName(), []int{-i, -i}, []int{i, i, k, k}, scale},
			{4, "E0", []int{}, []int{i, i, k, k}, scale},
		}
	}
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
}

func Make4D_1_1_1_1(i, j, k, l int, scale float64) []ProtoCalc {
	return []ProtoCalc{
		{2, HashName(), []int{i, j, k, l}, []int{i, j, k, l}, scale},
		{2, HashName(), []int{i, -j, -k, l}, []int{i, j, k, l}, scale},
		{2, HashName(), []int{i, -j, k, -l}, []int{i, j, k, l}, scale},
		{2, HashName(), []int{i, j, -k, -l}, []int{i, j, k, l}, scale},
		{-2, HashName(), []int{i, -j, k, l}, []int{i, j, k, l}, scale},
		{-2, HashName(), []int{i, j, -k, l}, []int{i, j, k, l}, scale},
		{-2, HashName(), []int{i, j, k, -l}, []int{i, j, k, l}, scale},
		{-2, HashName(), []int{i, -j, -k, -l}, []int{i, j, k, l}, scale},
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
		if OOP(i, mol) {
			return []ProtoCalc{
				{2, HashName(), []int{i, i, i, i}, []int{i, i, i, i}, scale},
				{-8, HashName(), []int{i, i}, []int{i, i, i, i}, scale},
				{6, "E0", []int{}, []int{i, i, i, i}, scale},
			}
		}
		return []ProtoCalc{
			{1, HashName(), []int{i, i, i, i}, []int{i, i, i, i}, scale},
			{-4, HashName(), []int{i, i}, []int{i, i, i, i}, scale},
			{6, "E0", []int{}, []int{i, i, i, i}, scale},
			{-4, HashName(), []int{-i, -i}, []int{i, i, i, i}, scale},
			{1, HashName(), []int{-i, -i, -i, -i}, []int{i, i, i, i}, scale},
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
		switch {
		// all OOP
		case OOP(i, mol) && OOP(j, mol) && OOP(k, mol) && OOP(l, mol):
			return []ProtoCalc{
				{2, HashName(), []int{i, j, k, l}, []int{i, j, k, l}, scale},
				{-2, HashName(), []int{i, -j, k, l}, []int{i, j, k, l}, scale},
				{-2, HashName(), []int{-i, j, k, l}, []int{i, j, k, l}, scale},
				{2, HashName(), []int{-i, -j, k, l}, []int{i, j, k, l}, scale},
				{2, HashName(), []int{-i, j, -k, l}, []int{i, j, k, l}, scale},
				{2, HashName(), []int{i, -j, -k, l}, []int{i, j, k, l}, scale},
				{-2, HashName(), []int{-i, -j, -k, l}, []int{i, j, k, l}, scale},
				{-2, HashName(), []int{i, j, -k, l}, []int{i, j, k, l}, scale},
			}

		// 3 OOP
		case OOP(i, mol) && OOP(j, mol) && OOP(k, mol):
			fallthrough
		case OOP(i, mol) && OOP(j, mol) && OOP(l, mol):
			fallthrough
		case OOP(i, mol) && OOP(k, mol) && OOP(l, mol):
			fallthrough
		case OOP(j, mol) && OOP(k, mol) && OOP(l, mol):
			return []ProtoCalc{
				None,
			}

		// 2 OOP
		case OOP(i, mol) && OOP(j, mol):
			return Make4D_1_1_1_1(i, j, k, l, scale)
		case OOP(i, mol) && OOP(k, mol):
			return Make4D_1_1_1_1(i, k, j, l, scale)
		case OOP(i, mol) && OOP(l, mol):
			return Make4D_1_1_1_1(i, l, j, k, scale)
		case OOP(j, mol) && OOP(k, mol):
			return Make4D_1_1_1_1(j, k, i, l, scale)
		case OOP(j, mol) && OOP(l, mol):
			return Make4D_1_1_1_1(j, l, i, k, scale)
		case OOP(k, mol) && OOP(l, mol):
			return Make4D_1_1_1_1(k, l, i, j, scale)

		// 1 OOP
		case OOP(i, mol) || OOP(j, mol) || OOP(k, mol) || OOP(l, mol):
			return []ProtoCalc{
				None,
			}
		}
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
