package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// CleanSplit splits a line using strings.Split and then removes
// empty entries
func CleanSplit(str, sep string) []string {
	lines := strings.Split(str, sep)
	clean := make([]string, 0, len(lines))
	for s := range lines {
		if lines[s] != "" {
			clean = append(clean, lines[s])
		}
	}
	return clean
}

// RunProgram runs a program, redirecting STDIN from filename.in
// and STDOUT to filename.out
func RunProgram(progName, filename string) (err error) {
	infile := filename + ".in"
	outfile := filename + ".out"
	cmd := exec.Command(progName)
	f, err := os.Open(infile)
	defer f.Close()
	cmd.Stdin = f
	if err != nil {
		return err
	}
	of, err := os.Create(outfile)
	cmd.Stdout = of
	defer of.Close()
	cmd.Dir = filepath.Dir(filename)
	if err != nil {
		fmt.Println("RunProgram: opening stdout")
		return err
	}
	err = cmd.Run()
	if err != nil {
		fmt.Println("RunProgram: running cmd")
		return err
	}
	return nil
}

// MakeName builds a molecule name from a geometry
func MakeName(geom string) (name string) {
	atoms := make(map[string]int)
	split := strings.Split(geom, "\n")
	var skip int
	for _, line := range split {
		fields := strings.Fields(line)
		// not a dummy atom and not a coordinate
		switch {
		case Conf.Str(GeomType) == "xyz" && len(fields) == 1:
			// natoms line in xyz
			skip++
		case skip > 0:
			skip--
		case len(fields) >= 1 &&
			!strings.Contains(strings.ToUpper(fields[0]), "X") &&
			!strings.Contains(line, "="):
			atoms[strings.ToLower(fields[0])]++
		}
	}
	toSort := make([]string, 0, len(atoms))
	for k := range atoms {
		toSort = append(toSort, k)
	}
	sort.Strings(toSort)
	for _, k := range toSort {
		v := atoms[k]
		k = strings.ToUpper(string(k[0])) + k[1:]
		name += fmt.Sprintf("%s", k)
		if v > 1 {
			name += fmt.Sprintf("%d", v)
		}
	}
	return
}

// ReadFile reads a file and returns a slice of strings of the lines
func ReadFile(filename string) (lines []string, err error) {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

// MakeDirs sets up the directory structure described by dirs
func MakeDirs(root string) (err error) {
	var dirs []string
	if DoCart() || DoGrad() {
		dirs = []string{"pts/inp"}
	} else {
		dirs = []string{"opt", "freq", "pts", "freqs", "pts/inp"}
	}
	for _, dir := range dirs {
		filename := root + "/" + dir
		if _, err := os.Stat(filename); !os.IsNotExist(err) {
			if *overwrite {
				os.RemoveAll(filename)
			} else {
				log.Fatalf("MakeDirs: directory %q already exists "+
					"overwrite with -o\n", dir)
			}
		}
		e := os.MkdirAll(filename, 0755)
		if e != nil {
			err = fmt.Errorf("error MakeDirs: %q on making directory %q",
				e, dir)
		}
	}
	return err
}

func errExit(err error, msg string) {
	fmt.Fprintf(os.Stderr, "pbqff: %v %s\n", err, msg)
	os.Exit(1)
}

// TrimExt takes a file name and returns it with the extension removed
// using filepath.Ext
func TrimExt(filename string) string {
	lext := len(filepath.Ext(filename))
	return filename[:len(filename)-lext]
}

// PrettyPrint pretty prints arr wrapped to three columns
func PrettyPrint(arr []CountFloat) {
	for i, v := range arr {
		if i%3 == 0 && i > 0 {
			fmt.Print("\n")
		}
		fmt.Printf("%20.12f", v.Val)
	}
	fmt.Print("\n")
}

// Warn prints a warning message to stdout and increments the global
// warning counter
func Warn(format string, a ...interface{}) {
	fmt.Printf("warning: "+format+"\n", a...)
	Global.Warnings++
}
