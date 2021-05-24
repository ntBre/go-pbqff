/*
Push-button QFF
---------------
The goal of this program is to streamline the generation
of quartic force fields, automating as many pieces as possible.
(setq compile-command "go build . && scp -C pbqff woods:Programs/pbqff/.")
(my-recompile)
Copy to Programs directory
(progn (setq compile-command "go build . && scp -C pbqff woods:Programs/pbqff/.") (my-recompile))
Copy to home area so as not to disrupt running
(progn (setq compile-command "go build . && scp -C pbqff woods:") (my-recompile))

To decrease CPU usage increase sleepint input from default of 1 sec
and increase checkint from default 100 or disable entirely by setting
it to "no"
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

	"bytes"
	"io"
	"path"
	"runtime/pprof"

	"github.com/ntBre/chemutils/spectro"
	"github.com/ntBre/chemutils/summarize"
)

const (
	resBound = 1e-16 // warn if anpass residuals above this
	// this could  be in the input
	help = `Requirements (* denotes requirements for SICs only):
- pbqff input file minimally specifying the geometry and the
- paths to intder*, anpass*, and spectro executables
- template intder.in*, anpass.in*, spectro.in, and molpro.in files
  - intder.in should be a pts intder input and have the old geometry to serve as a template
  - anpass.in should be a first run anpass file, not a stationary point
  - spectro.in should not have any resonance information
  - molpro.in should have the geometry removed and have no closing brace to the geometry section
    - on sequoia, the custom energy parameter pbqff=energy is required for parsing
    - for gradients, use forces,varsav and show[f20.15],grad{x,y,z} to print the gradients
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

// DoCart is a helper function for checking whether the CART flag is
// set
func DoGrad() bool { return flags&GRAD > 0 }

// Flags
var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
	overwrite  = flag.Bool("o", false, "overwrite existing inp directory")
	pts        = flag.Bool("pts", false, "start by running pts on optimized geometry from opt")
	freqs      = flag.Bool("freqs", false, "start from running anpass on the pts output")
	debug      = flag.Bool("debug", false, "for debugging, print 2nd derivative energies array")
	checkpoint = flag.Bool("c", false, "resume from checkpoint")
	read       = flag.Bool("r", false, "read reference energy from pts/inp/ref.out")
	irdy       = flag.String("irdy", "",
		"intder file is ready to be used in pts; specify the atom order")
	count = flag.Bool("count", false,
		"read the input file and print the number of calculations needed then exit")
	nodel  = flag.Bool("nodel", false, "don't delete used output files")
	format = flag.Bool("fmt", false,
		"parse existing output files and print them in anpass format")
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
	paraJobs         []string // counters for parallel jobs
	paraCount        map[string]int
	errMap           map[error]int
	nodes            []string
	jobLimit         int = 1000
	chunkSize        int = 64
	checkAfter       int = 100
	sleep                = 1
	nocheck          bool
	flags            int
	delta            = 0.005   // default step size
	deltas           []float64 // slice for holding step sizes
	numJobs          int       = 8
	StartCPU         int64
)

// Finite differences denominators for cartesians
var (
	angbohr = 0.529177249
	// Going to get rid of all of these with divisors instead
	// TODO get rid of all of these denominators, can just delete
	fc2Scale = angbohr * angbohr / (4 * delta * delta)
	fc3Scale = angbohr * angbohr * angbohr / (8 * delta * delta * delta)
	fc4Scale = angbohr * angbohr * angbohr * angbohr / (16 * delta * delta * delta * delta)
)

// Finite differences denominators for gradients
var (
	gradFc2Scale = angbohr / (2 * delta)
	gradFc3Scale = angbohr * angbohr / (4 * delta * delta)
	gradFc4Scale = angbohr * angbohr * angbohr / (8 * delta * delta * delta)
)

// Cartesian arrays
var (
	fc2 []CountFloat
	fc3 []CountFloat
	fc4 []CountFloat
	e2d []CountFloat
)

// Errors
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
	Scale    float64
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
func (c *CountFloat) Add(t Target, scale float64, plus float64) {
	c.Val += plus
	c.Count--
	if c.Count < 0 {
		panic("added to CountFloat too many times")
	} else if c.Count == 0 && t.Slice != &e2d {
		c.Val *= scale
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
	g.heap = append(g.heap, basename+".inp", basename+".out")
}

// Len returns the length of g's underlying slice
func (g *GarbageHeap) Len() int {
	return len(g.heap)
}

// Dump deletes the globbed files in the heap using an appended *
func (g *GarbageHeap) Dump() {
	for _, f := range g.heap {
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
	if *format {
		FormatOutput("pts/inp/")
		os.Exit(0)
	}
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
		&Job{MakeName(Input[Geometry]) + "-opt", "opt/opt.inp",
			35, "", "", numJobs}, pbsMaple)
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
	outfile := "ref.out"
	E0, _, _, err := prog.ReadOut(dir + outfile)
	if *read && err == nil {
		return
	}

	if DoOpt() {
		prog.WriteInput(dir+infile, opt)
	} else {
		prog.WriteInput(dir+infile, none)
	}
	WritePBS(dir+pbsfile,
		&Job{MakeName(Input[Geometry]) + "-ref", dir + infile, 35, "", "", numJobs}, pbsMaple)
	// submit opt, wait for it to finish in main goroutine - block
	Submit(dir + pbsfile)
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
		&Job{MakeName(Input[Geometry]) + "-freq", absPath + "/freq.inp", 35, "", "", numJobs}, pbsMaple)
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
	WritePBS(name+"_redo.pbs", &Job{"redo", name + "_redo.inp", 35, "", "", numJobs}, pbsMaple)
	return Submit(name + "_redo.pbs")
}

// Drain drains the queue of jobs and receives on ch when ready for more
func Drain(prog *Molpro, ncoords int, ch chan Calc, E0 float64) (min, realTime float64) {
	start := time.Now()
	fmt.Println("step sizes: ", deltas)
	points := make([]Calc, 0)
	var (
		nJobs     int
		finished  int
		resubs    int
		success   bool
		energy    float64
		gradients []float64
		err       error
		t         float64
		check     int = 1
		norun     int
	)
	heap := new(GarbageHeap)
	maxjobs := jobLimit
	for {
		shortenBy := 0
		pollStart := time.Now()
		for i := 0; i < nJobs; i++ {
			job := points[i]
			if strings.Contains(job.Name, "E0") {
				energy = E0
				gradients = make([]float64, ncoords) // zero gradients at ref geom
				success = true
			} else if job.Result != 0 {
				energy = job.Result
				success = true
			} else if job.Src != nil {
				if job.Src.Len() > job.Src.Index && job.Src.Value() != 0 {
					energy = job.Src.Value()
					success = true
				}
			} else if energy, t, gradients, err = prog.ReadOut(job.Name + ".out"); err == nil {
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
				// THIS DOESNT CATCH FILE EXISTS BUT IS HUNG
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
				// should also check if resubmitted
				// job has finished with qsub and set
				// pointer to nil if it has without
				// success
				if energy, t, gradients, err = prog.ReadOut(job.Resub.Name + ".out"); err == nil {
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
				if !DoGrad() {
					for _, t := range job.Targets {
						(*t.Slice)[t.Index].Add(t, job.Scale, t.Coeff*energy)
					}
				} else {
					// Targets line up with gradients
					for g := range job.Targets {
						(*job.Targets[g].Slice)[job.Targets[g].Index].Add(job.Targets[g],
							job.Scale, job.Targets[0].Coeff*gradients[g])
					}
				}
				shortenBy++
				if !job.noRun {
					finished++
					check++
					paraCount[paraJobs[job.chunkNum]]--
					if paraCount[paraJobs[job.chunkNum]] == 0 {
						queueClear([]string{paraJobs[job.chunkNum]})
						if *debug {
							fmt.Printf("clearing paracount of chunk %d, jobid %s\n",
								job.chunkNum, paraJobs[job.chunkNum])
						}
					}
				} else {
					norun--
				}
				success = false
			}
		}
		if shortenBy < 1 {
			fmt.Fprintln(os.Stderr, "Didn't shorten, sleeping")
			time.Sleep(time.Duration(sleep) * time.Second)
		}
		if check >= checkAfter {
			if !nocheck {
				MakeCheckpoint()
			}
			check = 1
			fmt.Fprintf(os.Stderr, "CPU time: %.3f s\n", float64(GetCPU()-StartCPU)/1e9)
		}
		if heap.Len() >= chunkSize && !*nodel {
			heap.Dump()
		}
		// Progress
		fmt.Fprintf(os.Stderr, "finished %d/%d submitted, %v polling %d jobs\n", finished, submitted,
			time.Since(pollStart).Round(time.Millisecond), nJobs-norun)
		// only receive more jobs if there is room
		for count := 0; count < chunkSize && nJobs < maxjobs+norun; count++ {
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
				if calc.noRun {
					norun++
				}
			} else if !ok {
				nJobs = len(points)
				break
			}
		}
	}
}

// Unused
// Qstat reports whether or not the job associated with jobid is
// running or queued
func Qstat(jobid string, statuses ...string) bool {
	out, _ := exec.Command("qstat", jobid).Output()
	fields := strings.Fields(string(out))
	if len(fields) >= 2 {
		test := fields[len(fields)-2]
		for _, status := range statuses {
			if test == status {
				return true
			}
		}
	}
	return false
}

// Unused
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

// ParseDeltas parses a sequence of step sizes input as a string into
// a slice of floats
func ParseDeltas(inp string) (ncoords int, out []float64, err error) {
	// assume problem
	err = errors.New("invalid deltas input")
	geom := strings.Split(Input[Geometry], "\n")
	if Input[GeomType] == "xyz" {
		ncoords = 3 * (len(geom) - 2)
	} else {
		ncoords = len(geom)
	}
	out = make([]float64, ncoords)
	// set up defaults
	for i := range out {
		out[i] = delta
	}
	if len(inp) == 0 {
		err = nil
		return
	}
	pairs := strings.Split(inp, ",")
	for _, p := range pairs {
		sp := strings.Split(p, ":")
		if len(sp) != 2 {
			return
		}
		d, e := strconv.Atoi(strings.TrimSpace(sp[0]))
		if e != nil || d > ncoords || d < 1 {
			return
		}
		f, e := strconv.ParseFloat(strings.TrimSpace(sp[1]), 64)
		if e != nil || f < 0.0 {
			return
		}
		out[d-1] = f
	}
	err = nil
	return
}

func totalPoints(n int) int {
	return 2 * n * (n*n*n + 2*n*n + 8*n + 1) / 3
}

func initialize() (prog *Molpro, intder *Intder, anpass *Anpass) {
	// parse flags for overwrite before mkdirs
	args := ParseFlags()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "pbqff: no input file supplied\n")
		os.Exit(1)
	}
	infile := args[0]
	// set up output and err files and dup their fds to stdout and stderr
	// https://github.com/golang/go/issues/325
	base := infile[:len(infile)-len(path.Ext(infile))]
	outfile, _ := os.Create(base + ".out")
	errfile, _ := os.Create(base + ".err")
	syscall.Dup2(int(outfile.Fd()), 1)
	syscall.Dup2(int(errfile.Fd()), 2)
	ParseInfile(infile)
	if Input[Flags] == "noopt" {
		flags = flags &^ OPT
	}
	spectro.SpectroCommand = Input[SpectroCmd]
	if Input[JobLimit] != "" {
		v, err := strconv.Atoi(Input[JobLimit])
		if err == nil {
			jobLimit = v
		}
	}
	if Input[ChunkSize] != "" {
		v, err := strconv.Atoi(Input[ChunkSize])
		if err == nil {
			chunkSize = v
		}
	}
	if Input[Deriv] != "" {
		d, err := strconv.Atoi(Input[Deriv])
		if err != nil {
			panic(fmt.Sprintf("%v parsing derivative level input: %q\n",
				err, Input[Deriv]))
		}
		nDerivative = d
	}
	if Input[NumJobs] != "" {
		d, err := strconv.Atoi(Input[NumJobs])
		if err != nil {
			panic(fmt.Sprintf("%v parsing number of jobs input: %q\n",
				err, Input[NumJobs]))
		}
		numJobs = d
	}
	if s := Input[SleepInt]; s != "" {
		d, err := strconv.Atoi(s)
		if err != nil {
			panic(fmt.Sprintf("%v parsing sleep interval: %q\n", err, s))
		}
		sleep = d
	}
	switch Input[CheckInt] {
	case "no":
		nocheck = true
	case "":
	default:
		d, err := strconv.Atoi(Input[CheckInt])
		if err != nil {
			panic(fmt.Sprintf("%v parsing checkpoint interval: %q\n",
				err, Input[CheckInt]))
		}
		checkAfter = d
	}

	if Input[Delta] != "" {
		f, err := strconv.ParseFloat(Input[Delta], 64)
		if err != nil {
			panic(fmt.Sprintf("%v parsing delta input: %q\n", err, Input[Delta]))
		}
		delta = f
	}
	// always parse deltas to fill with default even if no input
	ncoords, f, err := ParseDeltas(Input[Deltas])
	if err != nil {
		panic(fmt.Sprintf("%v parsing deltas input: %q\n", err, Input[Deltas]))
	}
	deltas = f
	WhichCluster()
	switch Input[Program] {
	case "cccr":
		energyLine = regexp.MustCompile(`^\s*CCCRE\s+=`)
	case "cart", "gocart":
		flags |= CART
		fmt.Printf("%d coords requires %d points\n", ncoords, totalPoints(ncoords))
		energyLine = regexp.MustCompile(`energy=`)
	case "grad":
		flags |= GRAD
		energyLine = regexp.MustCompile(`energy=`)
	case "molpro", "": // default if not specified
		energyLine = regexp.MustCompile(`energy=`)
	default:
		errExit(fmt.Errorf("%s not implemented as a Program", Input[Program]), "")
	}
	if *count {
		if !DoCart() {
			fmt.Println("-count only implemented for Cartesians")
		}
		os.Exit(0)
	}
	mpName := "molpro.in"
	idName := "intder.in"
	apName := "anpass.in"
	prog, err = LoadMolpro("molpro.in")
	if err != nil {
		errExit(err, fmt.Sprintf("loading molpro input %q", mpName))
	}
	if !(DoCart() || DoGrad()) {
		intder, err = LoadIntder("intder.in")
		if err != nil {
			errExit(err, fmt.Sprintf("loading intder input %q", idName))
		}
		anpass, err = LoadAnpass("anpass.in")
		if err != nil {
			errExit(err, fmt.Sprintf("loading anpass input %q", apName))
		}
	}
	if !*read {
		MakeDirs(".")
	}
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

// TODO these are all the same besides the 6*natoms in 15

// PrintFile15 prints the second derivative force constants in the
// format expected by SPECTRO
func PrintFile15(fc []CountFloat, natoms int, filename string) int {
	f, _ := os.Create(filename)
	fmt.Fprintf(f, "%5d%5d", natoms, 6*natoms) // still not sure why this is just times 6
	for i := range fc {
		if i%3 == 0 {
			fmt.Fprintf(f, "\n")
		}
		fmt.Fprintf(f, "%20.10f", fc[i].Val)
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
		fmt.Fprintf(f, "%20.10f", fc[i].Val)
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
		fmt.Fprintf(f, "%20.10f", fc[i].Val)
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

var submitted int

func main() {
	StartCPU = GetCPU()
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
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	cur, max := GetCPULimit() // run after initialize so output goes to file
	fmt.Printf("Maximum CPU time (s):\n\tCur: %d\n\tMax: %d\n", cur, max)
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
		ncoords   int
		names     []string
		coords    []float64
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
		// run the frequency in the background and don't wait
		go func() {
			absPath, _ := filepath.Abs("freq")
			mpHarm, finished = Frequency(prog, absPath)
		}()
	} else {
		cart = Input[Geometry]
	}

	ch := make(chan Calc, jobLimit)

	if !(DoCart() || DoGrad()) {
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
			// this works if no points were deleted, else
			// need a resume from checkpoint thing
		} else {
			prog.BuildPoints("pts/file07", atomNames, &cenergies, nil, false)
		}
	} else {
		names, coords = XYZGeom(Input[Geometry])
		natoms = len(names)
		ncoords = len(coords)
		prog.Geometry = Input[Geometry] + "\n}\n"
		if !DoOpt() {
			E0 = RefEnergy(prog)
		}
		if DoCart() {
			go func() {
				prog.BuildCartPoints(names, coords, &fc2, &fc3, &fc4, ch)
			}()
		} else if DoGrad() {
			go func() {
				prog.BuildGradPoints(names, coords, &fc2, &fc3, &fc4, ch)
			}()
		}
	}

	min, _ = Drain(prog, ncoords, ch, E0)
	queueClear(ptsJobs)

	if !(DoCart() || DoGrad()) {
		energies = FloatsFromCountFloats(cenergies)
		// convert to relative energies
		for i := range energies {
			energies[i] -= min
		}
		longLine := DoAnpass(anpass, energies)
		coords, intderHarms := DoIntder(intder, atomNames, longLine)
		spec, err := spectro.Load("spectro.in")
		if err != nil {
			errExit(err, "loading spectro input")
		}
		spec.FormatGeom(atomNames, coords)
		spec.WriteInput("freqs/spectro.in")
		err = spec.DoSpectro("freqs/")
		if err != nil {
			errExit(err, "running spectro")
		}
		if !finished {
			mpHarm = make([]float64, spec.Nfreqs)
		}
		res := summarize.Spectro(filepath.Join("freqs", "spectro2.out"))
		Summarize(res.ZPT, mpHarm, intderHarms, res.Harm, res.Fund, res.Corr)
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
			var buf bytes.Buffer
			for i := range coords {
				if i%3 == 0 && i > 0 {
					fmt.Fprint(&buf, "\n")
				}
				fmt.Fprintf(&buf, " %.10f", coords[i]/angbohr)
			}
			spec, err := spectro.Load("spectro.in")
			if err != nil {
				errExit(err, "loading spectro input")
			}
			spec.FormatGeom(names, buf.String())
			spec.WriteInput("spectro.in")
			err = spec.DoSpectro(".")
			if err != nil {
				errExit(err, "running spectro")
			}
			res := summarize.Spectro("spectro2.out")
			// fill molpro and intder freqs slots with empty slices
			nfreqs := len(res.Harm)
			err = Summarize(res.ZPT, make([]float64, nfreqs),
				make([]float64, nfreqs), res.Harm, res.Fund, res.Corr)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
	if *debug {
		PrintE2D()
	}
	fmt.Printf("total CPU time used: %.3f s\n", float64(GetCPU()-StartCPU)/1e9)
}
