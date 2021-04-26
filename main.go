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
	"errors"
	"fmt"
	"math"
	"os"
	"os/signal"
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

// DoPts is a helper function for checking whether the PTS flag is set
func DoPts() bool { return flags&PTS > 0 }

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
)

var Global struct {
	Nodes  []string
	JobNum int
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

// WhichCluster sets the PBS template and energyLine depending on the
// which computer is to be used
// Optimize runs a Molpro optimization in the opt directory
func (prog *Molpro) Optimize() (E0 float64) {
	// write opt.inp and mp.pbs
	prog.WriteInput("opt/opt.inp", opt)
	WritePBS("opt/mp.pbs",
		&Job{MakeName(Conf.Str(Geometry)) + "-opt", "opt/opt.inp",
			35, "", "", Conf.Int(NumJobs)}, pbsMaple)
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

// This should be combined with optimize; if DoOpt, optimize, but both
// are used as references

// RefEnergy runs a Molpro single point energy calculation in the
// pts/inp directory
func (prog *Molpro) RefEnergy() (E0 float64) {
	// TODO use filepath
	dir := "pts/inp/"
	infile := "ref.inp"
	pbsfile := "ref.pbs"
	outfile := "ref.out"
	E0, _, _, err := prog.ReadOut(dir + outfile)
	if *read && err == nil {
		return
	}

	prog.WriteInput(dir+infile, none)
	WritePBS(dir+pbsfile,
		&Job{
			Name:     MakeName(Conf.Str(Geometry)) + "-ref",
			Filename: dir + infile,
			Signal:   35,
			NumJobs:  Conf.Int(NumJobs),
		}, pbsMaple)
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
func (prog *Molpro) Frequency(absPath string) []float64 {
	// write freq.inp and that mp.pbs
	inp := filepath.Join(absPath, "freq.inp")
	pbs := filepath.Join(absPath, "mp.pbs")
	prog.WriteInput(inp, freq)
	WritePBS(pbs,
		&Job{
			Name:     MakeName(Conf.Str(Geometry)) + "-freq",
			Filename: inp,
			Signal:   35,
			NumJobs:  Conf.Int(NumJobs),
		}, pbsMaple)
	// submit freq, wait in separate goroutine
	// doesn't matter if this finishes
	Submit(pbs)
	outfile := filepath.Join(absPath, "freq.out")
	_, _, _, err := prog.ReadOut(outfile)
	for err != nil {
		HandleSignal(35, time.Minute)
		_, _, _, err = prog.ReadOut(outfile)
		// dont resubmit freq
		if err == ErrEnergyNotParsed || err == ErrFinishedButNoEnergy ||
			err == ErrFileContainsError {
			fmt.Fprintln(os.Stderr, "error in freq, aborting that calculation")
			return nil
		}
	}
	return prog.ReadFreqs(outfile)
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
	WritePBS(name+"_redo.pbs", &Job{"redo", name + "_redo.inp", 35, "", "",
		Conf.Int(NumJobs)}, pbsMaple)
	return Submit(name + "_redo.pbs")
}

// Drain drains the queue of jobs and receives on ch when ready for
// more. prog is only used for its ReadOut method, and ncoords is used
// to construct the zero gradient array.
func Drain(prog *Molpro, ncoords int, ch chan Calc, E0 float64) (min, realTime float64) {
	start := time.Now()
	fmt.Println("step sizes: ", Conf.FlSlice(Deltas))
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
	)
	heap := new(GarbageHeap)
	for {
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
				err = prog.ReadOut(job.Name + ".out"); err == nil {
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
					(err == ErrFileNotFound && CheckLog(job.CmdFile,
						job.Name) && CheckProg(job.CmdFile))) {
				// THIS DOESNT CATCH FILE EXISTS BUT IS HUNG
				if err == ErrFileContainsError {
					fmt.Fprintf(os.Stderr,
						"error: %v on %s\n", err, job.Name)
				}
				errMap[err]++
				// can't use job.whatever if you want to modify the thing
				points[i].Resub = &Calc{
					Name: job.Name + "_redo",
					ID:   Resubmit(job.Name, err),
				}
				resubs++
				ptsJobs = append(ptsJobs, points[i].Resub.ID)
			} else if job.Resub != nil {
				// should DRY this up, inside if is
				// same as case 3 above
				// should also check if resubmitted
				// job has finished with qsub and set
				// pointer to nil if it has without
				// success
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
				points[nJobs-1], points[i] = points[i], points[nJobs-1]
				nJobs--
				points = points[:nJobs]
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
						queueClear([]string{paraJobs[job.ChunkNum]})
						if *debug {
							fmt.Printf("clearing paracount of"+
								"chunk %d, jobid %s\n",
								job.ChunkNum,
								paraJobs[job.ChunkNum])
						}
					}
				}
				success = false
			}
		}
		if shortenBy < 1 {
			fmt.Fprintln(os.Stderr, "Didn't shorten, sleeping")
			time.Sleep(time.Duration(Conf.Int(SleepInt)) * time.Second)
		}
		if check >= Conf.Int(CheckInt) {
			if !nocheck {
				MakeCheckpoint()
			}
			check = 1
			fmt.Fprintf(os.Stderr, "CPU time: %.3f s\n",
				float64(GetCPU()-StartCPU)/1e9)
		}
		if heap.Len() >= Conf.Int(ChunkSize) && !*nodel {
			heap.Dump()
		}
		// Progress
		fmt.Fprintf(os.Stderr, "finished %d/%d submitted, %v polling %d jobs\n",
			finished, submitted,
			time.Since(pollStart).Round(time.Millisecond), nJobs)
		// only receive more jobs if there is room
		for count := 0; count < Conf.Int(ChunkSize) &&
			nJobs < Conf.Int(JobLimit); count++ {
			calc, ok := <-ch
			if !ok && finished == submitted {
				fmt.Fprintf(os.Stderr,
					"resubmitted %d/%d (%.1f%%),"+
						" points execution time: %v\n",
					resubs, submitted,
					float64(resubs)/float64(submitted)*100,
					time.Since(start))
				minutes := int(realTime) / 60
				secRem := realTime - 60*float64(minutes)
				fmt.Fprintf(os.Stderr,
					"total job time (wall): %.2f sec = %dm%.2fs\n",
					realTime, minutes, secRem)
				for k, v := range errMap {
					fmt.Fprintf(os.Stderr, "%v: %d occurrences\n", k, v)
				}
				return
			} else if ok {
				points = append(points, calc)
				nJobs = len(points)
			} else if !ok {
				nJobs = len(points)
				break
			}
		}
	}
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
			out, err := exec.Command("ssh", host, "-t",
				"rm -rf /tmp/$USER/"+job+".maple").CombinedOutput()
			if *debug {
				fmt.Println("CombinedOutput and error from queueClear: ",
					string(out), err)
			}
		}
	}
	err := exec.Command("qdel", jobs...).Run()
	return err
}

func CartPoints(n int) int {
	return 2 * n * (n*n*n + 2*n*n + 8*n + 1) / 3
}

func GradPoints(n int) int {
	return n * (4*n*n + 12*n + 8) / 3
}

func DupOutErr(infile string) {
	// set up output and err files and dup their fds to stdout and stderr
	// https://github.com/golang/go/issues/325
	base := infile[:len(infile)-len(path.Ext(infile))]
	outfile, _ := os.Create(base + ".out")
	errfile, _ := os.Create(base + ".err")
	syscall.Dup2(int(outfile.Fd()), 1)
	syscall.Dup2(int(errfile.Fd()), 2)
}

func initialize() (prog *Molpro, intder *Intder, anpass *Anpass) {
	// parse flags for overwrite before mkdirs
	args := ParseFlags()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "pbqff: no input file supplied\n")
		os.Exit(1)
	}
	infile := args[0]
	DupOutErr(infile)
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
	mpName := Conf.Str(MolproTmpl)
	idName := Conf.Str(IntderTmpl)
	apName := Conf.Str(AnpassTmpl)
	var err error
	prog, err = LoadMolpro(mpName)
	if err != nil {
		errExit(err, fmt.Sprintf("loading molpro input %q", mpName))
	}
	if DoSIC() {
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
	Global.Nodes = PBSnodes()
	fmt.Printf("nodes: %q\n", Global.Nodes)
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

// CatchPanic recovers from a panic to clear the queue and then
// continues the panic
func CatchPanic() {
	if r := recover(); r != nil {
		fmt.Println("running queueClear before panic")
		queueClear(ptsJobs)
		panic(r)
	}
}

// CatchKill catches SIGTERM to clear the queue before exiting cleanly
func CatchKill() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Signal(syscall.SIGTERM))
	<-c
	fmt.Println("running queueClear before SIGTERM")
	queueClear(ptsJobs)
	errExit(fmt.Errorf("received SIGTERM"), "")
}

func main() {
	StartCPU = GetCPU()
	defer CatchPanic()
	go CatchKill()
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
		cart      string
		zmat      string
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
		if Conf.Str(GeomType) != "zmat" {
			panic("optimization requires a zmat geometry")
		}
		err := prog.FormatZmat(Conf.Str(Geometry))
		if err != nil {
			panic(err)
		}
		E0 = prog.Optimize()
		cart, zmat, err = prog.HandleOutput("opt/opt")
		if err != nil {
			panic(err)
		}
		prog.UpdateZmat(zmat)
		go func() {
			absPath, _ := filepath.Abs("freq")
			mpHarm = prog.Frequency(absPath)
		}()
	} else {
		// assert geomtype is cart or xyz
		if !strings.Contains("cart,xyz", Conf.Str(GeomType)) {
			panic("expecting cartesian geometry")
		}
		err := prog.FormatCart(Conf.Str(Geometry))
		if err != nil {
			panic(err)
		}
		E0 = prog.RefEnergy()
	}

	ch := make(chan Calc, Conf.Int(JobLimit))

	if DoSIC() {
		if *irdy == "" {
			names = intder.ConvertCart(cart)
		} else {
			names = strings.Fields(*irdy)
		}
		if DoPts() {
			intder.WritePts("pts/intder.in")
			RunIntder("pts/intder")
			go func() {
				prog.BuildPoints("pts/file07", names, &cenergies, ch, true)
			}()
		} else {
			// this works if no points were deleted and
			// the files are named the same way between
			// runs, else need a resume from checkpoint
			// thing
			prog.BuildPoints("pts/file07", names, &cenergies, nil, false)
		}
	} else {
		names, coords = XYZGeom(cart)
		natoms = len(names)
		ncoords = len(coords)
		if DoCart() {
			go func() {
				prog.BuildCartPoints("pts/inp", names, coords,
					&fc2, &fc3, &fc4, ch)
			}()
		} else if DoGrad() {
			go func() {
				prog.BuildGradPoints("pts/inp", names, coords,
					&fc2, &fc3, &fc4, ch)
			}()
		}
	}

	min, _ = Drain(prog, ncoords, ch, E0)
	queueClear(ptsJobs)

	// TODO DRY this out some day
	if DoSIC() {
		energies = FloatsFromCountFloats(cenergies)
		// convert to relative energies
		for i := range energies {
			energies[i] -= min
		}
		longLine := DoAnpass(anpass, energies)
		coords, intderHarms := DoIntder(intder, names, longLine)
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
		if mpHarm == nil {
			mpHarm = make([]float64, spec.Nfreqs)
		}
		res := summarize.Spectro(filepath.Join("freqs", "spectro2.out"))
		Summarize(res.ZPT, mpHarm, intderHarms, res.Harm, res.Fund, res.Corr)
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
