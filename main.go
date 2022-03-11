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
	"os"
	"strings"
	"time"

	"path/filepath"

	"strconv"

	"bytes"
	"runtime/pprof"

	rtdebug "runtime/debug"

	"runtime"

	anp "github.com/ntBre/anpass"
	"github.com/ntBre/chemutils/spectro"
	"github.com/ntBre/chemutils/summarize"
	symm "github.com/ntBre/chemutils/symmetry"
	"gonum.org/v1/gonum/mat"
)

// Flags for the procedures to be run
var (
	OPT   bool
	PTS   bool
	SIC   bool
	CART  bool
	GRAD  bool
	FREQS bool
)

// Global variables
var (
	Conf Config
)

// Global is a struct for holding global variables
var Global struct {
	ErrMap      map[error]int
	Nodes       []string
	WatchedJobs []string
	JobNum      int
	Submitted   int
	Warnings    int
	StartCPU    int64
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
	ANGBOHR  = 0.529177249 // angstrom per bohr
	KCALHT   = 627.5091809 // kcal/mol per hartree
	resBound = 1e-16       // warn if anpass residuals above this
)

// Cartesian arrays
var (
	fc2 []CountFloat
	fc3 []CountFloat
	fc4 []CountFloat
)

// initArrays initializes the global force constant and second
// derivative arrays to the right size based on the number of
// atoms. The formulas for the dimensions of the arrays are from the
// SPECTRO manual on page 12
func initArrays(natoms int) (int, int) {
	N3N := natoms * 3
	other3 := N3N * (N3N + 1) * (N3N + 2) / 6
	other4 := N3N * (N3N + 1) * (N3N + 2) * (N3N + 3) / 24
	fc2 = make([]CountFloat, N3N*N3N)
	fc3 = make([]CountFloat, other3)
	fc4 = make([]CountFloat, other4)
	Table = make(BigHash)
	return other3, other4
}

// SIC array
var (
	cenergies []CountFloat
)

// Errors
var (
	ErrBlankOutput         = errors.New("Molpro output file exists but is blank")
	ErrEnergyNotFound      = errors.New("Energy not found in Molpro output")
	ErrFileContainsError   = errors.New("Molpro output file contains an error")
	ErrFileNotFound        = errors.New("Molpro output file not found")
	ErrGaussNotFound       = errors.New("Gaussan output file not found")
	ErrFinishedButNoEnergy = errors.New("Molpro output finished but no energy found")
)

// Drain drains the queue of jobs and receives on ch when ready for
// more. prog is only used for its ReadOut method, and ncoords is used
// to construct the zero gradient array.
func Drain(prog Program, q Queue, ncoords int, E0 float64,
	gen func() ([]Calc, bool)) (min, realTime float64) {
	min = 1 // to work with semi-empirical methods with positives
	start := time.Now()
	if Conf.Deltas != nil {
		fmt.Println("step sizes: ", Conf.Deltas)
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
	maxjobs := Conf.JobLimit
	if *dump {
		f, err := os.Create("dump.dat")
		defer f.Close()
		if err != nil {
			panic(err)
		}
		dumper = f
	}
	for {
		for maxjobs+norun-nJobs >= Conf.ChunkSize && ok {
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
			} else if job.Src != nil && job.Src.Status == Done {
				energy = job.Src.Value
				success = true
			} else if energy, t, gradients,
				err = prog.ReadOut(job.Name + ".out"); err == nil {
				success = true
				if energy < min {
					min = energy
				}
				realTime += t
				heap.Add(job.Name)
				// job has not been resubmitted && there is an error
			} else if !job.noRun &&
				(err == ErrFinishedButNoEnergy || err ==
					ErrFileContainsError || err ==
					ErrBlankOutput || (err ==
					ErrFileNotFound && !qstat[job.JobID])) {
				// THIS DOESNT CATCH FILE EXISTS BUT IS HUNG
				if err == ErrFileContainsError {
					fmt.Fprintf(os.Stderr,
						"error: %v on %s\n", err, job.Name)
				}
				Global.ErrMap[err]++
				// can't use job.whatever if you want to modify the thing
				points[i].Resub = &Calc{
					Name:    job.Name + "_redo",
					ResubID: q.Resubmit(job.Name, err),
				}
				resubs++
				Global.WatchedJobs = append(Global.WatchedJobs, points[i].Resub.ResubID)
			} else if job.Resub != nil {
				if energy, t, gradients,
					err = prog.ReadOut(job.Resub.Name +
					".out"); err == nil {
					success = true
					if energy < min {
						min = energy
					}
					realTime += t
					heap.Add(job.Name)
				}
			}
			if success {
				if job.Src != nil && job.Src.Status == NotCalculated {
					job.Src.Status = Done
					job.Src.Value = energy
				}
				points[nJobs-1], points[i] = points[i], points[nJobs-1]
				nJobs--
				points = points[:nJobs]
				if *dump {
					for _, c := range job.Coords {
						fmt.Fprintf(dumper, "%15.10f", c)
					}
					fmt.Fprintf(dumper, "%20.12f\n", energy)
				}
				if !GRAD {
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
				} else {
					norun--
				}
				success = false
			}
		}
		if shortenBy < 1 {
			fmt.Fprintln(os.Stderr, "Didn't shorten, sleeping")
			q.Stat(&qstat)
			time.Sleep(time.Duration(Conf.SleepInt) * time.Second)
		} else {
			fmt.Fprintf(os.Stderr,
				"%s finished %d/%d submitted, %v polling %d jobs\n",
				time.Now().Format("[2006-01-02 15:04]"),
				finished, Global.Submitted,
				time.Since(pollStart).Round(time.Millisecond), nJobs-norun)
		}
		if check >= Conf.CheckInt {
			if Conf.CheckInt > 0 {
				MakeCheckpoint(prog.GetDir())
			}
			check = 1
			fmt.Fprintf(os.Stderr, "CPU time: %.3f s\n",
				float64(GetCPU()-Global.StartCPU)/1e9)
		}
		if heap.Len() >= Conf.ChunkSize && !*nodel {
			heap.Dump()
			stackDump()
		}
		// Termination
		if nJobs == 0 {
			fmt.Printf("resubmitted %d/%d (%.1f%%),"+
				" points execution time: %v\n",
				resubs, Global.Submitted,
				float64(resubs)/float64(Global.Submitted)*100,
				time.Since(start))
			minutes := int(realTime) / 60
			secRem := realTime - 60*float64(minutes)
			fmt.Printf("total job time (wall): %.2f sec = %dm%.2fs\n",
				realTime, minutes, secRem)
			for k, v := range Global.ErrMap {
				fmt.Printf("%v: %d occurrences\n", k, v)
			}
			return
		}
	}
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
	err = prog.FormatCart(Conf.Geometry)
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
	mpHarm := make([]float64, spec.Nfreqs)
	Summarize(os.Stdout, res.ZPT, mpHarm, intderHarms, res.Harm, res.Fund, res.Corr)
}

func initialize(infile string) (prog Program, intder *Intder, anpass *Anpass) {
	if !*test {
		DupOutErr(infile)
	}
	if *freqs {
		Conf = ParseInfile(infile).ToConfig()
		spectro.Command = Conf.Spectro
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
	if !*test {
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
	}
	dir := filepath.Dir(infile)
	Conf = ParseInfile(infile).ToConfig()
	if *checkpoint {
		LoadCheckpoint(dir)
	}
	spectro.Command = Conf.Spectro
	switch {
	case CART:
		fmt.Printf("%d coords requires %d Cartesian points\n",
			Conf.Ncoords, CartPoints(Conf.Ncoords))
		if *count {
			os.Exit(0)
		}
	case GRAD:
		fmt.Printf("%d coords requires %d gradient points\n",
			Conf.Ncoords, GradPoints(Conf.Ncoords))
		if *count {
			os.Exit(0)
		}
	case *count:
		fmt.Println("-count only implemented for gradients and Cartesians")
		os.Exit(1)
	}
	mpName := filepath.Join(dir, Conf.MolproTmpl)
	idName := filepath.Join(dir, Conf.IntderTmpl)
	apName := filepath.Join(dir, Conf.AnpassTmpl)
	var err error
	switch Conf.Package {
	case "molpro", "":
		prog = new(Molpro)
		Conf.Queue.NewMolpro()
	case "g16", "gaussian", "gauss":
		prog = new(Gaussian)
		Conf.Queue.NewGauss()
	case "mopac":
		prog = new(Mopac)
		Conf.Queue.NewMopac()
	}
	err = prog.Load(mpName)
	if err != nil {
		errExit(err, fmt.Sprintf("loading qc template %q", mpName))
	}
	prog.SetDir(dir)
	if SIC {
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
	Global.ErrMap = make(map[error]int)
	Global.Nodes = PBSnodes()
	if !*test {
		fmt.Printf("Available nodes: %q\n\n", Global.Nodes)
	}
	return prog, intder, anpass
}

func main() {
	Global.StartCPU = GetCPU()
	defer CatchPanic()
	go CatchKill()
	args := ParseFlags()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "pbqff: no input file supplied\n")
		os.Exit(1)
	}
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	prog, intder, anpass := initialize(args[0])
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
	)
	if OPT {
		if Conf.GeomType != "zmat" {
			panic("optimization requires a zmat geometry")
		}
		err := prog.FormatZmat(Conf.Geometry)
		if err != nil {
			panic(err)
		}
		E0 = prog.Run(opt, Conf.Queue)
		cart, zmat, err = prog.HandleOutput("opt/opt")
		if err != nil {
			panic(err)
		}
		prog.UpdateZmat(zmat)
		prog.Run(freq, Conf.Queue)
	} else {
		if !strings.Contains("cart,xyz", Conf.GeomType) {
			panic("expecting cartesian geometry")
		}
		err := prog.FormatCart(Conf.Geometry)
		if err != nil {
			panic(err)
		}
		cart = prog.GetGeom()
		// only required for cartesians
		if CART {
			E0 = prog.Run(none, Conf.Queue)
		}
	}
	mol := symm.ReadXYZ(strings.NewReader(cart))
	fmt.Printf("Point group %s\n", mol.Group)

	var gen func() ([]Calc, bool)

	switch {
	case SIC && *irdy == "":
		names = intder.ConvertCart(cart)
	case SIC:
		names = strings.Fields(*irdy)
	default:
		names, coords = XYZGeom(cart)
	}
	natoms = len(names)
	other3, other4 := initArrays(natoms)

	var forces [][]int
	fmt.Println(forces)
	if SIC {
		intder.WritePts("pts/intder.in")
		RunIntder("pts/intder")
		gen = BuildPoints(prog, Conf.Queue, "pts/file07", names, true)
	} else {
		ncoords = len(coords)
		if CART {
			gen, forces = BuildCartPoints(prog, Conf.Queue, "pts/inp", names,
				coords, mol)
		} else if GRAD {
			gen = BuildGradPoints(prog, Conf.Queue, "pts/inp", names,
				coords, mol)
		}
	}

	if gen == nil {
		fmt.Println("sic, cart, grad", SIC, CART, GRAD)
		panic("nil gen closure")
	}
	min, _ = Drain(prog, Conf.Queue, ncoords, E0, gen)
	queueClear(Global.WatchedJobs)

	if SIC {
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
		Summarize(os.Stdout, res.ZPT, mpHarm, intderHarms, res.Harm,
			res.Fund, res.Corr)
	} else if CART {
		energies := FloatsFromCountFloats(cenergies)
		for i := range energies {
			energies[i] -= min
		}
		nforces := make([]float64, 0)
		for i := 0; i < len(forces[0]); i++ {
			for j := 0; j < len(forces); j++ {
				nforces = append(nforces, float64(forces[j][i]))
			}
		}
		exps := mat.NewDense(len(forces[0]), len(forces), nforces)
		steps := DispToStep(Disps(forces))
		stepdat := make([]float64, 0)
		for _, step := range steps {
			stepdat = append(stepdat,
				Step(make([]float64, ncoords), step...)...)
		}
		disps := mat.NewDense(len(stepdat)/ncoords, ncoords, stepdat)
		out, _ := os.Create("/tmp/anpass.out")
		defer out.Close()
		longLine, _, _ := anp.Run(
			out, os.TempDir(), disps, energies, exps,
		)
		disps, energies = anp.Bias(disps, energies, longLine)
		coeffs, _ := anp.Fit(disps, energies, exps)
		fcs := anp.MakeFCs(coeffs, exps)
		Format9903(ncoords, fcs)
		PrintFortFile(fc2, natoms, 6*natoms, "fort.15")
		if Conf.Deriv > 2 {
			PrintFortFile(fc3, natoms, other3, "fort.30")
		}
		if Conf.Deriv > 3 {
			PrintFortFile(fc4, natoms, other4, "fort.40")
			var buf bytes.Buffer
			for i := range coords {
				if i%3 == 0 && i > 0 {
					fmt.Fprint(&buf, "\n")
				}
				fmt.Fprintf(&buf, " %.10f", coords[i]/ANGBOHR)
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
	} else {
		PrintFortFile(fc2, natoms, 6*natoms, "fort.15")
		if Conf.Deriv > 2 {
			PrintFortFile(fc3, natoms, other3, "fort.30")
		}
		if Conf.Deriv > 3 {
			PrintFortFile(fc4, natoms, other4, "fort.40")
			var buf bytes.Buffer
			for i := range coords {
				if i%3 == 0 && i > 0 {
					fmt.Fprint(&buf, "\n")
				}
				fmt.Fprintf(&buf, " %.10f", coords[i]/ANGBOHR)
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
	fmt.Printf("total CPU time used: %.3f s\n", float64(GetCPU()-Global.StartCPU)/1e9)
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
