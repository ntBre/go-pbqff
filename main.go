/*
Push-button QFF
---------------
The goal of this program is to streamline the generation
of quartic force fields, automating as many pieces as possible.
*/

package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"path/filepath"

	"github.com/ntBre/chemutils/summarize"
)

// Points is a wrapper for a []Calc with an embedded sync.Mutex
type Points struct {
	Calcs []Calc
	sync.Mutex
}

// Globals for queue
var (
	points Points
)

const (
	// these should  be in the input
	jobLimit  = 50
	chunkSize = 50
	resBound  = 1e-16
	help      = `Requirements:
- intder, anpass, and spectro executables
- template intder.in, anpass.in, spectro.in, and molpro.in files
  - intder.in should be a pts intder input and have the old geometry to serve as template
  - anpass.in should be a first run anpass file, not a stationary point
  - spectro.in should not have any resonance information
  - molpro.in should have the geometry removed
    - on sequoia, the custom energy parameter pbqff=energy is required for parsing
Flags:
`
)

// Flags for the procedures to be run
const (
	OPT int = 1 << iota
	PTS
	FREQS
)

// DoOpt is a helper function for checking whether the OPT flag is set
func DoOpt() bool { return flags&OPT > 0 }

// DoPts is a helper function for checking whether the PTS flag is set
func DoPts() bool { return flags&PTS > 0 }

// DoFreqs is a helper function for checking whether the FREQS flag is set
func DoFreqs() bool { return flags&FREQS > 0 }

var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	overwrite  = flag.Bool("o", false, "overwrite existing inp directory")
	pts        = flag.Bool("pts", false, "start by running pts on optimized geometry from opt")
	freqs      = flag.Bool("freqs", false, "start from running anpass on the pts output")
	irdy       = flag.String("irdy", "", "intder file is ready to be used in pts; specify the atom order")
	flags      int
)

// Global variables
var (
	Input            [NumKeys]string
	dirs             = []string{"opt", "freq", "pts", "freqs", "pts/inp"}
	brokenFloat      = math.NaN()
	energyLine       = regexp.MustCompile(`energy=`) // default search patterns, altered for sequoia in pbs.go
	molproTerminated = "Molpro calculation terminated"
	defaultOpt       = "optg,grms=1.d-8,srms=1.d-8"
	pbs              string
)

// Errors used throughout
var (
	ErrEnergyNotFound      = errors.New("Energy not found in Molpro output")
	ErrFileNotFound        = errors.New("Molpro output file not found")
	ErrEnergyNotParsed     = errors.New("Energy not parsed in Molpro output")
	ErrFinishedButNoEnergy = errors.New("Molpro output finished but no energy found")
	ErrFileContainsError   = errors.New("Molpro output file contains an error")
	ErrBlankOutput         = errors.New("Molpro output file exists but is blank")
	ErrInputGeomNotFound   = errors.New("Geometry not found in input file")
	ErrTimeout             = errors.New("Timeout waiting for signal")
)

// MakeName builds a molecule name from a geometry
func MakeName(geom string) (name string) {
	atoms := make(map[string]int)
	split := strings.Split(geom, "\n")
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
func ReadFile(filename string) (lines []string) {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("ReadFile: error %q open file %q\n", err, filename)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			lines = append(lines, line)
		}
	}
	return
}

// MakeDirs sets up the directory structure described by dirs
func MakeDirs(root string) (err error) {
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
		e := os.Mkdir(filename, 0755)
		if e != nil {
			err = fmt.Errorf("error MakeDirs: %q on making directory %q",
				e, dir)
		}
	}
	return err
}

// ParseFlags parses command line flags and returns a slice of
// the remaining arguments
func ParseFlags() []string {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), help)
		flag.PrintDefaults()
	}
	flag.Parse()
	switch {
	case *freqs:
		flags = FREQS
	case *pts:
		flags = PTS | FREQS
	default:
		flags = OPT | PTS | FREQS
	}
	return flag.Args()
}

// HandleSignal waits to receive a real-time signal or times out
func HandleSignal(sig int, timeout time.Duration) error {
	sigChan := make(chan os.Signal, 1)
	sig1Want := os.Signal(syscall.Signal(sig))
	signal.Notify(sigChan, sig1Want)
	select {
	// either receive signal
	case <-sigChan:
		return nil
	// or timeout after and retry
	case <-time.After(timeout):
		return ErrTimeout
	}
}

// GetNames takes a cartesian geometry and extract the atom names
func GetNames(cart string) (names []string) {
	lines := strings.Split(cart, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 4 {
			names = append(names, fields[0])
		}
	}
	return
}

// Tennis moves intder output files to the filenames expected by spectro
func Tennis() {
	err := os.Rename("freqs/file15", "freqs/fort.15")
	if err == nil {
		err = os.Rename("freqs/file20", "freqs/fort.30")
	}
	if err == nil {
		err = os.Rename("freqs/file24", "freqs/fort.40")
	}
	if err != nil {
		panic(err)
	}
}

// Summarize prints a summary table of the vibrational frequency data
func Summarize(zpt float64, mpHarm, idHarm, spHarm, spFund, spCorr []float64) error {
	if len(mpHarm) != len(idHarm) ||
		len(mpHarm) != len(spHarm) ||
		len(mpHarm) != len(spFund) ||
		len(mpHarm) != len(spCorr) {
		return fmt.Errorf("error Summarize: dimension mismatch")
	}
	fmt.Printf("ZPT = %.1f\n", zpt)
	fmt.Printf("+%8s-+%8s-+%8s-+%8s-+%8s-+\n",
		"--------", "--------", "--------", "--------", "--------")
	fmt.Printf("|%8s |%8s |%8s |%8s |%8s |\n",
		"Mp Harm", "Id Harm", "Sp Harm", "Sp Fund", "Sp Corr")
	fmt.Printf("+%8s-+%8s-+%8s-+%8s-+%8s-+\n",
		"--------", "--------", "--------", "--------", "--------")
	for i := range mpHarm {
		fmt.Printf("|%8.1f |%8.1f |%8.1f |%8.1f |%8.1f |\n",
			mpHarm[i], idHarm[i], spHarm[i], spFund[i], spCorr[i])
	}
	fmt.Printf("+%8s-+%8s-+%8s-+%8s-+%8s-+\n",
		"--------", "--------", "--------", "--------", "--------")
	return nil
}

// UpdateZmat updates an old zmat with new parameters
func UpdateZmat(old, new string) string {
	lines := strings.Split(old, "\n")
	for i, line := range lines {
		if strings.Contains(line, "}") {
			lines = lines[:i+1]
			break
		}
	}
	updated := strings.Join(lines, "\n")
	return updated + "\n" + new
}

// WhichCluster sets the PBS template and energyLine depending on the
// which computer is to be used
func WhichCluster() {
	sequoia := regexp.MustCompile(`(?i)sequoia`)
	maple := regexp.MustCompile(`(?i)maple`)
	q := Input[QueueType]
	switch {
	case q == "", maple.MatchString(q):
		pbs = pbsMaple
	case sequoia.MatchString(q):
		energyLine = regexp.MustCompile(`PBQFF\(2\)`)
		pbs = pbsSequoia
	default:
		panic("no queue selected")
	}
}

// Optimize runs a Molpro optimization in the opt directory
func Optimize(prog *Molpro) {
	// write opt.inp and mp.pbs
	prog.WriteInput("opt/opt.inp", opt)
	WritePBS("opt/mp.pbs",
		&Job{MakeName(Input[Geometry]) + "-opt", "opt/opt.inp", 35})
	// submit opt, wait for it to finish in main goroutine - block
	Submit("opt/mp.pbs")
	outfile := "opt/opt.out"
	_, err := prog.ReadOut(outfile)
	for err != nil {
		HandleSignal(35, time.Minute)
		_, err = prog.ReadOut(outfile)
		if (err == ErrEnergyNotParsed || err == ErrFinishedButNoEnergy ||
			err == ErrFileContainsError || err == ErrBlankOutput) ||
			err == ErrFileNotFound {

			fmt.Fprintln(os.Stderr, "resubmitting for", err)
			Submit("opt/mp.pbs")
		}
	}
}

// Frequency runs a Molpro harmonic frequency calculation in the freq
// directory
func Frequency(prog *Molpro) ([]float64, bool) {
	// write freq.inp and that mp.pbs
	prog.WriteInput("freq/freq.inp", freq)
	WritePBS("freq/mp.pbs",
		&Job{MakeName(Input[Geometry]) + "-freq", "freq/freq.inp", 35})
	// submit freq, wait in separate goroutine
	// doesn't matter if this finishes
	Submit("freq/mp.pbs")
	outfile := "freq/freq.out"
	_, err := prog.ReadOut(outfile)
	for err != nil {
		HandleSignal(35, time.Minute)
		_, err = prog.ReadOut(outfile)
		// dont resubmit freq
		if err == ErrEnergyNotParsed || err == ErrFinishedButNoEnergy ||
			err == ErrFileContainsError {
			fmt.Fprintln(os.Stderr, "error in freq, aborting that calculation")
			return nil, false
		}
	}
	return prog.ReadFreqs(outfile), true
}

// DoAnpass runs anpass
func DoAnpass(anp *Anpass, energies []float64) string {
	anp.WriteAnpass("freqs/anpass1.in", energies)
	RunAnpass("freqs/anpass1")
	longLine, ok := GetLongLine("freqs/anpass1.out")
	if !ok {
		panic("Problem getting long line from anpass1.out")
	}
	anp.WriteAnpass2("freqs/anpass2.in", longLine, energies)
	RunAnpass("freqs/anpass2")
	return longLine
}

// DoIntder runs freqs intder
func DoIntder(intder *Intder, atomNames []string, longLine string) (string, []float64) {
	intder.WriteGeom("freqs/intder_geom.in", longLine)
	RunIntder("freqs/intder_geom")
	coords := intder.ReadGeom("freqs/intder_geom.out")
	intder.Read9903("freqs/fort.9903")
	intder.WriteFreqs("freqs/intder.in", atomNames)
	RunIntder("freqs/intder")
	intderHarms := intder.ReadOut("freqs/intder.out")
	Tennis()
	return coords, intderHarms
}

// DoSpectro runs spectro
func DoSpectro(spectro *Spectro, harms []float64) (float64, []float64, []float64, []float64) {
	spectro.Nfreqs = len(harms)
	spectro.WriteInput("freqs/spectro.in")
	RunSpectro("freqs/spectro")
	spectro.ReadOutput("freqs/spectro.out")
	spectro.WriteInput("freqs/spectro2.in")
	RunSpectro("freqs/spectro2")
	// have rotational constants from FreqReport, but need to incorporate them
	zpt, spHarm, spFund, spCorr,
		_, _, _ := summarize.Spectro("freqs/spectro2.out", spectro.Nfreqs)
	return zpt, spHarm, spFund, spCorr
}

// Drain drains the queue of jobs and receives on ch when ready for more
func Drain(prog *Molpro) (min float64, energies []float64) {
	nJobs := len(points.Calcs)
	energies = make([]float64, nJobs, nJobs)
	for nJobs > 0 {
		shortenBy := 0
		for i := 0; i < nJobs; i++ {
			job := points.Calcs[i]
			energy, err := prog.ReadOut(job.Name + ".out")
			if err == nil {
				points.Lock()
				points.Calcs[nJobs-1], points.Calcs[i] = points.Calcs[i], points.Calcs[nJobs-1]
				nJobs--
				points.Calcs = points.Calcs[:nJobs]
				points.Unlock()
				if energy < min {
					min = energy
				}
				energies[job.Index] = energy
				shortenBy++
			} else if err == ErrEnergyNotParsed || err == ErrFinishedButNoEnergy ||
				err == ErrFileContainsError || err == ErrBlankOutput {
				if err == ErrFileContainsError {
					fmt.Fprintf(os.Stderr, "error: %v on %s\n", err, job.Name)
				}
				fmt.Fprintf(os.Stderr,
					"resubmitting %s for %s, with %d jobs remaining\n", job.Name, err, nJobs)
				// delete output file to prevent rereading the same one
				os.Remove(job.Name + ".out")
				// if we have to resubmit, need individual submission from pbsMaple
				pbs = pbsMaple
				WritePBS(job.Name+".pbs", &Job{"redo", job.Name + ".inp", 35})
				Submit(job.Name + ".pbs")
			}
		}
		// if the list is shortened by less than 10%,
		// sleep. could play with both of these values
		if nJobs > 0 && float64(shortenBy/nJobs) < 0.1 {
			fmt.Fprintf(os.Stderr,
				"only shortened by %d out of %d remaining, sleeping\n", shortenBy, nJobs)
			time.Sleep(time.Second)
		}
		nJobs = len(points.Calcs)
		fmt.Fprintf(os.Stderr, "nJobs: %d\n", nJobs)
	}
	return
}

func initialize() (*Molpro, *Intder, *Anpass) {
	// parse flags for overwrite before mkdirs
	args := ParseFlags()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "pbqff: no input file supplied\n")
		os.Exit(1)
	}
	ParseInfile(args[0])
	WhichCluster()
	if Input[Program] == "cccr" {
		energyLine = regexp.MustCompile(`^\s*CCCRE\s+=`)
	}
	mpName := "molpro.in"
	idName := "intder.in"
	apName := "anpass.in"
	prog, err := LoadMolpro("molpro.in")
	if err != nil {
		errExit(err, fmt.Sprintf("loading molpro input %q", mpName))
	}
	intder, err := LoadIntder("intder.in")
	if err != nil {
		errExit(err, fmt.Sprintf("loading intder input %q", idName))
	}
	anpass, err := LoadAnpass("anpass.in")
	if err != nil {
		errExit(err, fmt.Sprintf("loading anpass input %q", apName))
	}
	return prog, intder, anpass
}

func errExit(err error, msg string) {
	fmt.Fprintf(os.Stderr, "pbqff: %v %s\n", err, msg)
	os.Exit(1)
}

func main() {
	prog, intder, anpass := initialize()
	var (
		mpHarm   []float64
		finished bool
		cart     string
		zmat     string
		err      error
	)
	if DoOpt() {
		MakeDirs(".")
		prog.Geometry = FormatZmat(Input[Geometry])
		Optimize(prog)
		cart, zmat, err = prog.HandleOutput("opt/opt")
		if err != nil {
			panic(err)
		}
		// only need this if running a freq
		prog.Geometry = UpdateZmat(prog.Geometry, zmat)
		// run the frequency in the background
		go func() {
			mpHarm, finished = Frequency(prog)
		}()
	} else {
		cart = Input[Geometry]
	}
	var atomNames []string
	if *irdy == "" {
		atomNames = intder.ConvertCart(cart)
	} else {
		atomNames = strings.Fields(*irdy)
	}
	if DoPts() {
		intder.WritePts("pts/intder.in")
		RunIntder("pts/intder")
		points.Calcs = prog.BuildPoints("pts/file07", atomNames, true)
		subfiles, err := filepath.Glob("pts/inp/main*.pbs")
		if err != nil {
			panic(err)
		}
		for _, file := range subfiles {
			Submit(file)
		}
		// this works if no points were deleted, else need a resume from checkpoint thing
	} else {
		points.Calcs = prog.BuildPoints("pts/file07", atomNames, false)
	}
	var (
		energies []float64
		min      float64
	)
	min, energies = Drain(prog)

	// convert to relative energies
	for i := range energies {
		energies[i] -= min
	}
	longLine := DoAnpass(anpass, energies)
	coords, intderHarms := DoIntder(intder, atomNames, longLine)
	spectro, err := LoadSpectro("spectro.in", atomNames, coords)
	if err != nil {
		errExit(err, "loading spectro input")
	}
	zpt, spHarm, spFund, spCorr := DoSpectro(spectro, intderHarms)
	if !finished {
		mpHarm = make([]float64, spectro.Nfreqs)
	}
	Summarize(zpt, mpHarm, intderHarms, spHarm, spFund, spCorr)
}
