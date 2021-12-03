/*
Push-button QFF
---------------
The goal of this program is to streamline the generation of quartic
force fields, automating as many pieces as possible. To decrease CPU
usage increase sleepint input from default of 1 sec and increase
checkint from default 100 or disable entirely by setting it to "no"
*/

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"

	"path/filepath"

	"strconv"

	"bytes"
	"path"
	"runtime/pprof"

	rtdebug "runtime/debug"

	"runtime"

	"github.com/ntBre/chemutils/spectro"
	"github.com/ntBre/chemutils/summarize"
	symm "github.com/ntBre/chemutils/symmetry"
)

// Flags for the procedures to be run
const (
	OPT int = 1 << iota
	PTS
	CART
	GRAD
	FREQS
)

// I hate these

// DoOpt is a helper function for checking whether the OPT flag is set
func DoOpt() bool { return flags&OPT > 0 }

// DoFreqs is a helper function for checking whether the FREQS flag is
// set
func DoFreqs() bool { return flags&FREQS > 0 }

// DoCart is a helper function for checking whether the CART flag is
// set
func DoCart() bool { return flags&CART > 0 }

// DoGrad is a helper function for checking whether the CART flag is
// set
func DoGrad() bool { return flags&GRAD > 0 }

// DoSIC if neither CART or GRAD
func DoSIC() bool { return flags&(CART|GRAD) == 0 }

// Global variables
var (
	brokenFloat      = math.NaN()
	molproTerminated = "Molpro calculation terminated"
	ptsJobs          []string
	paraJobs         []string // counters for parallel jobs
	paraCount        map[string]int
	errMap           map[error]int
	nocheck          bool
	flags            int
	submitted        int
	StartCPU         int64
	Conf             = NewConfig()
	ErrorLine        = regexp.MustCompile(`(?i)[^_]error`)
	GaussErrorLine   = regexp.MustCompile(`(?i)error termination`)
	OutExt           = ".out"
)

// Global is a structure for holding global variables
var Global struct {
	Nodes    []string
	JobNum   int
	Warnings int
}

// HashName returns a hashed filename. Well it used to, but now it
// returns JobNum and increments it
func HashName() string {
	defer func() {
		Global.JobNum++
	}()
	return fmt.Sprintf("job.%010d", Global.JobNum)
}

const (
	angbohr  = 0.529177249 // angstrom per bohr
	resBound = 1e-16       // warn if anpass residuals above this
)

// Cartesian arrays
var (
	fc2 []CountFloat
	fc3 []CountFloat
	fc4 []CountFloat
	e2d []CountFloat
)

// SIC array
var (
	cenergies []CountFloat
)

// Errors
var (
	ErrBlankOutput         = errors.New("Molpro output file exists but is blank")
	ErrEnergyNotFound      = errors.New("Energy not found in Molpro output")
	ErrEnergyNotParsed     = errors.New("Energy not parsed in Molpro output")
	ErrFileContainsError   = errors.New("Molpro output file contains an error")
	ErrFileNotFound        = errors.New("Molpro output file not found")
	ErrFinishedButNoEnergy = errors.New("Molpro output finished but no energy found")
	ErrInputGeomNotFound   = errors.New("Geometry not found in input file")
	ErrTimeout             = errors.New("Timeout waiting for signal")
)

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

// Drain drains the queue of jobs and receives on ch when ready for
// more. prog is only used for its ReadOut method, and ncoords is used
// to construct the zero gradient array.
func Drain(prog Program, q Queue, ncoords int, E0 float64,
	gen func() ([]Calc, bool)) (min, realTime float64) {
	start := time.Now()
	if Conf.At(Deltas) != nil {
		fmt.Println("step sizes: ", Conf.FlSlice(Deltas))
	}
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
		dumper    *os.File
	)
	qstat := make(map[string]bool)
	heap := new(GarbageHeap)
	ok := true
	var calcs []Calc
	maxjobs := Conf.Int(JobLimit)
	if *dump {
		f, err := os.Create("dump.dat")
		defer f.Close()
		if err != nil {
			panic(err)
		}
		dumper = f
	}
	for {
		for maxjobs+norun-nJobs >= Conf.Int(ChunkSize) && ok {
			calcs, ok = gen()
			points = append(points, calcs...)
			nJobs = len(points)
			for _, c := range calcs {
				if c.noRun {
					norun++
				} else {
					// default to true and only
					// check when no jobs finish
					qstat[c.JobID] = true
				}
			}
		}
		shortenBy := 0
		pollStart := time.Now()
		for i := 0; i < nJobs; i++ {
			job := points[i]
			if strings.Contains(job.Name, "E0") {
				energy = E0
				// zero gradients at ref geom
				gradients = make([]float64, ncoords)
				success = true
			} else if job.Result != 0 {
				energy = job.Result
				success = true
			} else if job.Src != nil {
				if job.Src.Len() > job.Src.Index && job.Src.Value() != 0 {
					energy = job.Src.Value()
					success = true
				}
			} else if energy, t, gradients,
				err = prog.ReadOut(job.Name + OutExt); err == nil {
				success = true
				if energy < min {
					min = energy
				}
				realTime += t
				heap.Add(job.Name)
				// job has not been resubmitted && there is an error
			} else if job.Resub == nil &&
				(err == ErrEnergyNotParsed || err == ErrFinishedButNoEnergy ||
					err == ErrFileContainsError || err == ErrBlankOutput ||
					(err == ErrFileNotFound && !qstat[job.JobID])) {
				// THIS DOESNT CATCH FILE EXISTS BUT IS HUNG
				if err == ErrFileContainsError {
					fmt.Fprintf(os.Stderr,
						"error: %v on %s\n", err, job.Name)
				}
				errMap[err]++
				// can't use job.whatever if you want to modify the thing
				points[i].Resub = &Calc{
					Name:    job.Name + "_redo",
					ResubID: q.Resubmit(job.Name, err),
				}
				resubs++
				ptsJobs = append(ptsJobs, points[i].Resub.ResubID)
			} else if job.Resub != nil {
				if energy, t, gradients,
					err = prog.ReadOut(job.Resub.Name +
					OutExt); err == nil {
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
				if *dump {
					for _, c := range job.Coords {
						fmt.Fprintf(dumper, "%15.10f", c)
					}
					fmt.Fprintf(dumper, "%20.12f\n", energy)
				}
				if !DoGrad() {
					for _, t := range job.Targets {
						(*t.Slice)[t.Index].Add(t,
							job.Scale, t.Coeff*energy)
					}
				} else {
					// Targets line up with gradients
					for g := range job.Targets {
						(*job.Targets[g].Slice)[job.Targets[g].Index].
							Add(job.Targets[g],
								job.Scale,
								job.Targets[0].
									Coeff*gradients[g])
					}
				}
				shortenBy++
				if !job.noRun {
					finished++
					check++
					paraCount[paraJobs[job.ChunkNum]]--
					if paraCount[paraJobs[job.ChunkNum]] == 0 {
						// queueClear([]string{paraJobs[job.ChunkNum]})
						if *debug {
							fmt.Printf("clearing paracount of"+
								"chunk %d, jobid %s\n",
								job.ChunkNum,
								paraJobs[job.ChunkNum])
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
			q.Stat(&qstat)
			time.Sleep(time.Duration(Conf.Int(SleepInt)) * time.Second)
		} else {
			fmt.Fprintf(os.Stderr,
				"finished %d/%d submitted, %v polling %d jobs\n",
				finished, submitted,
				time.Since(pollStart).Round(time.Millisecond), nJobs-norun)
		}
		if check >= Conf.Int(CheckInt) {
			if !nocheck {
				MakeCheckpoint(prog.GetDir())
			}
			check = 1
			fmt.Fprintf(os.Stderr, "CPU time: %.3f s\n",
				float64(GetCPU()-StartCPU)/1e9)
		}
		if heap.Len() >= Conf.Int(ChunkSize) && !*nodel {
			heap.Dump()
			stackDump()
		}
		// Termination
		if nJobs == 0 {
			fmt.Printf("resubmitted %d/%d (%.1f%%),"+
				" points execution time: %v\n",
				resubs, submitted,
				float64(resubs)/float64(submitted)*100,
				time.Since(start))
			minutes := int(realTime) / 60
			secRem := realTime - 60*float64(minutes)
			fmt.Printf("total job time (wall): %.2f sec = %dm%.2fs\n",
				realTime, minutes, secRem)
			for k, v := range errMap {
				fmt.Printf("%v: %d occurrences\n", k, v)
			}
			return
		}
	}
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

// RunFreqs runs the frequency portion of the QFF starting from anpass
// in the current directory
func RunFreqs(intder *Intder, anp *Anpass) {
	energies := make([]float64, 0)
	f, err := os.Open("rel.dat")
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 1 {
			v, err := strconv.ParseFloat(fields[0], 64)
			if err != nil {
				panic(err)
			}
			energies = append(energies, v)
		}
	}
	prog := new(Molpro)
	err = prog.FormatCart(Conf.Str(Geometry))
	if err != nil {
		panic(err)
	}
	cart := prog.GetGeom()
	// only required for cartesians
	names := intder.ConvertCart(cart)
	longLine, lin := DoAnpass(anp, ".", energies, intder)
	intder.WriteGeom("intder_geom.in", longLine)
	RunIntder("intder_geom")
	coords := intder.ReadGeom("intder_geom.out")
	// if triatomic and linear
	intder.Read9903("fort.9903", len(names) == 3 && lin)
	intder.WriteFreqs("intder.in", names, len(names) == 3 && lin)
	RunIntder("intder")
	intderHarms := intder.ReadOut("intder.out")
	err = os.Rename("file15", "fort.15")
	if err == nil {
		err = os.Rename("file20", "fort.30")
	}
	if err == nil {
		err = os.Rename("file24", "fort.40")
	}
	if err != nil {
		panic(err)
	}
	spec, err := spectro.Load("spectro.in")
	if err != nil {
		errExit(err, "loading spectro input")
	}
	spec.FormatGeom(names, coords)
	err = spec.WriteInput("spectro.in")
	if err != nil {
		panic(err)
	}
	err = spec.DoSpectro(".")
	if err != nil {
		errExit(err, "running spectro")
	}
	res := summarize.SpectroFile("spectro2.out")
	var mpHarm []float64
	if mpHarm == nil || len(mpHarm) < len(res.Harm) {
		mpHarm = make([]float64, spec.Nfreqs)
	}
	Summarize(os.Stdout, res.ZPT, mpHarm, intderHarms, res.Harm, res.Fund, res.Corr)
}

func initialize(infile string) (prog Program, intder *Intder, anpass *Anpass) {
	if !*test {
		DupOutErr(infile)
	}
	if *freqs {
		ParseInfile(infile)
		spectro.Command = Conf.Str(SpectroCmd)
		var err error
		intder, err = LoadIntder("intder.in")
		if err != nil {
			errExit(err, "loading intder.in")
		}
		anpass, err = LoadAnpass("anpass.in")
		if err != nil {
			errExit(err, "loading anpass.in")
		}
		RunFreqs(intder, anpass)
		os.Exit(0)
	}
	dir := filepath.Dir(infile)
	ParseInfile(infile)
	spectro.Command = Conf.Str(SpectroCmd)
	nc := Conf.Int(Ncoords)
	switch {
	case DoCart():
		fmt.Printf("%d coords requires %d Cartesian points\n",
			nc, CartPoints(nc))
		if *count {
			os.Exit(0)
		}
	case DoGrad():
		fmt.Printf("%d coords requires %d gradient points\n",
			nc, GradPoints(nc))
		if *count {
			os.Exit(0)
		}
	case *count:
		fmt.Println("-count only implemented for gradients and Cartesians")
		os.Exit(1)
	}
	mpName := filepath.Join(dir, Conf.Str(MolproTmpl))
	idName := filepath.Join(dir, Conf.Str(IntderTmpl))
	apName := filepath.Join(dir, Conf.Str(AnpassTmpl))
	var err error
	switch Conf.Str(Package) {
	case "molpro", "":
		prog, err = LoadMolpro(mpName)
	case "g16", "gaussian", "gauss":
		prog, err = LoadGaussian(mpName)
		ptsMaple = ptsMapleGauss
		pbsMaple = pbsMapleGauss
		OutExt = ".log"
	}
	if err != nil {
		errExit(err, fmt.Sprintf("loading molpro input %q", mpName))
	}
	prog.SetDir(dir)
	if DoSIC() {
		intder, err = LoadIntder(filepath.Join(dir, "intder.in"))
		if err != nil {
			errExit(err, fmt.Sprintf("loading intder input %q", idName))
		}
		anpass, err = LoadAnpass(filepath.Join(dir, "anpass.in"))
		if err != nil {
			errExit(err, fmt.Sprintf("loading anpass input %q", apName))
		}
	}
	if !*read {
		MakeDirs(dir)
	}
	errMap = make(map[error]int)
	paraCount = make(map[string]int)
	Global.Nodes = PBSnodes()
	return prog, intder, anpass
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

func main() {
	StartCPU = GetCPU()
	defer CatchPanic()
	go CatchKill()
	args := ParseFlags()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "pbqff: no input file supplied\n")
		os.Exit(1)
	}
	infile := args[0]
	prog, intder, anpass := initialize(infile)
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	// run after initialize so output goes to file
	// TODO is this part of the initialization?
	fmt.Printf("pbqff version: %s\ncompiled at %s\n", VERSION, COMP_TIME)
	fmt.Printf("\nRun started at %s under PID %d\n",
		time.Now().Format("Mon Jan 2, 2006 at 15:04:05"), os.Getpid())
	cur, _ := GetCPULimit()
	fmt.Printf("Maximum CPU time (s): %d\n", cur)
	if *maxthreads >= 1 {
		fmt.Printf("Maximum number of threads: %d\n", *maxthreads)
		rtdebug.SetMaxThreads(*maxthreads)
	}
	if *maxprocs >= 1 {
		fmt.Printf("Maximum number of simultaneous CPUs: %d\n", *maxprocs)
		runtime.GOMAXPROCS(*maxprocs)
	}
	fmt.Printf("Available nodes: %q\n\n", Global.Nodes)
	var (
		mpHarm   []float64
		cart     string
		zmat     string
		energies []float64
		min      float64
		E0       float64
		natoms   int
		ncoords  int
		names    []string
		coords   []float64
		queue    Queue
	)

	switch Conf.Str(QueueSystem) {
	case "pbs":
		queue = PBS{
			SinglePt: pbsMaple,
			ChunkPts: ptsMaple,
		}
	case "slurm":
		queue = Slurm{
			SinglePt: pbsSlurm,
			ChunkPts: ptsSlurm,
		}
	}

	if DoOpt() {
		if Conf.Str(GeomType) != "zmat" {
			panic("optimization requires a zmat geometry")
		}
		err := prog.FormatZmat(Conf.Str(Geometry))
		if err != nil {
			panic(err)
		}
		E0 = prog.Run(opt, queue)
		cart, zmat, err = prog.HandleOutput("opt/opt")
		if err != nil {
			panic(err)
		}
		prog.UpdateZmat(zmat)
		prog.Run(freq, queue)
	} else {
		if !strings.Contains("cart,xyz", Conf.Str(GeomType)) {
			panic("expecting cartesian geometry")
		}
		err := prog.FormatCart(Conf.Str(Geometry))
		if err != nil {
			panic(err)
		}
		cart = prog.GetGeom()
		// only required for cartesians
		if DoCart() {
			E0 = prog.Run(none, queue)
		}
	}
	mol := symm.ReadXYZ(strings.NewReader(cart))
	fmt.Printf("Point group %s\n", mol.Group)

	var gen func() ([]Calc, bool)

	if DoSIC() {
		if *irdy == "" {
			names = intder.ConvertCart(cart)
		} else {
			names = strings.Fields(*irdy)
		}
		intder.WritePts("pts/intder.in")
		RunIntder("pts/intder")
		gen = BuildPoints(prog, queue, "pts/file07", names, true)
	} else {
		names, coords = XYZGeom(cart)
		natoms = len(names)
		ncoords = len(coords)
		if DoCart() {
			gen = BuildCartPoints(prog, queue, "pts/inp", names, coords, mol)
		} else if DoGrad() {
			gen = BuildGradPoints(prog, queue, "pts/inp", names, coords, mol)
		}
	}

	min, _ = Drain(prog, queue, ncoords, E0, gen)
	queueClear(ptsJobs)

	if DoSIC() {
		energies = FloatsFromCountFloats(cenergies)
		f, err := os.Create(filepath.Join(prog.GetDir(), "rel.dat"))
		if err != nil {
			// just dump the raw energies in a worst-case
			// scenario
			fmt.Println(energies)
		}
		// convert to relative energies
		for i := range energies {
			energies[i] -= min
			fmt.Fprintf(f, "%20.12f\n", energies[i])
		}
		f.Close()
		longLine, lin := DoAnpass(anpass,
			filepath.Join(prog.GetDir(), "freqs"), energies, intder)
		coords, intderHarms := DoIntder(intder, names, longLine,
			prog.GetDir(), lin)
		spec, err := spectro.Load("spectro.in")
		if err != nil {
			errExit(err, "loading spectro input")
		}
		spec.FormatGeom(names, coords)
		spec.WriteInput("freqs/spectro.in")
		err = spec.DoSpectro("freqs/")
		if err != nil {
			errExit(err, "running spectro")
		}
		mpHarm = prog.ReadFreqs("freqs/freq.out")
		res := summarize.SpectroFile(filepath.Join("freqs", "spectro2.out"))
		if mpHarm == nil || len(mpHarm) < len(res.Harm) {
			mpHarm = make([]float64, spec.Nfreqs)
		}
		Summarize(os.Stdout, res.ZPT, mpHarm, intderHarms, res.Harm, res.Fund, res.Corr)
	} else {
		N3N := natoms * 3 // from spectro manual pg 12
		other3 := N3N * (N3N + 1) * (N3N + 2) / 6
		other4 := N3N * (N3N + 1) * (N3N + 2) * (N3N + 3) / 24
		PrintFortFile(fc2, natoms, 6*natoms, "fort.15")
		if Conf.Int(Deriv) > 2 {
			PrintFortFile(fc3, natoms, other3, "fort.30")
		}
		if Conf.Int(Deriv) > 3 {
			PrintFortFile(fc4, natoms, other4, "fort.40")
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
			res := summarize.SpectroFile("spectro2.out")
			// fill molpro and intder freqs slots with empty slices
			nfreqs := len(res.Harm)
			err = Summarize(os.Stdout, res.ZPT, make([]float64, nfreqs),
				make([]float64, nfreqs), res.Harm, res.Fund, res.Corr)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
	if *debug {
		PrettyPrint(e2d)
	}
	fmt.Printf("total CPU time used: %.3f s\n", float64(GetCPU()-StartCPU)/1e9)
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			panic(err)
		}
		pprof.WriteHeapProfile(f)
		f.Close()
	}
	if Global.Warnings > 0 {
		fmt.Printf("pbqff terminated with %d warnings\n", Global.Warnings)
	} else {
		fmt.Println("\nNormal termination of pbqff")
	}
}
