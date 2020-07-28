package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

type Anpass struct {
	Head string
	Fmt1 string
	Fmt2 string
	Body string
	Tail string
}

// Helper for building anpass file body
func (a *Anpass) BuildBody(buf *bytes.Buffer, energies []float64) {
	for i, line := range strings.Split(a.Body, "\n") {
		if line != "" {
			for _, field := range strings.Fields(line) {
				f, _ := strconv.ParseFloat(field, 64)
				fmt.Fprintf(buf, a.Fmt1, f)
			}
			fmt.Fprintf(buf, a.Fmt2+"\n", energies[i])
		}
	}
}

func (a *Anpass) WriteAnpass(filename string, energies []float64) {
	var buf bytes.Buffer
	buf.WriteString(a.Head)
	a.BuildBody(&buf, energies)
	buf.WriteString(a.Tail)
	ioutil.WriteFile(filename, []byte(buf.String()), 0755)
}

func (a *Anpass) WriteAnpass2(filename, longLine string, energies []float64) {
	var buf bytes.Buffer
	buf.WriteString(a.Head)
	a.BuildBody(&buf, energies)
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

// Scan an anpass output file and return the "long line"
func GetLongLine(filename string) (string, bool) {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
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
