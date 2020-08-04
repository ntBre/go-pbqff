/*
Push-button QFF
---------------
The goal of this program is to streamline the generation
of quartic force fields, automating as many pieces as possible.
*/

package main

import (
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"path/filepath"

	"github.com/ntBre/chemutils/summarize"
)

// Calc holds the name of a job to be run and its result's index in
// the output array
type Calc struct {
	Name   string
	Target *[]float64 // Target result array
	Index  int
}

// GarbageHeap is a slice of Basenames to be deleted
type GarbageHeap struct {
	heap []string // list of basenames
}

// Add a filename to the heap
func (g *GarbageHeap) Add(basename string) {
	g.heap = append(g.heap, basename)
}

func (g *GarbageHeap) Len() int {
	return len(g.heap)
}

// Dump deletes the globbed files in the heap using an appended *
func (g *GarbageHeap) Dump() {
	toDelete := make([]string, 0)
	for _, v := range g.heap {
		files, _ := filepath.Glob(v + "*")
		toDelete = append(toDelete, files...)
	}
	for _, f := range toDelete {
		os.Remove(f)
	}
	g.heap = []string{}
}

const (
	// these should  be in the input
	chunkSize = 100
	jobLimit  = 2 * chunkSize
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
	CART
	FREQS
)

// DoOpt is a helper function for checking whether the OPT flag is set
func DoOpt() bool { return flags&OPT > 0 }

// DoPts is a helper function for checking whether the PTS flag is set
func DoPts() bool { return flags&PTS > 0 }

// DoFreqs is a helper function for checking whether the FREQS flag is
// set
func DoFreqs() bool { return flags&FREQS > 0 }

// DoCart is a helper function for checking whether the CART flag is
// set
func DoCart() bool { return flags&CART > 0 }

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

// Globals for queue
var (
	fc2 []float64
	fc3 []float64
	fc4 []float64
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
func Frequency(prog *Molpro, absPath string) ([]float64, bool) {
	// write freq.inp and that mp.pbs
	prog.WriteInput(absPath+"/freq.inp", freq)
	WritePBS(absPath+"/mp.pbs",
		&Job{MakeName(Input[Geometry]) + "-freq", absPath + "/freq.inp", 35})
	// submit freq, wait in separate goroutine
	// doesn't matter if this finishes
	Submit(absPath + "/mp.pbs")
	outfile := absPath + "/freq.out"
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
func DoSpectro(spectro *Spectro, nharms int) (float64, []float64, []float64, []float64) {
	spectro.Nfreqs = nharms
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
func Drain(prog *Molpro, ch chan Calc) (min float64) {
	points := make([]Calc, 0)
	var nJobs int
	var finished int
	heap := new(GarbageHeap)
	for {
		shortenBy := 0
		for i := 0; i < nJobs; i++ {
			job := points[i]
			energy, err := prog.ReadOut(job.Name + ".out")
			if err == nil {
				points[nJobs-1], points[i] = points[i], points[nJobs-1]
				nJobs--
				points = points[:nJobs]
				if energy < min {
					min = energy
				}
				for len(*job.Target) <= job.Index {
					(*job.Target) = append(*job.Target, 0)
				}
				(*job.Target)[job.Index] = energy
				heap.Add(job.Name)
				shortenBy++
				finished++
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
		if shortenBy < 1 {
			fmt.Fprintln(os.Stderr, "Didn't shorten, sleeping")
			time.Sleep(time.Second)
		}
		if heap.Len() >= chunkSize {
			heap.Dump()
		}
		fmt.Fprintf(os.Stderr, "finished: %d of %d submitted\n", finished, submitted)
		calc, ok := <-ch
		points = append(points, calc)
		nJobs = len(points)
		if !ok && finished == submitted {
			return
		}
	}
	// unreachable
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
	switch Input[Program] {
	case "cccr":
		energyLine = regexp.MustCompile(`^\s*CCCRE\s+=`)
	case "gocart":
		flags |= CART
		energyLine = regexp.MustCompile(`^\s*CARTFC\s+=`)
		// default:
		// 	errExit(fmt.Errorf("%s not implemented as a Program", Input[Program]), "")
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

var submitted int

func main() {
	prog, intder, anpass := initialize()
	var (
		mpHarm    []float64
		finished  bool
		cart      string
		zmat      string
		err       error
		atomNames []string
		energies  []float64
		min       float64
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
			absPath, _ := filepath.Abs("freq")
			mpHarm, finished = Frequency(prog, absPath)
		}()
	} else {
		cart = Input[Geometry]
	}

	ch := make(chan Calc, jobLimit)

	if !DoCart() {
		if *irdy == "" {
			atomNames = intder.ConvertCart(cart)
		} else {
			atomNames = strings.Fields(*irdy)
		}
		if DoPts() {
			intder.WritePts("pts/intder.in")
			RunIntder("pts/intder")
			go func() {
				prog.BuildPoints("pts/file07", atomNames, &energies, ch, true)
			}()
			// this works if no points were deleted, else need a resume from checkpoint thing
		} else {
			prog.BuildPoints("pts/file07", atomNames, &energies, nil, false)
		}
	}
	// Instead of returning energies, use job.Target = energies, also need a function
	// for getting the index in the array for fc2,3,4 but do that before setting job.Index
	min = Drain(prog, ch)

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
	zpt, spHarm, spFund, spCorr := DoSpectro(spectro, len(intderHarms))
	if !finished {
		mpHarm = make([]float64, spectro.Nfreqs)
	}
	Summarize(zpt, mpHarm, intderHarms, spHarm, spFund, spCorr)
}
