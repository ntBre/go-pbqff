package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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
	file, _ := ioutil.ReadFile(filename)
	lines := strings.Split(string(file), "\n")
	var (
		a          Anpass
		buf        bytes.Buffer
		body, tail bool
	)
	head := true
	for _, line := range lines {
		if head && string(line[0]) == "(" {
			head = false
			buf.WriteString(line + "\n")
			a.Head = buf.String()
			buf.Reset()
			// assume leading and trailing parentheses
			s := strings.Split(strings.ToUpper(line[1:len(line)-1]), "F")
			// assume trailing comma
			a.Fmt1 = "%" + string(s[1][:len(s[1])-1]) + "f"
			a.Fmt2 = "%" + string(s[2]) + "f"
			body = true
			continue
		}
		if body && strings.Contains(line, "UNKNOWNS") {
			body = false
			tail = true
		} else if body {
			f := strings.Fields(line)
			for i := 0; i < len(f)-1; i++ {
				val, _ := strconv.ParseFloat(f[i], 64)
				fmt.Fprintf(&buf, a.Fmt1, val)
			}
			buf.WriteString("\n")
			a.Body += buf.String()
			buf.Reset()
			continue
		}
		if tail {
			a.Tail += line + "\n"
		}
		buf.WriteString(line + "\n")
	}
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
		fmt.Fprintln(os.Stderr, "warning: linear molecule detected")
		lin = true
		bodyLines := FromIntder(intder.Name, energies, true)
		body = strings.Split(bodyLines, "\n")
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
	ioutil.WriteFile(filename, []byte(buf.String()), 0755)
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
	ioutil.WriteFile(filename, []byte(buf.String()), 0755)
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
				fmt.Fprintf(os.Stderr, "GetLongLine: warning: sum of squared"+
					" residuals %e greater than %e\n", res, resBound)
			}
		}
		lastLine = line
	}
	return "", false
}

// RunAnpass takes a filename like freqs/anpass1, runs anpass
// on freqs/anpass1.in and redirects the output into
// freqs/anpass1.out
func RunAnpass(filename string) {
	err := RunProgram(Conf.Str(AnpassCmd), filename)
	if err != nil {
		panic(err)
	}
}

// DoAnpass runs anpass
func DoAnpass(anp *Anpass, dir string, energies []float64, intder *Intder) (string, bool) {
	lin := anp.WriteAnpass(filepath.Join(dir, "freqs/anpass1.in"), energies, intder)
	RunAnpass(filepath.Join(dir, "freqs/anpass1"))
	longLine, ok := GetLongLine(filepath.Join(dir, "freqs/anpass1.out"))
	if !ok {
		panic("Problem getting long line from anpass1.out")
	}
	anp.WriteAnpass2(filepath.Join(dir, "freqs/anpass2.in"),
		longLine, energies, intder)
	RunAnpass(filepath.Join(dir, "freqs/anpass2"))
	return longLine, lin
}
