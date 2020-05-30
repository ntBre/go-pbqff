package main

import (
	"os"
	"strings"
)

type Molpro struct {
	Geometry string
	Basis    string
	Charge   string
	Spin     string
	Method   string
}

// Takes an input filename and template filename
// and writes an input file
func (m *Molpro) WriteInput(infile, tfile string) {
	f, err := os.Create(infile)
	if err != nil {
		panic(err)
	}
	t := LoadTemplate(tfile)
	t.Execute(f, m)
}

// Format z-matrix for use in Molpro input
func FormatZmat(geom string) string {
	var out []string
	split := strings.Split(geom, "\n")
	for i, line := range split {
		if strings.Contains(line, "=") {
			out = append(append(append(out, split[:i]...), "}"), split[i:]...)
			break
		}
	}
	return strings.Join(out, "\n")
}
