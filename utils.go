package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
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
		fmt.Fprintf(os.Stderr, "RunProgram: running %q on %q\n",
			progName, filename)
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
		case skip > 0:
			skip--
		case Conf.GeomType == "xyz" && len(fields) == 1:
			// natoms line in xyz
			skip++
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
	if OPT {
		dirs = []string{"opt", "freq"}
	}
	dirs = append(dirs, "pts/inp", "freqs")
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

// IntAbs returns the absolute value of n
func IntAbs(n int) int {
	if n < 0 {
		return -1 * n
	}
	return n
}

// ZipXYZ puts slices of atom names and Cartesian coordinates together
// into a single string
func ZipXYZ(names []string, coords []float64) string {
	var buf bytes.Buffer
	if len(names) != len(coords)/3 {
		panic("ZipXYZ: dimension mismatch on names and coords")
	} else if len(coords)%3 != 0 {
		panic("ZipXYZ: coords not divisible by 3")
	}
	for i, c := range coords {
		if math.Abs(c) < 1e-10 {
			coords[i] = 0
		}
	}
	for i := range names {
		fmt.Fprintf(&buf, "%s %.10f %.10f %.10f\n",
			names[i], coords[3*i], coords[3*i+1], coords[3*i+2])
	}
	return buf.String()
}

// Step adjusts coords by delta in the steps indices
func Step(coords []float64, steps ...int) []float64 {
	var c = make([]float64, len(coords))
	copy(c, coords)
	for _, v := range steps {
		if v < 0 {
			v = -1 * v
			c[v-1] -= Conf.Deltas[v-1]
		} else {
			c[v-1] += Conf.Deltas[v-1]
		}
	}
	return c
}

// GetCPU returns the CPU time used by the current process in
// nanoseconds
func GetCPU() int64 {
	use := new(syscall.Rusage)
	syscall.Getrusage(syscall.RUSAGE_SELF, use)
	return use.Utime.Nano() + use.Stime.Nano()
}

// GetCPULimit returns the Cur (soft) and Max (hard) CPU time limits
// in seconds
func GetCPULimit() (cur, max uint64) {
	lim := new(syscall.Rlimit)
	syscall.Getrlimit(syscall.RLIMIT_CPU, lim)
	return lim.Cur, lim.Max
}

// CatchPanic recovers from a panic to clear the queue and then
// continues the panic
func CatchPanic() {
	if r := recover(); r != nil {
		fmt.Println("running queueClear before panic")
		queueClear(Global.WatchedJobs)
		panic(r)
	}
}

// CatchKill catches SIGTERM to clear the queue before exiting cleanly
func CatchKill() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Signal(syscall.SIGTERM))
	<-c
	fmt.Println("running queueClear before SIGTERM")
	queueClear(Global.WatchedJobs)
	errExit(fmt.Errorf("received SIGTERM"), "")
}

// Summarize prints a summary table of the vibrational frequency data
func Summarize(w io.Writer, zpt float64, mpHarm, idHarm, spHarm, spFund,
	spCorr []float64) error {
	fmt.Fprint(w, "\n== Results == \n\n")
	if len(mpHarm) != len(idHarm) ||
		len(mpHarm) != len(spHarm) ||
		len(mpHarm) != len(spFund) ||
		len(mpHarm) != len(spCorr) {
		return fmt.Errorf("error Summarize: dimension mismatch")
	}
	fmt.Fprintf(w, "ZPT = %.1f\n", zpt)
	fmt.Fprintf(w, "+%8s-+%8s-+%8s-+%8s-+%8s-+\n",
		"--------", "--------", "--------", "--------", "--------")
	fmt.Fprintf(w, "|%8s |%8s |%8s |%8s |%8s |\n",
		"Mp Harm", "Id Harm", "Sp Harm", "Sp Fund", "Sp Corr")
	fmt.Fprintf(w, "+%8s-+%8s-+%8s-+%8s-+%8s-+\n",
		"--------", "--------", "--------", "--------", "--------")
	for i := range mpHarm {
		fmt.Fprintf(w, "|%8.1f |%8.1f |%8.1f |%8.1f |%8.1f |\n",
			mpHarm[i], idHarm[i], spHarm[i], spFund[i], spCorr[i])
	}
	fmt.Fprintf(w, "+%8s-+%8s-+%8s-+%8s-+%8s-+\n\n",
		"--------", "--------", "--------", "--------", "--------")
	return nil
}

// CartPoints returns the number of points required for a Cartesian
// force field with n coordinates
func CartPoints(n int) int {
	return 2 * n * (n*n*n + 2*n*n + 8*n + 1) / 3
}

// GradPoints returns the number of points required for a Cartesian
// gradient force field with n coordinates
func GradPoints(n int) int {
	return n * (4*n*n + 12*n + 8) / 3
}

// DupOutErr uses syscall.Dup2 to direct the stdout and stderr streams
// to files
func DupOutErr(infile string) {
	// set up output and err files and dup their fds to stdout and stderr
	// https://github.com/golang/go/issues/325
	base := infile[:len(infile)-len(path.Ext(infile))]
	outfile, _ := os.Create(base + ".out")
	errfile, _ := os.Create(base + ".err")
	syscall.Dup2(int(outfile.Fd()), 1)
	syscall.Dup2(int(errfile.Fd()), 2)
}

// XYZGeom converts a string xyz style geometry into a list of atom
// names and coords
func XYZGeom(geom string) (names []string, coords []float64) {
	lines := strings.Split(geom, "\n")
	var skip int
	for i, line := range lines {
		if line == "" {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}
		fields := strings.Fields(line)
		if i == 0 && len(fields) == 1 {
			skip++
			continue
		}
		if len(fields) == 4 {
			names = append(names, fields[0])
			for _, s := range fields[1:] {
				f, _ := strconv.ParseFloat(s, 64)
				coords = append(coords, f)
			}
		}
	}
	return
}

// PrintFortFile prints the third derivative force constants in the
// format expected by SPECTRO
func PrintFortFile(fc []CountFloat, natoms, other int, filename string) int {
	f, _ := os.Create(filename)
	defer f.Close()
	fmt.Fprintf(f, "%5d%5d", natoms, other)
	for i := range fc {
		if i%3 == 0 {
			fmt.Fprintf(f, "\n")
		}
		fmt.Fprintf(f, "%20.10f", fc[i].Val)
	}
	fmt.Fprint(f, "\n")
	return len(fc)
}
