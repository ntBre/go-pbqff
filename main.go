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
	"runtime/pprof"
	"sort"
	"strings"
	"syscall"
	"time"
)

const (
	resBound = 1e-16
	help     = `Requirements:
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

var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	overwrite  = flag.Bool("o", false, "overwrite existing inp directory")
)

// Global variables
var (
	Input            [NumKeys]string
	dirs             = []string{"opt", "freq", "pts", "freqs", "pts/inp"}
	brokenFloat      = math.NaN()
	energyLine       = "energy=" // default search patterns, altered for sequoia in pbs.go
	energySpace      = 1
	molproTerminated = "Molpro calculation terminated"
	defaultOpt       = "optg,grms=1.d-8,srms=1.d-8"
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

func main() {
	// parse flags for overwrite before mkdirs
	Args := ParseFlags()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	MakeDirs(".")
	if len(Args) < 1 {
		log.Fatal("pbqff: no input file supplied")
	}
	// might want a LoadDefaults function or something
	// and then overwrite parts with ParseInfile
	ParseInfile(Args[0])
	prog := LoadMolpro("molpro.in")
	prog.Geometry = FormatZmat(Input[Geometry])
	// write opt.inp and mp.pbs
	prog.WriteInput("opt/opt.inp", opt)
	WritePBS("opt/mp.pbs",
		&Job{MakeName(Input[Geometry]) + "/opt", "opt/opt.inp", 35})
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
	cart, zmat, err := prog.HandleOutput("opt/opt")
	if err != nil {
		// actually want to try to recover here probably
		panic(err)
	}
	// write freq.inp and that mp.pbs
	prog.Geometry = UpdateZmat(prog.Geometry, zmat)
	prog.WriteInput("freq/freq.inp", freq)
	WritePBS("freq/mp.pbs",
		&Job{MakeName(Input[Geometry]) + "/freq", "freq/freq.inp", 35})
	// submit freq, wait in separate goroutine
	// doesn't matter if this finishes
	Submit("freq/mp.pbs")
	outfile = "freq/freq.out"
	var (
		mpHarm   []float64
		finished bool
	)
	go func() {
		_, err = prog.ReadOut(outfile)
		for err != nil {
			HandleSignal(35, time.Minute)
			_, err = prog.ReadOut(outfile)
			// dont resubmit freq
			if err == ErrEnergyNotParsed || err == ErrFinishedButNoEnergy ||
				err == ErrFileContainsError {
				fmt.Fprintln(os.Stderr, "error in freq, aborting that calculation")
				return
			}
		}
		mpHarm = prog.ReadFreqs(outfile)
		finished = true
	}()
	// set up pts using opt.log geometry and given intder.in file
	intder := LoadIntder("intder.in")
	atomNames := intder.ConvertCart(cart)
	intder.WritePts("pts/intder.in")
	// run intder
	RunIntder("pts/intder")
	// build points and the list of pts to submit
	pts := prog.BuildPoints("pts/file07", atomNames)
	// need to limit this to 220 on sequoia
	// must have communicating goroutines between submit
	// and readOut since we can't submit until some of the running ones finish
	// submit points, wait for them to finish
	for _, job := range pts {
		Submit(job.Name + ".pbs")
	}

	// - check for failed jobs, probably just loop at some interval
	//   doesnt need to be fast (and resource intensive) like gocart
	ptsInit := len(pts)
	energies := make([]float64, ptsInit)
	var min float64
	nJobs := ptsInit
	for nJobs > 0 {
		shortenBy := 0
		for i := 0; i < nJobs; i++ {
			job := pts[i]
			energy, err := prog.ReadOut(job.Name + ".out")
			if err == nil {
				pts[nJobs-1], pts[i] = pts[i], pts[nJobs-1]
				nJobs--
				pts = pts[:nJobs]
				if energy < min {
					min = energy
				}
				energies[job.Index] = energy
				shortenBy++
			} else if err == ErrEnergyNotParsed || err == ErrFinishedButNoEnergy ||
				err == ErrFileContainsError {
				// Removing this one too now since problem on Sequoia
				// same problem as below, solved by queue
				// || err == ErrBlankOutput { // ||
				// must be a better way to do this -> check queue
				// disable for now
				// (err == ErrFileNotFound && len(pts) < ptsInit/20) {
				// write error found in case it can't be handled by resubmit
				// then we need to kill it, manually for now
				if err == ErrFileContainsError {
					fmt.Fprintf(os.Stderr, "error: %v on %s\n", err, job.Name)
				}
				fmt.Fprintf(os.Stderr,
					"resubmitting %s for %s, with %d jobs remaining\n", job.Name, err, nJobs)
				// delete output file to prevent rereading the same one
				os.Remove(job.Name + ".out")
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
	}

	// convert to relative energies
	for i := range energies {
		energies[i] -= min
	}
	// write anpass1.in
	// TODO make test case out of this
	// PROBLEM WITH NUMERICAL DISPS - 14 extra points in anpass not in intder
	// why the extra dummy atom in freqs intder too?  r2666=mason/hco+/freqs
	anpass := LoadAnpass("anpass.in")
	anpass.WriteAnpass("freqs/anpass1.in", energies)
	// run anpass1.in
	RunAnpass("freqs/anpass1")
	// Read anpass1.out
	longLine, ok := GetLongLine("freqs/anpass1.out")
	if !ok {
		panic("Problem getting long line from anpass1.out")
	}
	// - write anpass2.in, run anpass
	anpass.WriteAnpass2("freqs/anpass2.in", longLine, energies)
	// run anpass2.in
	RunAnpass("freqs/anpass2")
	// write intder_geom.in, run intder_geom
	intder.WriteGeom("freqs/intder_geom.in", longLine)
	RunIntder("freqs/intder_geom")
	// update intder geometry
	coords := intder.ReadGeom("freqs/intder_geom.out")
	// read freqs/intder.in bottom from fort.9903
	intder.Read9903("freqs/fort.9903")
	// write freqs/intder.in, run intder
	intder.WriteFreqs("freqs/intder.in", atomNames)
	RunIntder("freqs/intder")
	// read harmonics from intder.out
	intderHarms := intder.ReadOut("freqs/intder.out")
	// move files (tennis)
	Tennis()
	// load spectro template
	spectro := LoadSpectro("spectro.in", atomNames, coords)
	spectro.Nfreqs = len(intderHarms)
	// write spectro input file
	spectro.WriteInput("freqs/spectro.in")
	// run spectro
	RunSpectro("freqs/spectro")
	// read spectro output, handle resonances
	spectro.ReadOutput("freqs/spectro.out")
	// write the new input
	spectro.WriteInput("freqs/spectro2.in")
	// run spectro
	RunSpectro("freqs/spectro2")
	// extract output
	zpt, spHarm, spFund, spCorr, _, _, _ := spectro.FreqReport("freqs/spectro2.out")
	if !finished {
		mpHarm = make([]float64, spectro.Nfreqs)
	}
	// print summary table
	Summarize(zpt, mpHarm, intderHarms, spHarm, spFund, spCorr)
	// TODO summarize rotational constants, geometry parameters,
	//      maybe assignments too
}
