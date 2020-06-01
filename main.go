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
	"strings"
	"syscall"
	"text/template"
	"time"
)

var (
	Input            [NumKeys]string
	overwrite        bool
	dirs             = []string{"opt", "freq", "pts", "freqs"}
	brokenFloat      = math.NaN()
	energyLine       = "energy="
	molproTerminated = "Molpro calculation terminated"
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

func main() {
	MakeDirs(".")
	Args := ParseFlags()
	if len(Args) < 1 {
		log.Fatal("pbqff: no input file supplied")
	}
	// might want a LoadDefaults function or something
	// and then overwrite parts with ParseInfile
	ParseInfile(Args[0])
	// check for local templates and then use main one
	// - add template name to infile
	// write opt.inp and mp.pbs
	prog := Molpro{
		Geometry: Input[Geometry],
		Basis:    Input[Basis],
		Charge:   Input[Charge],
		Spin:     Input[Spin],
		Method:   Input[Method],
	}

	// need to figure out how to handle template stuff
	// maybe bundle the defaults with the executable?
	// otherwise weird handling path
	prog.WriteInput("opt/opt.inp", "templates/molpro.in")
	WritePBS("opt/mp.pbs", "templates/pbs.in",
		&Job{MakeName(Input[Geometry]), "opt/opt.inp", 35})
	// submit opt, wait for it to finish in main goroutine - block
	Submit("opt/mp.pbs")
	outfile := "opt/opt.out"
	energy, err := prog.ReadOut(outfile)
	for err != nil {
		HandleSignal(35, time.Minute)
		energy, err = prog.ReadOut(outfile)
		if (err == ErrEnergyNotParsed || err == ErrFinishedButNoEnergy ||
			err == ErrFileContainsError || err == ErrBlankOutput) ||
			err == ErrFileNotFound {

			fmt.Println("resubmitting for", err)
			Submit("opt/mp.pbs")
		}
	}
	fmt.Println(energy)
	// - report any errors and warnings from molpro
	// write freq.inp and that mp.pbs
	// submit freq, wait in separate goroutine
	// set up pts using opt.log geometry and given intder.in file
	// submit points, wait for them to finish
	// - check for failed jobs, probably just loop at some interval
	//   doesnt need to be fast (and resource intensive) like gocart
	// gather energies, convert to relative
	// write anpass1.in, run anpass
	// write anpass2.in, run anpass
	// write intder_geom.in, run intder_geom
	// write freqs/intder.in, run intder
	// move files (tennis)
	// run spectro
	// handle resonances
	// run spectro
	// extract output
}
