package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Spectro struct {
	Head   string
	Fermi1 string
	Fermi2 string
	Polyad string
	Coriol string
	Nfreqs int
}

// Load spectro input file, assumes no resonances included
func LoadSpectro(filename string) *Spectro {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var (
		buf  bytes.Buffer
		line string
	)
	for scanner.Scan() {
		line = scanner.Text()
		buf.WriteString(line + "\n")
	}
	return &Spectro{Head: buf.String()}
}

// Write a Spectro to an input file for use
func (s *Spectro) WriteInput(filename string) {
	var buf bytes.Buffer
	buf.WriteString(s.Head)
	buf.WriteString("# CORIOL #####\n")
	if s.Coriol != "" {
		buf.WriteString(s.Coriol)
	} else {
		fmt.Fprintf(&buf, "%5d\n", 0)
	}
	buf.WriteString("# FERMI1 ####\n")
	if s.Fermi1 != "" {
		buf.WriteString(s.Fermi1)
	} else {
		fmt.Fprintf(&buf, "%5d\n", 0)
	}
	buf.WriteString("# FERMI2 ####\n")
	if s.Fermi2 != "" {
		buf.WriteString(s.Fermi2)
	} else {
		fmt.Fprintf(&buf, "%5d\n", 0)
	}
	buf.WriteString("# RESIN ####\n")
	if s.Polyad != "" {
		buf.WriteString(s.Polyad)
	} else {
		fmt.Fprintf(&buf, "%5d\n", 0)
	}
	ioutil.WriteFile(filename, buf.Bytes(), 0755)
}

// Parse a Coriolis resonance from a spectro
// output line
func ParseCoriol(line string) string {
	letter := regexp.MustCompile(`A|B|C`)
	fields := strings.Fields(line)[1:3]
	// letters are only one character, so just take start index
	abcIndex := letter.FindStringIndex(fields[0])[0]
	abc := string(fields[0][abcIndex])
	switch {
	case abc == "A":
		abc = fmt.Sprintf("%5d%5d%5d", 1, 0, 0)
	case abc == "B":
		abc = fmt.Sprintf("%5d%5d%5d", 0, 1, 0)
	case abc == "C":
		abc = fmt.Sprintf("%5d%5d%5d", 0, 0, 1)
	}
	i := string(fields[0][:abcIndex])
	j := fields[1]
	return fmt.Sprintf("%5s%5s%s\n", i, j, abc)
}

func ParseFermi1(line string) string {
	fields := strings.Fields(line)[2:4]
	return fmt.Sprintf("%5s%5s\n", fields[0], fields[1])
}

func ParseFermi2(line string) string {
	fields := strings.Fields(line)[1:4]
	return fmt.Sprintf("%5s%5s%5s\n", fields[0], fields[1], fields[2])
}

// Read spectro output and prepare resonance fields
// for rerunning spectro
func (s *Spectro) ReadOutput(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var (
		// buf  bytes.Buffer
		line        string
		coriol      bool
		fermi1      bool
		fermi2      bool
		skip        int
		coriolCount int
		fermi1Count int
		fermi2Count int
	)
	for scanner.Scan() {
		line = scanner.Text()
		if skip > 0 {
			skip--
			continue
		}
		if coriol {
			if line == "" {
				coriol = false
			} else {
				coriolCount++
				s.Coriol += ParseCoriol(line)
			}
		}
		if fermi1 {
			if line == "" {
				fermi1 = false
			} else {
				fermi1Count++
				s.Fermi1 += ParseFermi1(line)
			}
		}
		if fermi2 {
			if line == "" {
				fermi2 = false
			} else {
				fermi2Count++
				s.Fermi2 += ParseFermi2(line)
			}
		}
		if strings.Contains(line, "CORIOLIS RESONANCES") {
			skip = 3
			coriol = true
		} else if strings.Contains(line, "FERMI RESONANCE") {
			fields := strings.Fields(line)
			if fields[3] == "1" {
				fermi1 = true
			} else {
				fermi2 = true
			}
			skip = 3
		}
	}
	// prepend the counts
	s.Coriol = fmt.Sprintf("%5d\n%5d\n", coriolCount, 0) + s.Coriol
	s.Fermi1 = fmt.Sprintf("%5d\n", fermi1Count) + s.Fermi1
	s.Fermi2 = fmt.Sprintf("%5d\n", fermi2Count) + s.Fermi2
	s.CheckPolyad()
}

// Check for Fermi Polyads and set the Polyad field
// as necessary
func (s *Spectro) CheckPolyad() {
	f1 := strings.Split(s.Fermi1, "\n")
	f2 := strings.Split(s.Fermi2, "\n")
	rhSet := make(map[int]bool)
	lhSet := make(map[string]bool)
	var poly bool
	// skip count line and empty last split
	for _, line := range f1[1 : len(f1)-1] {
		lhs, rhs := EqnSeparate(line)
		if !rhSet[rhs] {
			rhSet[rhs] = true
		}
		if !lhSet[MakeKey(lhs)] {
			lhSet[MakeKey(lhs)] = true
		}
	}
	for _, line := range f2[1 : len(f2)-1] {
		lhs, rhs := EqnSeparate(line)
		if rhSet[rhs] {
			poly = true
		} else {
			rhSet[rhs] = true
		}
		if !lhSet[MakeKey(lhs)] {
			lhSet[MakeKey(lhs)] = true
		}
	}
	if !poly {
		return
	}
	var (
		resin string
		count int
	)
	for k := range rhSet {
		resin += ResinLine(s.Nfreqs, 1, k)
		count++
	}
	for k := range lhSet {
		num := 1
		ints := make([]int, 0)
		for _, f := range strings.Fields(k) {
			i, _ := strconv.Atoi(f)
			ints = append(ints, i)
		}
		if len(ints) == 1 {
			num = 2
		}
		resin += ResinLine(s.Nfreqs, num, ints...)
		count++
	}
	s.Polyad = fmt.Sprintf("%5d\n%5d\n%s", 1, count, resin)
}

// Format a frequency number as a spectro RESIN line
func ResinLine(nfreqs, fill int, freqs ...int) string {
	var (
		buf   bytes.Buffer
		wrote bool
	)
	for j := 1; j <= nfreqs; j++ {
		for _, i := range freqs {
			if i == j {
				fmt.Fprintf(&buf, "%5d", fill)
				wrote = true
			}
		}
		if !wrote {
			fmt.Fprintf(&buf, "%5d", 0)
		}
		wrote = false
	}
	return buf.String() + "\n"
}

// Make a mappable key from a slice of ints
func MakeKey(ints []int) string {
	var buf bytes.Buffer
	for i, v := range ints {
		fmt.Fprintf(&buf, "%d", v)
		if i < len(ints)-1 {
			fmt.Fprint(&buf, " ")
		}
	}
	return buf.String()
}

// Separate the fields of a spectro Fermi resonance into
// a  left- and right-hand side
func EqnSeparate(line string) (lhs []int, rhs int) {
	fields := strings.Fields(line)
	last := len(fields) - 1
	ints := make([]int, last+1)
	for i, val := range fields {
		val, _ := strconv.Atoi(val)
		ints[i] = val
	}
	lhs = append(lhs, ints[:last]...)
	rhs = ints[last]
	return
}

// Gather harmonic, anharmonic, and resonance-corrected
// frequencies for reporting
func (s *Spectro) FreqReport(filename string) (zpt float64, harm, fund, corr []float64) {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var (
		line     string
		skip     int
		freqs    int
		harmFund bool
	)
	freq := regexp.MustCompile(`[0-9]+`)
	for scanner.Scan() {
		line = scanner.Text()
		switch {
		case skip > 0:
			skip--
		case strings.Contains(line, "BAND CENTER ANALYSIS"):
			skip += 3
			freqs = s.Nfreqs
			harmFund = true
		case harmFund && freqs > 0 && len(line) > 1:
			fields := strings.Fields(line)
			if freq.MatchString(fields[0]) {
				h, _ := strconv.ParseFloat(fields[1], 64)
				f, _ := strconv.ParseFloat(fields[2], 64)
				harm = append(harm, h)
				fund = append(fund, f)
				freqs--
			}
			if freqs == 0 {
				harmFund = false
			}
		case strings.Contains(line, "STATE NO."):
			skip += 2
			freqs = s.Nfreqs + 1 // add ZPT
		case !harmFund && freqs > 0 && len(line) > 1:
			fields := strings.Fields(line)
			if freq.MatchString(fields[0]) {
				state, _ := strconv.Atoi(fields[0])
				if state == 1 {
					zpt, _ = strconv.ParseFloat(fields[1], 64)
					freqs--
				} else if state <= s.Nfreqs+1 {
					f, _ := strconv.ParseFloat(fields[2], 64)
					corr = append(corr, f)
					freqs--
				}
			}
		}
	}
	// put in decreasing order
	sort.Sort(sort.Reverse(sort.Float64Slice(harm)))
	sort.Sort(sort.Reverse(sort.Float64Slice(fund)))
	sort.Sort(sort.Reverse(sort.Float64Slice(corr)))
	return
}
