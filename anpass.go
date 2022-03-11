package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/ntBre/anpass"
)

// Anpass is a type for storing the information for an Anpass run
type Anpass struct {
	Head string
	Fmt1 string
	Fmt2 string
	Body string
	Tail string
}

// LoadAnpass reads a template anpass input file and stores the
// results in an Anpass
func LoadAnpass(filename string) (*Anpass, error) {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	var (
		a    Anpass
		buf  strings.Builder
		line string
		body bool
	)
	fstr := regexp.MustCompile(`(?i)\(\d+f([0-9.]+),f([0-9.]+)\)`)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line = scanner.Text()
		switch {
		case fstr.MatchString(line):
			fmt.Fprintf(&buf, "%s\n", line)
			a.Head = buf.String()
			buf.Reset()
			matches := fstr.FindStringSubmatch(line)
			a.Fmt1 = fmt.Sprintf("%%%sf", matches[1])
			a.Fmt2 = fmt.Sprintf("%%%sf", matches[2])
			body = true
		case strings.Contains(line, "UNKNOWNS"):
			a.Body = buf.String()
			buf.Reset()
			fmt.Fprintf(&buf, "%s\n", line)
			body = false
		case body:
			// TODO check length and don't trim if the
			// energies aren't there
			fields := strings.Fields(line)
			for _, f := range fields[:len(fields)-1] {
				v, _ := strconv.ParseFloat(f, 64)
				fmt.Fprintf(&buf, a.Fmt1, v)
			}
			fmt.Fprint(&buf, "\n")
		default:
			fmt.Fprintf(&buf, "%s\n", line)
		}
	}
	a.Tail = buf.String()
	return &a, nil
}

// FromIntder takes an intder file and constructs the corresponding
// Body of an Anpass, combining it with energies. If linear is true,
// duplicate the lines where last displacement is nonzero and negate
// the last displacement.
func FromIntder(filename string, energies []float64, linear bool) string {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(f)
	dispfmt := "%12.8f"
	var (
		line   string
		start  bool
		fields []string
		ncoord int
		str    strings.Builder
		nc     int = 1
		disp   int
		ret    strings.Builder
	)
	zero := regexp.MustCompile(`^    0$`)
	for i := 0; scanner.Scan(); i++ {
		line = scanner.Text()
		fields = strings.Fields(line)
		switch {
		case i == 1:
			// use number of simple internals if no SICs
			if fields[2] == "0" {
				ncoord, _ = strconv.Atoi(fields[1])
			} else {
				ncoord, _ = strconv.Atoi(fields[2])
			}
		case strings.Contains(line, "DISP"):
			start = true
		case start && zero.MatchString(line):
			for ; nc <= ncoord; nc++ {
				fmt.Fprintf(&str, dispfmt, 0.0)
			}
			fmt.Fprintf(&str, "%20.12f\n", energies[disp])
			if linear {
				fields := strings.Fields(str.String())
				d, _ := strconv.ParseFloat(fields[len(fields)-2], 64)
				if d != 0.0 {
					for i := 0; i < len(fields)-2; i++ {
						d, _ := strconv.ParseFloat(fields[i], 64)
						fmt.Fprintf(&str, dispfmt, d)
					}
					fmt.Fprintf(&str, dispfmt, -1*d)
					fmt.Fprintf(&str, "%20.12f\n", energies[disp])
				}
			}
			fmt.Fprint(&ret, str.String())
			nc = 1
			str.Reset()
			disp++
		case start && len(fields) >= 1:
			d, _ := strconv.Atoi(fields[0])
			for nc < d {
				fmt.Fprintf(&str, dispfmt, 0.0)
				nc++
			}
			nc++
			v, _ := strconv.ParseFloat(fields[1], 64)
			fmt.Fprintf(&str, dispfmt, v)
		}
	}
	return ret.String()
}

// BuildBody is a helper for building anpass file body
func (a *Anpass) BuildBody(buf *bytes.Buffer, energies []float64, intder *Intder) (lin bool) {
	body := strings.Split(strings.TrimSpace(a.Body), "\n")
	if len(body) > len(energies) {
		lin = true
		bodyLines := FromIntder(intder.Name, energies, true)
		buf.WriteString(bodyLines)
		return
	}
	for i, line := range body {
		if line != "" {
			for _, field := range strings.Fields(line) {
				f, _ := strconv.ParseFloat(field, 64)
				fmt.Fprintf(buf, a.Fmt1, f)
			}
			fmt.Fprintf(buf, a.Fmt2+"\n", energies[i])
		}
	}
	return
}

// WriteAnpass writes an anpass input file
func (a *Anpass) WriteAnpass(filename string, energies []float64, intder *Intder) (lin bool) {
	var buf bytes.Buffer
	buf.WriteString(a.Head)
	lin = a.BuildBody(&buf, energies, intder)
	buf.WriteString(a.Tail)
	os.WriteFile(filename, []byte(buf.String()), 0755)
	return
}

// WriteAnpass2 writes an anpass input file for a stationary point
func (a *Anpass) WriteAnpass2(filename, longLine string, energies []float64, intder *Intder) {
	var buf bytes.Buffer
	buf.WriteString(a.Head)
	a.BuildBody(&buf, energies, intder)
	for _, line := range strings.Split(a.Tail, "\n") {
		if strings.Contains(line, "END OF DATA") {
			buf.WriteString("STATIONARY POINT\n" +
				longLine + "\n")
		} else if strings.Contains(line, "!STATIONARY POINT") {
			continue
		}
		buf.WriteString(line + "\n")
	}
	os.WriteFile(filename, []byte(buf.String()), 0755)
}

// GetLongLine scans an anpass output file and return the "long line"
func GetLongLine(filename string) (string, bool) {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(f)
	var line, lastLine string
	for scanner.Scan() {
		line = scanner.Text()
		if strings.Contains(line, "0EIGENVALUE") {
			return lastLine, true
		} else if strings.Contains(line, "RESIDUALS") {
			fields := strings.Fields(line)
			if res, _ := strconv.
				ParseFloat(fields[len(fields)-1], 64); res > resBound {
				Warn("GetLongLine: sum of squared"+
					" residuals %e greater than %e", res, resBound)
			}
		}
		lastLine = line
	}
	return "", false
}

// DoAnpass runs anpass
func DoAnpass(anp *Anpass, dir string, energies []float64, intder *Intder) (
	string, bool) {
	lin := anp.WriteAnpass(filepath.Join(dir, "anpass1.in"),
		energies, intder)
	out, err := os.Create(filepath.Join(dir, "anpass1.out"))
	defer out.Close()
	if err != nil {
		panic(err)
	}
	// use the energies passed in instead of rereading them from the input
	disps, _, exps, biases, _ := anpass.ReadInput(
		filepath.Join(dir, "anpass1.in"),
	)
	anpass.PrintBias(out, biases)
	disps, energies = anpass.Bias(disps, energies, biases)
	longLine, _, _ := anpass.Run(out, dir, disps, energies, exps)
	infile2 := filepath.Join(dir, "anpass2.in")
	anpass.CopyAnpass(filepath.Join(dir, "anpass1.in"), infile2, longLine)
	outfile2 := strings.Replace(infile2, "in", "out", -1)
	out2, err := os.Create(outfile2)
	defer out2.Close()
	if err != nil {
		panic(err)
	}
	anpass.PrintBias(out2, longLine)
	disps, energies = anpass.Bias(disps, energies, longLine)
	anpass.Run(out2, dir, disps, energies, exps)
	if lin {
		Warn("linear molecule detected")
	}
	var str strings.Builder
	for _, f := range longLine {
		fmt.Fprintf(&str, "%20.12f", f)
	}
	return str.String(), lin
}

func Format9903(ncoords int, fcs []anpass.FC) {
	for _, fc := range fcs {
		i, j, k, l :=
			fc.Coord[0], fc.Coord[1],
			fc.Coord[2], fc.Coord[3]
		var (
			targ *[]CountFloat
			ids  []int
		)
		switch {
		case i == 0 || j == 0:
			continue
		case k == 0:
			targ = &fc2
			ids = Index(ncoords, false, i, j)
		case l == 0:
			targ = &fc3
			ids = Index(ncoords, false, i, j, k)
		default:
			targ = &fc4
			ids = Index(ncoords, false, i, j, k, l)
		}
		for _, id := range ids {
			(*targ)[id].Val = fc.Val
		}
	}
}
