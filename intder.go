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
	"strconv"
	"strings"
)

type Intder struct {
	Head     string
	Geometry string
	Tail     string
	Pattern  [][]int
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

// pretty print a slice of slice of int
func Pprint(vals [][]int) {
	for _, line := range vals {
		fmt.Println(line)
	}
}

func Pattern(geom string) [][]int {
	lines := strings.Split(geom, "\n")
	pattern := make([][]int, 0)
	floats := make([][]float64, 0)
	var line string
	for i := range lines {
		line = lines[i]
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
	return pattern
}

// Loads an intder input file with the geometry lines
// stripped out
func LoadIntder(filename string) *Intder {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var (
		buf  bytes.Buffer
		line string
		i    Intder
		geom string
	)
	// end of symmetry internal coordinates
	eosic := regexp.MustCompile(`(?U)^\s+0\s*$`)
	head := true
	for scanner.Scan() {
		line = scanner.Text()
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
	i.Pattern = Pattern(geom)
	return &i
}

// Takes a cartesian geometry as a single string
// and formats it as needed by intder, saving the
// result in the passed in Intder. Return the ordered
// slice atom names
func (i *Intder) ConvertCart(cart string) (order []string) {
	lines := strings.Split(cart, "\n")
	// slice off last newline
	lines = lines[:len(lines)-1]
	var buf bytes.Buffer
	floats := make([][]float64, 0)
	strs := make([]string, 0)
	for i, line := range lines {
		if len(line) > 3 {
			floats = append(floats, make([]float64, 3))
			fields := strings.Fields(line)
			order = append(order, fields[0])
			x, _ := strconv.ParseFloat(fields[1], 64)
			floats[i][0] = x
			y, _ := strconv.ParseFloat(fields[2], 64)
			floats[i][1] = y
			z, _ := strconv.ParseFloat(fields[3], 64)
			floats[i][2] = z
			str := fmt.Sprintf("%17.9f%19.9f%19.9f", x, y, z)
			fmt.Fprint(&buf, str+"\n")
			strs = append(strs, str)
		}
	}
	// remove last newline
	buf.Truncate(buf.Len() - 1)
	pattern := Pattern(buf.String())
	transform, ok := MatchPattern(i.Pattern, pattern)
	if !ok {
		panic("transform failed")
	}
	i.Geometry = strings.Join(ApplyPattern(transform, strs), "\n")
	return ApplyPattern(transform, order)
}

// Take source and destination patterns and return
// the order of the source lines that will match that
// of the destination
func MatchPattern(dst, src [][]int) (order []int, ok bool) {
	for i := 0; i < len(dst); i++ {
		for j := 0; j < len(src); j++ {
			if reflect.DeepEqual(dst[i], src[j]) {
				order = append(order, j)
			}
		}
	}
	if len(order) == len(dst) {
		ok = true
	}
	return
}

// Because of the lack of generics, only order slices
// of strings based on the ordering ord :(
func ApplyPattern(ord []int, lines []string) (ordered []string) {
	for i := range ord {
		ordered = append(ordered, lines[ord[i]])
	}
	return
}

// Write an intder.in file for points to filename
func (i *Intder) WritePts(filename string) {
	var buf bytes.Buffer
	buf.WriteString(i.Head + i.Geometry + "\n" + i.Tail)
	ioutil.WriteFile(filename, buf.Bytes(), 0755)
}

// Write an intder_geom.in file to filename, using
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

// Write an intder.in file for freqs to filename
// TODO might need updating for many atoms - multiline mass format?
func (i *Intder) WriteFreqs(filename string, names []string) {
	var buf bytes.Buffer
	buf.WriteString(i.Head + i.Geometry + "\n")
	for i, name := range names {
		num, ok := ptable[name]
		if !ok {
			fmt.Errorf("WriteFreqs: element %q not found in ptable\n", name)
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

// Update i.Geometry with the results of intder_geom
func (i *Intder) ReadGeom(filename string) {
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
	i.Geometry = buf.String()
}

// Read a freqs/intder.out and return the harmonic
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

// Set i.Tail to ouput from anpass for freqs/intder
func (i *Intder) Read9903(filename string) {
	var buf bytes.Buffer
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var (
		line          string
		third, fourth bool
	)
	for scanner.Scan() {
		line = scanner.Text()
		fields := strings.Fields(line)
		if len(fields) == 5 {
			if !third && fields[2] != "0" {
				fmt.Fprintf(&buf, "%5d\n", 0)
				third = true
			}
			if !fourth && fields[3] != "0" {
				fmt.Fprintf(&buf, "%5d\n", 0)
				fourth = true
			}
			if fields[0] != "0" && fields[1] != "0" {
				fmt.Fprintln(&buf, line)
			}
		}
	}
	fmt.Fprintf(&buf, "%5d\n", 0)
	i.Tail = buf.String()
}
