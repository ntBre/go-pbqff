package main

import (
	_ "embed"
	"fmt"
)

//go:embed embed/taylor.py
var taylor string

// func Taylor(names []string, intder *Intder) {
// 	// Geometry has to match the coordinates in the template file
// 	fields := strings.Fields(intder.Geometry)
// 	coords := make([]float64, len(fields))
// 	for i, f := range fields {
// 		coords[i], _ = strconv.ParseFloat(f, 64)
// 	}
// 	// mol := symm.ReadXYZ(strings.NewReader(ZipXYZ(names, coords)))
// 	params := strings.Fields(strings.Split(intder.Head, "\n")[1])
// 	var (
// 		nsic    int
// 		nsicStr string
// 	)
// 	if params[2] == "0" {
// 		// accept number of simple internals if no SICs
// 		nsic, _ = strconv.Atoi(params[1])
// 		nsicStr = params[1]
// 	} else {
// 		nsic, _ = strconv.Atoi(params[2])
// 		nsicStr = params[2]
// 	}
// 	var str strings.Builder
// 	fmt.Fprintf(&str, "DISP%4d\n", nsic)
// 	for i := 0; i < nsic; i++ {
// 		fmt.Fprintf(&str, "%5d %18.10f\n%5d\n", i+1, 0.005, 0)
// 	}
// 	// These are the only fields needed by WritePts
// 	tmpder := &Intder{
// 		Head:     intder.Head,
// 		Geometry: intder.Geometry,
// 		Tail:     str.String(),
// 	}
// 	dir := os.TempDir()
// 	infile := filepath.Join(dir, "intder")
// 	tmpder.WritePts(infile + ".in")
// 	RunIntder(infile)
// 	// TODO parse file07 - extract the read file07 part from
// 	// Molpro.BuildPoints and reuse it here. see python and CL
// 	// implementations
// 	flags := ""
// 	// Example usage:
// 	// groups are b2, b1, a2, although b2 vs b1 shouldn't matter
// 	// python2 taylor.py 5 3 -m 2:[2-2,0-0,0-0] -q 1:[2-2,0-0,0-0]
// 	cmd := exec.Command("python2", "-c", taylor,
// 		// hard-code deriv=4 for now, giving 5
// 		"5", nsicStr, flags)
// 	cmd.Run()
// 	// symm.ReadXYZ(cartesian geometry) -> Molecule

// 	// actually need to take the geometry from the intder input
// 	// since it has to be in the right order relative to the
// 	// coordinates

// 	// then read intder.in, write one disp for each SIC, run
// 	// intder on it to get file07, call symm.Symmetry(Molecule) on
// 	// each of the resulting geometries to get the irreps for the
// 	// modes

// 	// sort the SICs if needed, probably skip this for now

// 	// run taylor.py with the input corresponding to the order -
// 	// should make a directory for this since it's going to be
// 	// messy

// 	// parse taylor.py output files to generate anpass and the
// 	// rest of the intder file. this also means I can actually use
// 	// delta and deltas keywords for SICs now
// }

// Ipow returns a^b
func Ipow(a, b int) int {
	prod := 1
	for i := b; i > 0; i-- {
		prod *= a
	}
	return prod
}

// Row returns the directly-derived Cartesian product row, where index
// is the desired row index, n is the truncation order and m is the
// number of variables in the Taylor series expansion. This
// corresponds to Algorithm 3 from Thackston18 with the meanings of n
// and m reversed to actually work
func Row(index, n, m int) []int {
	ret := make([]int, 0)
	for i := n - 1; i >= 0; i-- {
		ni := Ipow(m, i)
		di := index / ni
		ret = append(ret, di)
		index -= di * ni
	}
	return ret
}

// Sum over the elements of is
func Sum(is []int) int {
	var sum int
	for _, i := range is {
		sum += i
	}
	return sum
}

// NextRow takes an invalid row of the Cartesian product, the number
// of variables n, and the truncation order m and returns the index of
// the next valid row
func NextRow(row []int, n, m int) int {
	fmt.Println(row)
	for i := n - 1; i >= 0; i-- {
		if row[i] > 0 {
			row[i] = 0
			if i > 0 {
				row[i-1] += 1
			} else {
				break
			}
		}
	}
	fmt.Println(row)
	var index int
	for i := 0; i < n; i++ {
		index += row[i] * Ipow(m, i)
	}
	return index
}

// Disps generates the displacments corresponding to fcs
func Disps(fcs [][]int) (disps [][]int) {
	var idx int = 1
	for _, row := range fcs {
		indices := make([]int, 0)
		values := make([]int, 0)
		for i, digit := range row {
			if digit != 0 {
				indices = append(indices, i)
				values = append(values, digit)
			}
		}
		if len(values) == 0 {
			disps = append(disps, row)
			continue
		}
		prods := make([][]int, 0)
		for _, digit := range values {
			tmp := make([]int, 0)
			for j := -digit; j <= digit; j += 2 {
				tmp = append(tmp, j)
			}
			prods = append(prods, tmp)
		}
		newrows := CartProd(prods)
		for _, nrow := range newrows {
			r := make([]int, len(row))
			copy(r, row)
			for i, index := range indices {
				r[index] = nrow[i]
			}
			disps = append(disps, r)
			idx++
		}
	}
	return
}

// CartProd returns the Cartesian product of the elements in prods.
// Implementation adapted from
// https://docs.python.org/3/library/itertools.html#itertools.product
func CartProd(pools [][]int) [][]int {
	result := make([][]int, 1)
	for _, pool := range pools {
		tmp := make([][]int, 0)
		for _, x := range result {
			for _, y := range pool {
				tmp = append(tmp, append(x, y))
			}
		}
		result = tmp
	}
	return result
}

// Taylor computes the Taylor series expansion of order m-1 with n
// variables. See Thackston18 for details
func newTaylor(m, n int) (forces [][]int) {
	lastIndex := Ipow(m, n)
	var count int
	for i := 0; i < lastIndex; i++ {
		e := Row(i, n, m)
		s := Sum(e)
		if s < m {
			forces = append(forces, e)
			count++
		}
	}
	// TODO do the symmetry checks - mod and equivalence checks

	// TODO how to get disps? - the aptly named `displacements`
	// function generates them from the corresponding fc row
	return
}
