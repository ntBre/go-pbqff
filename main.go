/*
Push-button QFF
---------------
The goal of this program is to streamline the generation
of quartic force fields, automating as many pieces as possible.
(setq compile-command "go build . && scp pbqff woods:Programs/pbqff/.")
(recompile)
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

	"strconv"

	"os/exec"

	"io"
)

const (
	resBound = 1e-16 // warn if anpass residuals above this
	// this could  be in the input
	delta = 0.005
	help  = `Requirements:
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
	GRAD
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

// Flags
var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	overwrite  = flag.Bool("o", false, "overwrite existing inp directory")
	pts        = flag.Bool("pts", false, "start by running pts on optimized geometry from opt")
	freqs      = flag.Bool("freqs", false, "start from running anpass on the pts output")
	debug      = flag.Bool("debug", false, "for debugging, print 2nd derivative energies array")
	checkpoint = flag.Bool("c", false, "resume from checkpoint")
	irdy       = flag.String("irdy", "", "intder file is ready to be used in pts; specify the atom order")
)

// Global variables
var (
	Input            [NumKeys]string
	dirs             = []string{"opt", "freq", "pts", "freqs", "pts/inp"}
	brokenFloat      = math.NaN()
	energyLine       *regexp.Regexp
	molproTerminated = "Molpro calculation terminated"
	defaultOpt       = "optg,grms=1.d-8,srms=1.d-8"
	pbs              string
	nDerivative      int = 4
	ptsJobs          []string
	paraJobs         []string
	paraCount        map[string]int
	errMap           map[error]int
	nodes            []string
	jobLimit         int = 1000
	chunkSize        int = 64
	checkAfter       int = 100
	flags            int
)

// Finite differences denominators
var (
	angbohr  = 0.529177249
	fc2Scale = angbohr * angbohr / (4 * delta * delta)
	fc3Scale = angbohr * angbohr * angbohr / (8 * delta * delta * delta)
	fc4Scale = angbohr * angbohr * angbohr * angbohr / (16 * delta * delta * delta * delta)
)

// Cartesian arrays
var (
	fc2 []CountFloat
	fc3 []CountFloat
	fc4 []CountFloat
	e2d []CountFloat
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

// Calc holds the name of a job to be run and its result's index in
// the output array
type Calc struct {
	Name     string
	Targets  []Target
	Result   float64
	ID       string
	noRun    bool
	cmdfile  string
	chunkNum int
	Resub    *Calc
	Src      *Source
}

// CountFloat combines a value with a counter that keeps track of how
// many times it has been modified, and a boolean Loaded to see if it
// was loaded from a checkpoint file
type CountFloat struct {
	Val    float64
	Count  int
	Loaded bool
}

// Add modifies the underlying value of c and decrements its counter
func (c *CountFloat) Add(plus float64) {
	c.Val += plus
	c.Count--
	if c.Count < 0 {
		panic("added to CountFloat too many times")
	}
}

// Done reports whether or not c's count has reached zero
func (c *CountFloat) Done() bool { return c.Count == 0 }

// FloatsFromCountFloats converts a slice of CountFloats to the
// corresponding Float64s
func FloatsFromCountFloats(cfs []CountFloat) (floats []float64) {
	for _, cf := range cfs {
		floats = append(floats, cf.Val)
	}
	return
}

// A Source is CountFloat slice and an index in that slice
type Source struct {
	Slice *[]CountFloat
	Index int
}

// Len returns the length of s's underlying slice
func (s *Source) Len() int { return len(*s.Slice) }

// Value returns s's underlying value
func (s *Source) Value() float64 {
	return (*s.Slice)[s.Index].Val
}

// Target combines a coefficient, target array, and the index into
// that array
type Target struct {
	Coeff float64
	Slice *[]CountFloat
	Index int
}

// GarbageHeap is a slice of Basenames to be deleted
type GarbageHeap struct {
	heap []string // list of basenames
}

// Add a filename to the heap
func (g *GarbageHeap) Add(basename string) {
	g.heap = append(g.heap, basename)
}

// Len returns the length of g's underlying slice
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
	if *checkpoint {
		LoadCheckpoint()
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
func Optimize(prog *Molpro) (E0 float64) {
	// write opt.inp and mp.pbs
	prog.WriteInput("opt/opt.inp", opt)
	WritePBS("opt/mp.pbs",
		&Job{MakeName(Input[Geometry]) + "-opt", "opt/opt.inp", 35, ""}, pbsMaple)
	// submit opt, wait for it to finish in main goroutine - block
	Submit("opt/mp.pbs")
	outfile := "opt/opt.out"
	_, _, _, err := prog.ReadOut(outfile)
	for err != nil {
		HandleSignal(35, time.Minute)
		E0, _, _, err = prog.ReadOut(outfile)
		if (err == ErrEnergyNotParsed || err == ErrFinishedButNoEnergy ||
			err == ErrFileContainsError || err == ErrBlankOutput) ||
			err == ErrFileNotFound {

			fmt.Fprintln(os.Stderr, "resubmitting for", err)
			Submit("opt/mp.pbs")
		}
	}
	return
}

// RefEnergy runs a Molpro single point energy calculation in the
// pts/inp directory
func RefEnergy(prog *Molpro) (E0 float64) {
	dir := "pts/inp/"
	infile := "ref.inp"
	pbsfile := "ref.pbs"
	prog.WriteInput(dir+infile, opt)
	WritePBS(dir+pbsfile,
		&Job{MakeName(Input[Geometry]) + "-ref", dir + infile, 35, ""}, pbsMaple)
	// submit opt, wait for it to finish in main goroutine - block
	Submit(dir + pbsfile)
	outfile := "ref.out"
	_, _, _, err := prog.ReadOut(dir + outfile)
	for err != nil {
		HandleSignal(35, time.Minute)
		E0, _, _, err = prog.ReadOut(dir + outfile)
		if (err == ErrEnergyNotParsed || err == ErrFinishedButNoEnergy ||
			err == ErrFileContainsError || err == ErrBlankOutput) ||
			err == ErrFileNotFound {

			fmt.Fprintln(os.Stderr, "resubmitting for", err)
			Submit(dir + pbsfile)
		}
	}
	return
}

// Frequency runs a Molpro harmonic frequency calculation in the freq
// directory
func Frequency(prog *Molpro, absPath string) ([]float64, bool) {
	// write freq.inp and that mp.pbs
	prog.WriteInput(absPath+"/freq.inp", freq)
	WritePBS(absPath+"/mp.pbs",
		&Job{MakeName(Input[Geometry]) + "-freq", absPath + "/freq.inp", 35, ""}, pbsMaple)
	// submit freq, wait in separate goroutine
	// doesn't matter if this finishes
	Submit(absPath + "/mp.pbs")
	outfile := absPath + "/freq.out"
	_, _, _, err := prog.ReadOut(outfile)
	for err != nil {
		HandleSignal(35, time.Minute)
		_, _, _, err = prog.ReadOut(outfile)
		// dont resubmit freq
		if err == ErrEnergyNotParsed || err == ErrFinishedButNoEnergy ||
			err == ErrFileContainsError {
			fmt.Fprintln(os.Stderr, "error in freq, aborting that calculation")
			return nil, false
		}
	}
	return prog.ReadFreqs(outfile), true
}

// Resubmit copies the input file associated with name to
// name_redo.inp, writes a new PBS file, submits the new PBS job, and
// returns the associated jobid
func Resubmit(name string, err error) string {
	fmt.Fprintf(os.Stderr, "resubmitting %s for %s\n", name, err)
	src, _ := os.Open(name + ".inp")
	dst, _ := os.Create(name + "_redo.inp")
	io.Copy(dst, src)
	defer func() {
		src.Close()
		dst.Close()
	}()
	WritePBS(name+"_redo.pbs", &Job{"redo", name + "_redo.inp", 35, ""}, pbsMaple)
	return Submit(name + "_redo.pbs")
}

// Drain drains the queue of jobs and receives on ch when ready for more
func Drain(prog *Molpro, ch chan Calc, E0 float64) (min, realTime float64) {
	start := time.Now()
	points := make([]Calc, 0)
	var (
		nJobs    int
		finished int
		resubs   int
		success  bool
		energy   float64
		err      error
		t        float64
		check    int = 1
	)
	heap := new(GarbageHeap)
	for {
		shortenBy := 0
		for i := 0; i < nJobs; i++ {
			job := points[i]
			if strings.Contains(job.Name, "E0") {
				energy = E0
				success = true
			} else if job.Result != 0 {
				energy = job.Result
				success = true
			} else if job.Src != nil {
				if job.Src.Len() > job.Src.Index && job.Src.Value() != 0 {
					energy = job.Src.Value()
					success = true
				}
			} else if energy, t, _, err = prog.ReadOut(job.Name + ".out"); err == nil {
				success = true
				if energy < min {
					min = energy
				}
				realTime += t
				heap.Add(job.Name)
				// job has not been resubmitted && there is an error
			} else if job.Resub == nil && (err == ErrEnergyNotParsed || err == ErrFinishedButNoEnergy ||
				err == ErrFileContainsError || err == ErrBlankOutput ||
				(err == ErrFileNotFound && CheckLog(job.cmdfile, job.Name) && CheckProg(job.cmdfile))) {
				if err == ErrFileContainsError {
					fmt.Fprintf(os.Stderr, "error: %v on %s\n", err, job.Name)
				}
				errMap[err]++
				// can't use job.whatever if you want to modify the thing
				points[i].Resub = &Calc{Name: job.Name + "_redo", ID: Resubmit(job.Name, err)}
				resubs++
				ptsJobs = append(ptsJobs, points[i].Resub.ID)
			} else if job.Resub != nil {
				// should DRY this up, inside if is same as case 3 above
				// should also check if resubmitted job has finished with qsub and set pointer to nil if it has without success
				if energy, t, _, err = prog.ReadOut(job.Resub.Name + ".out"); err == nil {
					success = true
					if energy < min {
						min = energy
					}
					realTime += t
					heap.Add(job.Name)
				}
			}
			if success {
				points[nJobs-1], points[i] = points[i], points[nJobs-1]
				nJobs--
				points = points[:nJobs]
				for _, t := range job.Targets {
					(*t.Slice)[t.Index].Add(t.Coeff * energy)
				}
				shortenBy++
				if !job.noRun {
					finished++
					check++
					paraCount[paraJobs[job.chunkNum]]--
					if paraCount[paraJobs[job.chunkNum]] == 0 {
						queueClear([]string{paraJobs[job.chunkNum]})
						if *debug {
							fmt.Printf("clearing paracount of chunk %d, jobid %s\n", job.chunkNum, paraJobs[job.chunkNum])
						}
					}
				}
				success = false
			}
		}
		if shortenBy < 1 {
			fmt.Fprintln(os.Stderr, "Didn't shorten, sleeping")
			time.Sleep(time.Second)
		}
		if check >= checkAfter {
			MakeCheckpoint()
			check = 1
		}
		if heap.Len() >= chunkSize {
			heap.Dump()
		}
		fmt.Fprintf(os.Stderr, "finished: %d of %d submitted\n", finished, submitted)
		// only receive more jobs if there is room
		if nJobs < jobLimit {
			calc, ok := <-ch
			if !ok && finished == submitted {
				fmt.Fprintf(os.Stderr, "resubmitted %d/%d (%.1f%%), points execution time: %v\n",
					resubs, submitted, float64(resubs)/float64(submitted)*100, time.Since(start))
				minutes := int(realTime) / 60
				secRem := realTime - 60*float64(minutes)
				fmt.Fprintf(os.Stderr, "total job time (wall): %.2f sec = %dm%.2fs\n", realTime, minutes, secRem)
				// this is deprecated now that all should be saved but leave to check
				if nDerivative == 4 {
					fmt.Fprintf(os.Stderr, "saved %d/%d (%.f%%) fourth derivative components from e2d\n",
						saved, fourTwos, float64(saved)/float64(fourTwos)*100)
				}
				for k, v := range errMap {
					fmt.Fprintf(os.Stderr, "%v: %d occurrences\n", k, v)
				}
				return
			} else if ok {
				points = append(points, calc)
				nJobs = len(points)
			}
		}
	}
	// unreachable
	return
}

// Qstat reports whether or not the job associated with jobid is
// running or queued
func Qstat(jobid string) bool {
	out, _ := exec.Command("qstat", jobid).Output()
	fields := strings.Fields(string(out))
	status := fields[len(fields)-2]
	if status == "R" || status == "Q" {
		return true
	}
	return false
}

// LookAhead looks at jobs around the given one to see if they have
// run yet
func LookAhead(jobname string, maxdepth int) bool {
	ext := filepath.Ext(jobname)
	endex := len(jobname) - len(ext) + 1
	strNum := jobname[endex:]
	num, _ := strconv.Atoi(strNum)
	if num+maxdepth > submitted {
		maxdepth = submitted
	}
	for i := num; i <= maxdepth; i++ {
		nextFile := fmt.Sprintf("%s%010d.out", jobname[:endex], i)
		if _, err := os.Stat(nextFile); !os.IsNotExist(err) {
			return true
		}
	}
	return false
}

// Clear the PBS queue of the pts jobs
func queueClear(jobs []string) error {
	for _, job := range jobs {
		var host string
		status, _ := exec.Command("qstat", "-f", job).Output()
		fields := strings.Fields(string(status))
		for f := range fields {
			if strings.Contains(fields[f], "exec_host") {
				host = strings.Split(fields[f+2], "/")[0]
				break
			}
		}
		if host != "" {
			out, err := exec.Command("ssh", host, "-t", "rm -rf /tmp/$USER/"+job+".maple").CombinedOutput()
			if *debug {
				fmt.Println("CombinedOutput and error from queueClear: ", string(out), err)
			}
		}
	}
	err := exec.Command("qdel", jobs...).Run()
	return err
}

func initialize() (prog *Molpro, intder *Intder, anpass *Anpass) {
	// parse flags for overwrite before mkdirs
	args := ParseFlags()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "pbqff: no input file supplied\n")
		os.Exit(1)
	}
	ParseInfile(args[0])
	if Input[Flags] == "noopt" {
		flags = flags &^ OPT
	}
	if Input[JobLimit] != "" {
		v, err := strconv.Atoi(Input[JobLimit])
		if err == nil {
			jobLimit = v
		}
	}
	if Input[ChunkSize] != "" {
		v, err := strconv.Atoi(Input[JobLimit])
		if err == nil {
			chunkSize = v
		}
	}
	if Input[Deriv] != "" {
		d, err := strconv.Atoi(Input[Deriv])
		if err != nil {
			panic(fmt.Sprintf("%v parsing derivative level input: %q\n", err, Input[Deriv]))
		}
		nDerivative = d
	}
	WhichCluster()
	switch Input[Program] {
	case "cccr":
		energyLine = regexp.MustCompile(`^\s*CCCRE\s+=`)
	case "gocart":
		flags |= CART
		energyLine = regexp.MustCompile(`energy=`)
	case "grad":
		flags |= GRAD
		// probably will not use energyline because it's going to be a bit different
	case "molpro", "": // default if not specified
		energyLine = regexp.MustCompile(`energy=`)
	default:
		errExit(fmt.Errorf("%s not implemented as a Program", Input[Program]), "")
	}
	mpName := "molpro.in"
	idName := "intder.in"
	apName := "anpass.in"
	prog, err := LoadMolpro("molpro.in")
	if err != nil {
		errExit(err, fmt.Sprintf("loading molpro input %q", mpName))
	}
	if !DoCart() {
		intder, err = LoadIntder("intder.in")
		if err != nil {
			errExit(err, fmt.Sprintf("loading intder input %q", idName))
		}
		anpass, err = LoadAnpass("anpass.in")
		if err != nil {
			errExit(err, fmt.Sprintf("loading anpass input %q", apName))
		}
	}
	MakeDirs(".")
	errMap = make(map[error]int)
	paraCount = make(map[string]int)
	nodes = PBSnodes()
	fmt.Printf("nodes: %q\n", nodes)
	return prog, intder, anpass
}

func errExit(err error, msg string) {
	fmt.Fprintf(os.Stderr, "pbqff: %v %s\n", err, msg)
	os.Exit(1)
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

// PrintFile15 prints the second derivative force constants in the
// format expected by SPECTRO
func PrintFile15(fc []CountFloat, natoms int, filename string) int {
	f, _ := os.Create(filename)
	fmt.Fprintf(f, "%5d%5d", natoms, 6*natoms) // still not sure why this is just times 6
	for i := range fc {
		if i%3 == 0 {
			fmt.Fprintf(f, "\n")
		}
		fmt.Fprintf(f, "%20.10f", fc[i].Val*fc2Scale)
	}
	fmt.Fprint(f, "\n")
	return len(fc)
}

// PrintFile30 prints the third derivative force constants in the
// format expected by SPECTRO
func PrintFile30(fc []CountFloat, natoms, other int, filename string) int {
	f, _ := os.Create(filename)
	fmt.Fprintf(f, "%5d%5d", natoms, other)
	for i := range fc {
		if i%3 == 0 {
			fmt.Fprintf(f, "\n")
		}
		fmt.Fprintf(f, "%20.10f", fc[i].Val*fc3Scale)
	}
	fmt.Fprint(f, "\n")
	return len(fc)
}

// PrintFile40 prints the fourth derivative force constants in the
// format expected by SPECTRO
func PrintFile40(fc []CountFloat, natoms, other int, filename string) int {
	f, _ := os.Create(filename)
	fmt.Fprintf(f, "%5d%5d", natoms, other)
	for i := range fc {
		if i%3 == 0 {
			fmt.Fprintf(f, "\n")
		}
		fmt.Fprintf(f, "%20.10f", fc[i].Val*fc4Scale)
	}
	fmt.Fprint(f, "\n")
	return len(fc)
}

// PrintE2D pretty prints the second derivative energy array
func PrintE2D() {
	for i := 0; i < len(e2d); i++ {
		if i%3 == 0 && i > 0 {
			fmt.Print("\n")
		}
		fmt.Printf("%20.12f", e2d[i].Val)
	}
	fmt.Print("\n")
}

var submitted int

func main() {
	// clear the queue if panicking
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("running queueClear before panic")
			queueClear(ptsJobs)
			panic(r)
		}
	}()
	// clear the queue if killed
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Signal(syscall.SIGTERM))
	go func() {
		<-c
		fmt.Println("running queueClear before SIGTERM")
		queueClear(ptsJobs)
		errExit(fmt.Errorf("received SIGTERM"), "")
	}()
	prog, intder, anpass := initialize()
	var (
		mpHarm    []float64
		finished  bool
		cart      string
		zmat      string
		err       error
		atomNames []string
		energies  []float64
		cenergies []CountFloat
		min       float64
		E0        float64
		natoms    int
	)

	if DoOpt() {
		prog.Geometry = FormatZmat(Input[Geometry])
		E0 = Optimize(prog)
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
				prog.BuildPoints("pts/file07", atomNames, &cenergies, ch, true)
			}()
			// this works if no points were deleted, else need a resume from checkpoint thing
		} else {
			prog.BuildPoints("pts/file07", atomNames, &cenergies, nil, false)
		}
	} else {
		names, coords := XYZGeom(Input[Geometry])
		natoms = len(names)
		prog.Geometry = Input[Geometry] + "\n}\n"
		if !DoOpt() {
			E0 = RefEnergy(prog)
		}
		go func() {
			prog.BuildCartPoints(names, coords, &fc2, &fc3, &fc4, ch)
		}()
	}
	// Instead of returning energies, use job.Target = energies, also need a function
	// for getting the index in the array for fc2,3,4 but do that before setting job.Index
	min, _ = Drain(prog, ch, E0)
	queueClear(ptsJobs)

	if !DoCart() {
		energies = FloatsFromCountFloats(cenergies)
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
	} else {
		N3N := natoms * 3 // from spectro manual pg 12
		other3 := N3N * (N3N + 1) * (N3N + 2) / 6
		other4 := N3N * (N3N + 1) * (N3N + 2) * (N3N + 3) / 24
		PrintFile15(fc2, natoms, "fort.15")
		if nDerivative > 2 {
			PrintFile30(fc3, natoms, other3, "fort.30")
		}
		if nDerivative > 3 {
			PrintFile40(fc4, natoms, other4, "fort.40")
		}
	}
	if *debug {
		PrintE2D()
	}
}
