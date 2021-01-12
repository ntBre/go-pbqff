package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
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
func RunProgram(progName, filename string) error {
	current, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fpath := path.Dir(filename)
	if err = os.Chdir(fpath); err != nil {
		panic(err)
	}
	file := path.Base(filename)
	infile := file + ".in"
	outfile := file + ".out"
	out, err := exec.Command("bash", "-c", progName+" < "+infile+" > "+outfile).Output()
	os.Chdir(current)
	if err != nil {
		return fmt.Errorf("error RunProgram: failed with %v running %q on %q"+
			"\nstdout: %q",
			err, progName, infile, out)
	}
	return nil
}

// MakeName builds a molecule name from a geometry
func MakeName(geom string) (name string) {
	atoms := make(map[string]int)
	split := strings.Split(geom, "\n")
	// TODO handle no comment/natom lines in xyz
	if Conf.Str(GeomType) == "xyz" {
		split = split[2:]
	}
	for _, line := range split {
		fields := strings.Fields(line)
		// not a dummy atom and not a coordinate lol
		if len(fields) >= 1 &&
			!strings.Contains(strings.ToUpper(fields[0]), "X") &&
			!strings.Contains(line, "=") {
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
	if err != nil {
		return nil, err
	}
	defer f.Close()
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
	if DoCart() || DoGrad() {
		dirs = []string{"pts/inp"}
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
