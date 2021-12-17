package main

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	nxo   = flag.Bool("no-xml-output", false, "dummy flag")
	procs = flag.Int("t", 1, "dummy flag")
)

type Calc struct {
	Energy   float64
	Gradient []float64
}

//go:embed cart.json
var cart []byte

//go:embed sic.json
var sic []byte

//go:embed grad.json
var grad []byte

//go:embed h2co_test.json
var h2co []byte

var geoms = make(map[string]Calc)

func init() {
	err := json.Unmarshal(cart, &geoms)
	if err != nil {
		fmt.Println("error unmarshalling json")
		os.Exit(2)
	}
	// Unmarshal reuses the map, keeping old entries
	err = json.Unmarshal(sic, &geoms)
	if err != nil {
		fmt.Println("error unmarshalling json")
		os.Exit(2)
	}
	err = json.Unmarshal(h2co, &geoms)
	if err != nil {
		fmt.Println("error unmarshalling json")
		os.Exit(2)
	}
	// grad last in case there's overlap we want the gradient too
	err = json.Unmarshal(grad, &geoms)
	if err != nil {
		fmt.Println("error unmarshalling json")
		os.Exit(2)
	}
}

func Molpro(filename string) {
	infile, err := os.Open(filename)
	defer infile.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "trouble opening %s\n", filename)
		os.Exit(3)
	}
	base := filename[:len(filename)-len(filepath.Ext(filename))]
	outfile, err := os.Create(base + ".out")
	defer outfile.Close()
	if err != nil {
		os.Exit(4)
	}
	// TODO include gradients
	scanner := bufio.NewScanner(infile)
	var (
		geom bool
		str  strings.Builder
	)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		switch {
		case strings.Contains(line, "geometry={"):
			geom = true
		case strings.Contains(line, "}") && geom:
			geom = false
		case geom && len(fields) == 4:
			str.WriteString(strings.Join(fields, " ") + "\n")
		}
	}
	val, ok := geoms[str.String()]
	if !ok {
		os.Exit(5)
	}
	fmt.Fprintf(outfile, "dummy output\nenergy= %20.12f\n", val.Energy)
	gl := len(val.Gradient)
	labels := []string{"X", "Y", "Z"}
	for i := 0; i < 3; i++ {
		line := val.Gradient[i*gl/3 : (i+1)*gl/3]
		fmt.Fprintf(outfile, "GRAD%s(1:%d)   = [ ",
			labels[i], gl)
		for _, l := range line {
			fmt.Fprintf(outfile, "%20.15f", l)
		}
		fmt.Fprint(outfile, "] AU\n")
	}
}
