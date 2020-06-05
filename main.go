/*
Push-button QFF
---------------
The goal of this program is to streamline the generation
of quartic force fields, automating as many pieces as possible.
Requirements:
- intder, anpass, and spectro executables
- template intder.in, anpass.in and spectro.in files
*/

package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"
)

const (
	resBound = 1e-16
)

var (
	Input            [NumKeys]string
	overwrite        bool
	dirs             = []string{"opt", "freq", "pts", "freqs", "pts/inp"}
	brokenFloat      = math.NaN()
	energyLine       = "energy="
	molproTerminated = "Molpro calculation terminated"
	defaultOpt       = "optg,grms=1.d-8,srms=1.d-8"
)

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

func LoadTemplate(filename string) *template.Template {
	t, err := template.ParseFiles(filename)
	if err != nil {
		panic(err)
	}
	return t
}

func MakeName(geom string) (name string) {
	atoms := make(map[string]int)
	split := strings.Split(geom, "\n")
	for _, line := range split {
		fields := strings.Fields(line)
		// not a dummy atom
		if len(fields) >= 1 &&
			!strings.Contains(strings.ToUpper(fields[0]), "X") {
			atoms[strings.ToLower(fields[0])] += 1
		}
	}
	for k, v := range atoms {
		name += fmt.Sprintf("%s%d", k, v)
	}
	return
}

// Read a file and return a slice of strings of the lines
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

// Set up the directory structure described by dirs
func MakeDirs(root string) (err error) {
	for _, dir := range dirs {
		filename := root + "/" + dir
		if _, err := os.Stat(filename); !os.IsNotExist(err) {
			if overwrite {
				os.RemoveAll(filename)
			} else {
				log.Fatalf("MakeDirs: directory %q already exists "+
					"overwrite with -o\n", dir)
			}
		}
		e := os.Mkdir(filename, 0755)
		if e != nil {
			err = fmt.Errorf("MakeDirs: %q on making directory %q\n",
				e, dir)
		}
	}
	return err
}

func ParseFlags() []string {
	flag.BoolVar(&overwrite, "o", false, "overwrite existing inp directory")
	flag.Parse()
	return flag.Args()
}

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

// Take a cartesian geometry and extract the atom names
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

type Anpass struct {
	Head string
	Fmt1 string
	Fmt2 string
	Body string
	Tail string
}

// Helper for building anpass file body
func (a *Anpass) BuildBody(buf *bytes.Buffer, energies []float64) {
	for i, line := range strings.Split(a.Body, "\n") {
		if line != "" {
			for _, field := range strings.Fields(line) {
				f, _ := strconv.ParseFloat(field, 64)
				fmt.Fprintf(buf, a.Fmt1, f)
			}
			fmt.Fprintf(buf, a.Fmt2+"\n", energies[i])
		}
	}
}

func (a *Anpass) WriteAnpass(filename string, energies []float64) {
	var buf bytes.Buffer
	buf.WriteString(a.Head)
	a.BuildBody(&buf, energies)
	buf.WriteString(a.Tail)
	ioutil.WriteFile(filename, []byte(buf.String()), 0755)
}

func (a *Anpass) WriteAnpass2(filename, longLine string, energies []float64) {
	var buf bytes.Buffer
	buf.WriteString(a.Head)
	a.BuildBody(&buf, energies)
	for _, line := range strings.Split(a.Tail, "\n") {
		if strings.Contains(line, "END OF DATA") {
			buf.WriteString("STATIONARY POINT\n" +
				longLine + "\n")
		} else if strings.Contains(line, "!STATIONARY POINT") {
			continue
		}
		buf.WriteString(line + "\n")
	}
	ioutil.WriteFile(filename, []byte(buf.String()), 0755)
}

func LoadAnpass(filename string) *Anpass {
	file, _ := ioutil.ReadFile(filename)
	lines := strings.Split(string(file), "\n")
	var (
		a          Anpass
		buf        bytes.Buffer
		body, tail bool
	)
	head := true
	for _, line := range lines {
		if head && string(line[0]) == "(" {
			head = false
			buf.WriteString(line + "\n")
			a.Head = buf.String()
			buf.Reset()
			// assume leading and trailing parentheses
			s := strings.Split(strings.ToUpper(line[1:len(line)-1]), "F")
			// assume trailing comma
			a.Fmt1 = "%" + string(s[1][:len(s[1])-1]) + "f"
			a.Fmt2 = "%" + string(s[2]) + "f"
			body = true
			continue
		}
		if body && strings.Contains(line, "UNKNOWNS") {
			body = false
			tail = true
		} else if body {
			f := strings.Fields(line)
			for i := 0; i < len(f)-1; i++ {
				val, _ := strconv.ParseFloat(f[i], 64)
				fmt.Fprintf(&buf, a.Fmt1, val)
			}
			buf.WriteString("\n")
			a.Body += buf.String()
			buf.Reset()
			continue
		}
		if tail {
			a.Tail += line + "\n"
		}
		buf.WriteString(line + "\n")
	}
	return &a
}

// Scan an anpass output file and return the "long line"
func GetLongLine(filename string) (string, bool) {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var line, lastLine string
	for scanner.Scan() {
		line = scanner.Text()
		if strings.Contains(line, "0EIGENVALUE") {
			return lastLine, true
		} else if strings.Contains(line, "RESIDUALS") {
			fields := strings.Fields(line)
			if res, _ := strconv.
				ParseFloat(fields[len(fields)-1], 64); res > resBound {
				fmt.Fprintf(os.Stderr, "GetLongLine: warning: sum of squared"+
					" residuals %e greater than %e\n", res, resBound)
			}
		}
		lastLine = line
	}
	return "", false
}

func main() {
	MakeDirs(".")
	Args := ParseFlags()
	if len(Args) < 1 {
		log.Fatal("pbqff: no input file supplied")
	}
	// might want a LoadDefaults function or something
	// and then overwrite parts with ParseInfile
	ParseInfile(Args[0])
	prog := Molpro{
		Geometry: Input[Geometry],
		Basis:    Input[Basis],
		Charge:   Input[Charge],
		Spin:     Input[Spin],
		Method:   Input[Method],
	}
	// check for local templates and then use main one
	// - add template name to infile
	// write opt.inp and mp.pbs
	// need to figure out how to handle template stuff
	// maybe bundle the defaults with the executable?
	// otherwise weird handling path
	prog.WriteInput("opt/opt.inp", "templates/molpro.in")
	WritePBS("opt/mp.pbs", "templates/pbs.in",
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

			fmt.Println("resubmitting for", err)
			Submit("opt/mp.pbs")
		}
	}
	cart, zmat, err := prog.HandleOutput("opt/opt")
	if err != nil {
		// actually want to try to recover here probably
		panic(err)
	}
	// write freq.inp and that mp.pbs
	prog.Geometry = zmat
	prog.WriteInput("freq/freq.inp", "templates/molpro.in")
	WritePBS("freq/mp.pbs", "templates/pbs.in",
		&Job{MakeName(Input[Geometry]) + "/freq", "freq/freq.inp", 35})
	// submit freq, wait in separate goroutine
	// TODO make this a closure - actually make it a function
	// since I use it twice
	Submit("freq/mp.pbs")
	outfile = "freq/freq.out"
	_, err = prog.ReadOut(outfile)
	for err != nil {
		HandleSignal(35, time.Minute)
		_, err = prog.ReadOut(outfile)
		if (err == ErrEnergyNotParsed || err == ErrFinishedButNoEnergy ||
			err == ErrFileContainsError || err == ErrBlankOutput) ||
			err == ErrFileNotFound {

			fmt.Println("resubmitting freq for", err)
			Submit("freq/mp.pbs")
		}
	}
	// set up pts using opt.log geometry and given intder.in file
	intder := LoadIntder("intder.in")
	intder.WritePts("pts/intder.in")
	// run intder
	RunIntder("pts/intder")
	// build points and the list of pts to submit
	atomNames := GetNames(cart)
	pts := BuildPoints("pts/file07", atomNames)
	// submit points, wait for them to finish
	for _, job := range pts {
		Submit(job + ".pbs")
	}

	// - check for failed jobs, probably just loop at some interval
	//   doesnt need to be fast (and resource intensive) like gocart
	ptsInit := len(pts)
	for len(pts) > 0 {
		shortenBy := 0
		for i, job := range pts {
			_, err := prog.ReadOut(job + ".out")
			if err == nil {
				pts[i], pts[len(pts)-1] = pts[len(pts)-1], pts[i]
				shortenBy++
			} else if (err == ErrEnergyNotParsed || err == ErrFinishedButNoEnergy ||
				err == ErrFileContainsError || err == ErrBlankOutput) ||
				(err == ErrFileNotFound && len(pts) < ptsInit/2) {

				fmt.Printf("resubmitting %s for %s", job, err)
				Submit(job + ".pbs")
			}
		}
		pts = pts[:len(pts)-shortenBy]
		// if the list is shortened by less than 10%,
		// sleep. could play with both of these values
		if float64(shortenBy/ptsInit) < 0.1 {
			time.Sleep(time.Second)
		}
	}

	// gather energies, convert to relative
	// - this should probably be part of the checking
	//   for finished jobs, but a little weird with rotating them to end
	energies := make([]float64, len(pts))
	for i, job := range pts {
		// disregard error because we checked them all above
		energy, _ := prog.ReadOut(job + ".out")
		energies[i] = energy
	}
	toSort := make([]float64, len(pts))
	copy(toSort, energies)
	sort.Float64s(toSort)
	min := toSort[0]
	// convert to relative energies
	for i, _ := range energies {
		energies[i] -= min
	}
	// write anpass1.in
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
	intder.ReadGeom("freqs/intder_geom.out")
	// read freqs/intder.in bottom from fort.9903
	intder.Read9903("freqs/fort.9903")
	// write freqs/intder.in, run intder
	intder.WriteFreqs("freqs/intder.in", atomNames)
	// read harmonics from intder.out
	intderHarms := intder.ReadOut("freqs/intder.out")
	fmt.Println(intderHarms)
	// move files (tennis)
	Tennis()
	// run spectro
	// handle resonances
	// run spectro
	// extract output
	// MolproFreq    IntderFreq    HARM  FUND CORR
	// later rotational constants, geometry
}

// Move intder output files to the
// filenames expected by spectro
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
