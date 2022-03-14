// Thackston18 DOI: https://doi.org/10.1007/s10910-017-0783-3
package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strings"
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
// the next valid row. This corresponds to Algorithm 4 in Thackston18
func NextRow(row []int, n, m int) int {
	for i := n - 1; i >= 0; i-- {
		if row[i] > 0 {
			row[i] = 0
			if i > 0 {
				row[i-1] += 1
			}
			break
		}
	}
	var index int
	for i := n - 1; i >= 0; i-- {
		index += row[len(row)-i-1] * Ipow(m, i)
	}
	return index
}

func MakeDisps(w io.Writer, disps [][]int) {
	for _, row := range disps {
		lrow := len(row) - 1
		for i, d := range row {
			fmt.Fprintf(w, "%d", d)
			if i < lrow {
				fmt.Fprint(w, ",")
			}
		}
		fmt.Fprint(w, "\n")
	}
}

func WriteDisps(filename string, disps [][]int) {
	f, _ := os.Create(filename)
	defer f.Close()
	bf := bufio.NewWriter(f)
	defer bf.Flush()
	MakeDisps(bf, disps)
}

// Disps generates the displacments corresponding to fcs
func Disps(fcs [][]int, dups bool) (disps [][]int) {
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
			// fmt.Printf("# Displacement list: %d\n", r)
			disps = append(disps, r)
		}
	}
	if dups {
		return disps
	}
	return Deduplicate(disps)
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
				l := len(x)+1
				a := make([]int, l)
				copy(a, x)
				a[l-1] = y
				tmp = append(tmp, a)
			}
		}
		result = tmp
	}
	return result
}

// Deduplicate removes the duplicate rows produced by Taylor, like running `sort
// -u` on disp.txt
func Deduplicate(rows [][]int) (ret [][]int) {
	toKey := func(rs []int) string {
		var str strings.Builder
		for i, r := range rs {
			if i > 0 {
				fmt.Fprint(&str, "-")
			}
			fmt.Fprintf(&str, "%d", r)
		}
		return str.String()
	}
	m := make(map[string]bool)
	for _, row := range rows {
		key := toKey(row)
		if !m[key] {
			m[key] = true
			ret = append(ret, row)
		}
	}
	return
}

// ModCheck computes a mod check of one or more subsets of digits. I'm honestly
// not too sure what it means, but it does something in taylor.py. Also,
// taylor.py takes modchecks as a dict of {2: [][]int}, so I've omitted the
// variable k=2 and hard-coded it since that's all we usually use.
func ModCheck(row []int, modchecks [][]int) bool {
	var start int
	for _, check := range modchecks {
		start = check[0] - 1
		// if check[0] == 0 in Python, subtracting 1 gives the end of
		// the list and slicing from the end of the list gives an empty
		// list, which is acceptable for the mod check
		if start >= 0 && Sum(
			row[start:check[1]],
		)%2 != 0 {
			return false
		}
	}
	return true
}

// EqCheck computes an equivalence check of one or more subsets of digits. Not
// sure what this means either, but it does something in taylor.py. Like
// ModCheck, this takes a dict of {1: eqchecks} in the Python version, so I've
// ommitted the variable for the 1 since that's all we use.
func EqCheck(row []int, eqchecks [][]int) bool {
	var start int
	for _, check := range eqchecks {
		start = check[0] - 1
		// this time, it's not acceptable to have an empty list since
		// that doesn't have a sum of 1
		if start < 0 || Sum(
			row[start:check[1]],
		) != 1 {
			return false
		}
	}
	return true
}

// Taylor computes the Taylor series expansion of order m-1 with n
// variables. See Thackston18 for details
func newTaylor(m, n int, modchecks, eqchecks [][]int) (forces [][]int) {
	lastIndex := Ipow(m, n)
	var mc, ec bool
	for i := 0; i < lastIndex; {
		row := Row(i, n, m)
		s := Sum(row)
		if s < m {
			if modchecks != nil {
				mc = ModCheck(row, modchecks)
			} else {
				mc = true
			}
			if eqchecks != nil {
				ec = EqCheck(row, eqchecks)
			} else {
				ec = true
			}
			if (modchecks == nil && !ec) ||
				(eqchecks == nil && !mc) ||
				(!ec && !mc) {
				i++
				continue
			}
			forces = append(forces, row)
			i++
		} else {
			i = NextRow(row, n, m)
		}
	}
	// TODO do the symmetry checks - mod and equivalence checks.
	// test both without (already have) and with these checks
	return forces
}
