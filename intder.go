package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	geomFmt = "%17.9f%19.9f%19.9f"
	freqFmt = "%20.10f%20.10f%20.10f"
	strFmt  = "%17s%19s%19s"
)

var (
	ptable = map[string]string{
		"H": "1", "HE": "4", "LI": "7",
		"BE": "9", "B": "11", "C": "12",
		"N": "14", "O": "16", "F": "19",
		"NE": "20", "NA": "23", "MG": "24",
		"AL": "27", "SI": "28", "P": "31",
		"S": "32", "CL": "35", "AR": "40",
	}
)

// Intder holds the information for an intder input file
type Intder struct {
	Head     string
	Geometry string
	Tail     string
	Pattern  [][]int
	Dummies  []Dummy
}

// Dummy is a dummy atom in an intder input file
type Dummy struct {
	Coords  []float64 // x,y,z coords of dummy atom
	Matches []int     // what real coordinate they match
}

/*
Example for pattern matching/sorting
      0.000000000        2.391678166        0.000000000
     -2.274263181        0.000000000        0.000000000
      2.274263181        0.000000000        0.000000000
      0.000000000       -2.391678166        0.000000000
number of coordinates = 4
so score each position 1-4, 1 is most positive
4x3 array initialization
0 0 0
0 0 0
0 0 0
0 0 0
algorithm example
atom 1) 1 1 1 -> all highest to start
full array) 1 1 1
atom 2) 2 2 1 -> first two fields lower, ties okay if identical
full array) 1 1 1
            2 2 1
atom 3) 1 2 1 -> first position is new highest, need to increment above
full array) 2 1 1
            3 2 1
            1 2 1
atom 4) 2 3 1 -> ties in first and last, lower than 1 in 2
full array) 2 1 1
            3 2 1
            1 2 1
            2 3 1
=> highest x coordinate goes in 3rd row, lowest in 2nd row,
   tie in 1st and 4th broken by y coordinate

to convert back, create similar array from new cartesians
record necessary operations to make the two the same
then peform those operations on the rows of the cartesian
start with this
 Al        -1.2426875991        0.0000000000        0.0000000000
 Al         1.2426875991        0.0000000000        0.0000000000
 O          0.0000000000        1.3089084707        0.0000000000
 O          0.0000000000       -1.3089084707        0.0000000000
convert to
            3 2 1
            1 2 1
            2 1 1
            2 3 1
rearrange to match array above:
	    third row
	    first row
	    second row
	    fourth row
and apply the same ordering to the cartesian coordinates:
 O          0.0000000000        1.3089084707        0.0000000000
 Al        -1.2426875991        0.0000000000        0.0000000000
 Al         1.2426875991        0.0000000000        0.0000000000
 O          0.0000000000       -1.3089084707        0.0000000000
and done!
*/

// Pprint pretty prints a slice of slice of int
func Pprint(vals [][]int) {
	for _, line := range vals {
		fmt.Println(line)
	}
}

// Pattern extracts the pattern of coordinates from a string
// and returns the pattern, along with the required dummy atoms
//y|0 1 2
//x|-----
//0|0 1 2 [x][y] -> [3x+y]
//1|3 4 5
//2|6 7 8
func Pattern(geom string, ndummy int) ([][]int, []Dummy) {
	lines := CleanSplit(geom, "\n")
	pattern := make([][]int, 0, len(lines))
	floats := make([][]float64, 0, len(lines))
	dummies := make([]Dummy, 0, ndummy)
	var line string
	for i := 0; i < len(lines); i++ {
		line = lines[i]
		if i >= len(lines)-ndummy && line != "" {
			// in a dummy atom
			d := new(Dummy)
			// compare fields of dummy to those in floats
			for _, v := range strings.Fields(line) {
				match := false
				v, _ := strconv.ParseFloat(v, 64)
				d.Coords = append(d.Coords, v)
			loop:
				for x := range floats {
					for y := range floats[x] {
						if floats[x][y] == v {
							d.Matches = append(d.Matches, 3*x+y)
							match = true
							break loop
						}
					}
				}
				if !match {
					d.Matches = append(d.Matches, -1)
				}
			}
			dummies = append(dummies, *d)
			continue
		}
		if line != "" {
			pattern = append(pattern, make([]int, 3))
			floats = append(floats, make([]float64, 3))
		}
		for j, field := range strings.Fields(line) {
			f, _ := strconv.ParseFloat(field, 64)
			floats[i][j] = f
			pattern[i][j] = 1
			for k := 0; k < i; k++ {
				// floats[k][j] are the elements of the same column
				switch {
				case f < floats[k][j]:
					pattern[i][j]++
				case f > floats[k][j]:
					pattern[k][j]++
				}
			}
		}
	}
	return pattern, dummies
}

// LoadIntder loads an intder input file with the geometry lines
// stripped out
func LoadIntder(filename string) (*Intder, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var (
		buf    bytes.Buffer
		line   string
		i      Intder
		geom   string
		ndummy int
	)
	// end of symmetry internal coordinates
	eosic := regexp.MustCompile(`(?U)^\s+0\s*$`)
	head := true
	c := 0
	for scanner.Scan() {
		c++
		line = scanner.Text()
		if c == 2 {
			fields := strings.Fields(line)
			// IOPT(8) NDUM - intder manual pg 5
			ndummy, _ = strconv.Atoi(fields[7])
		}
		if head && eosic.MatchString(line) {
			fmt.Fprintln(&buf, line)
			i.Head = buf.String()
			head = false
			buf.Reset()
			continue
		}
		if strings.Contains(line, "DISP") {
			geom = buf.String()
			buf.Reset()
		}
		fmt.Fprintln(&buf, line)
	}
	i.Tail = buf.String()
	i.Geometry = geom[:len(geom)-1]
	i.Pattern, i.Dummies = Pattern(geom, ndummy)
	return &i, nil
}

// ConvertCart takes a cartesian geometry as a single string
// and formats it as needed by intder, saving the
// result in the passed in Intder. Return the ordered
// slice atom names
func (i *Intder) ConvertCart(cart string) (names []string) {
	lines := strings.Split(cart, "\n")
	// slice off last newline
	lines = lines[:len(lines)-1]
	var buf bytes.Buffer
	strs := make([]string, 0)
	for _, line := range lines {
		if len(line) > 3 {
			fields := strings.Fields(line)
			names = append(names, fields[0])
			x, _ := strconv.ParseFloat(fields[1], 64)
			y, _ := strconv.ParseFloat(fields[2], 64)
			z, _ := strconv.ParseFloat(fields[3], 64)
			str := fmt.Sprintf(geomFmt, x, y, z)
			fmt.Fprint(&buf, str+"\n")
			strs = append(strs, str)
		}
	}
	// remove last newline
	buf.Truncate(buf.Len() - 1)
	pattern, _ := Pattern(buf.String(), 0)
	swaps, order, ok := MatchPattern(i.Pattern, pattern)
	if !ok {
		panic("transform failed")
	}
	// swap columns
	strs = SwapStr(swaps, strs, strFmt)
	// swap rows and place in geometry
	i.Geometry = strings.Join(ApplyPattern(order, strs), "\n")
	// need to add dummy to geometry
	i.AddDummy(false)
	return ApplyPattern(order, names)
}

// AddDummy modifies i.Geometry in place to add dummy atoms
func (i *Intder) AddDummy(freqs bool) {
	var format string
	if freqs {
		format = freqFmt
	} else {
		format = geomFmt
	}
	lines := CleanSplit(i.Geometry, "\n")
	coords := make([]float64, 0, 3*len(lines))
	for line := range lines {
		fields := strings.Fields(lines[line])
		x, _ := strconv.ParseFloat(fields[0], 64)
		y, _ := strconv.ParseFloat(fields[1], 64)
		z, _ := strconv.ParseFloat(fields[2], 64)
		coords = append(coords, x, y, z)
	}
	for d := range i.Dummies {
		for c := range i.Dummies[d].Coords {
			if i.Dummies[d].Matches[c] != -1 {
				i.Dummies[d].Coords[c] = coords[i.Dummies[d].Matches[c]]
			}
		}
		i.Geometry += fmt.Sprintf("\n"+format, i.Dummies[d].Coords[0],
			i.Dummies[d].Coords[1], i.Dummies[d].Coords[2])
	}
}

// SwapStr exchanges the strings of strs based on the pattern defined in swaps
func SwapStr(swaps [][]int, strs []string, format string) []string {
	for i := range swaps {
		x := swaps[i][0]
		y := swaps[i][1]
		for s := range strs {
			fields := strings.Fields(strs[s])
			fields[x], fields[y] = fields[y], fields[x]
			strs[s] = fmt.Sprintf(format, fields[0], fields[1], fields[2])
		}
	}
	return strs
}

// MatchPattern takes source and destination patterns and returns
// the order of the source lines that will match that
// of the destination
func MatchPattern(dst, src [][]int) (swaps [][]int, order []int, ok bool) {
	for s := 0; s < 6; s++ {
		switch {
		// first time dont swap
		case s == 0:
			order = CheckPattern(dst, src)
			// when s is even, swap 0,1
		case s%2 == 0:
			// helper is the loop below
			// swap returns src with columns arg1 and arg2 swapped
			order = CheckPattern(dst, Swap(src, 0, 1))
			swaps = append(swaps, []int{0, 1})
			// when odd, swap 1,2
		default:
			order = CheckPattern(dst, Swap(src, 1, 2))
			swaps = append(swaps, []int{1, 2})
		}
		if len(order) == len(dst) {
			return swaps, order, true
		}
	}
	return nil, nil, false
}

// Swap columns i and j of src
func Swap(src [][]int, i, j int) [][]int {
	for x := range src {
		src[x][i], src[x][j] = src[x][j], src[x][i]
	}
	return src
}

// CheckPattern is a helper for MatchPattern, which checks
// if dst and src have the same pattern
func CheckPattern(dst, src [][]int) (order []int) {
	for i := 0; i < len(dst); i++ {
		for j := 0; j < len(src); j++ {
			if reflect.DeepEqual(dst[i], src[j]) {
				order = append(order, j)
			}
		}
	}
	return
}

// ApplyPattern applies an ordering to a slice of strings
func ApplyPattern(ord []int, lines []string) (ordered []string) {
	for i := range ord {
		ordered = append(ordered, lines[ord[i]])
	}
	return
}

// WritePts writes an intder.in file for points to filename
func (i *Intder) WritePts(filename string) {
	var buf bytes.Buffer
	buf.WriteString(i.Head + i.Geometry + "\n" + i.Tail)
	ioutil.WriteFile(filename, buf.Bytes(), 0755)
}

// WriteGeom writes an intder_geom.in file to filename, using
// longLine as the displacement
func (i *Intder) WriteGeom(filename, longLine string) {
	var buf bytes.Buffer
	buf.WriteString(i.Head + i.Geometry + "\n")
	fmt.Fprintf(&buf, "DISP%4d\n", 1)
	fields := strings.Fields(longLine)
	for i, val := range fields[:len(fields)-1] {
		val, _ := strconv.ParseFloat(val, 64)
		// skip values that are zero to the precision of the printing
		if math.Abs(val) > 1e-10 {
			fmt.Fprintf(&buf, "%5d%20.10f\n", i+1, val)
		}
	}
	fmt.Fprintf(&buf, "%5d\n", 0)
	ioutil.WriteFile(filename, buf.Bytes(), 0755)
}

// SecondLine updates the input directives of an intder
// for the cartesian coordinate transform in freqs
func (i *Intder) SecondLine() string {
	lines := strings.Split(i.Head, "\n")
	lines = lines[:len(lines)-1] // trim trailing newline
	fields := strings.Fields(lines[1])
	fields[3] = "4"
	fields[6] = "2"
	fields[10] = "3"
	fields[13] = "0"
	fields[14] = "0"
	var buf bytes.Buffer
	for _, field := range fields {
		fmt.Fprintf(&buf, "%5s", field)
	}
	lines[1] = buf.String()
	i.Head = strings.Join(lines, "\n")
	return lines[1]
}

// WriteFreqs writes an intder.in file for freqs to filename
// TODO might need updating for many atoms - multiline mass format?
func (i *Intder) WriteFreqs(filename string, names []string) {
	var buf bytes.Buffer
	i.SecondLine()
	buf.WriteString(i.Head + "\n" + i.Geometry + "\n")
	for i, name := range names {
		num, ok := ptable[strings.ToUpper(name)]
		if !ok {
			fmt.Errorf("error WriteFreqs: element %q not found in ptable", name)
		}
		switch i {
		case 0:
			fmt.Fprintf(&buf, "%11s", name+num)
		case 1:
			fmt.Fprintf(&buf, "%13s", name+num)
		default:
			fmt.Fprintf(&buf, "%12s", name+num)
		}
	}
	fmt.Fprint(&buf, "\n")
	buf.WriteString(i.Tail)
	ioutil.WriteFile(filename, buf.Bytes(), 0755)
}

// ReadGeom updates i.Geometry with the results of intder_geom
func (i *Intder) ReadGeom(filename string) string {
	const target = "NEW CARTESIAN GEOMETRY (BOHR)"
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var (
		line string
		geom bool
		buf  bytes.Buffer
	)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line = scanner.Text()
		if strings.Contains(line, target) {
			geom = true
			continue
		}
		if geom && line != "" {
			buf.WriteString(line + "\n")
		}
	}
	// skip last newline
	buf.Truncate(buf.Len() - 1)
	geometry := buf.String()
	i.Geometry = geometry
	i.AddDummy(true)
	return geometry
}

// ReadOut reads a freqs/intder.out and returns the harmonic
// frequencies found therein
func (i *Intder) ReadOut(filename string) (freqs []float64) {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	mode := regexp.MustCompile(`^\s+MODE`)
	var (
		line  string
		modes bool
	)
	for scanner.Scan() {
		line = scanner.Text()
		switch {
		case mode.MatchString(line):
			modes = true
		case modes:
			if strings.Contains(line, "NORMAL") {
				sort.Sort(sort.Reverse(sort.Float64Slice(freqs)))
				return
			}
			if line != "" {
				fields := strings.Fields(line)
				val, _ := strconv.ParseFloat(fields[1], 64)
				freqs = append(freqs, val)
			}
		}
	}
	return
}

// Read9903 reads fort.9903 output by anpass and
// sets i.Tail to that for freqs/intder
func (i *Intder) Read9903(filename string) {
	var (
		buf2 bytes.Buffer
		buf3 bytes.Buffer
		buf4 bytes.Buffer
	)
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var (
		line string
	)
	for scanner.Scan() {
		line = scanner.Text()
		fields := strings.Fields(line)
		if len(fields) == 5 {
			switch {
			case fields[3] != "0":
				fmt.Fprintln(&buf4, line)
			case fields[2] != "0":
				fmt.Fprintln(&buf3, line)
			case fields[1] != "0":
				fmt.Fprintln(&buf2, line)
			}
		}
	}
	fmt.Fprintf(&buf2, "%5d\n", 0)
	fmt.Fprintf(&buf3, "%5d\n", 0)
	fmt.Fprintf(&buf4, "%5d\n", 0)
	i.Tail = buf2.String() + buf3.String() + buf4.String()
}
