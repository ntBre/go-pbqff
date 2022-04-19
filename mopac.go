package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Mopac struct {
	Dir  string
	Head string
	Geom string
}

func (m *Mopac) GetDir() string      { return m.Dir }
func (m *Mopac) SetDir(dir string)   { m.Dir = dir }
func (m *Mopac) GetGeom() string     { return m.Geom }
func (m *Mopac) SetGeom(geom string) { m.Geom = geom }
func (m *Mopac) AugmentHead() {
	m.Head = "A0  " + m.Head
}

// Load a MOPAC input file from filename
func (m *Mopac) Load(filename string) error {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(f)
	var str strings.Builder
	for i := 0; scanner.Scan() && i < 3; i++ {
		fmt.Fprintf(&str, "%s\n", scanner.Text())
	}
	m.Head = str.String()
	return nil
}

func (m *Mopac) WriteInput(filename string, proc Procedure) {
	var (
		head string = m.Head
		geom string = m.Geom
	)
	switch proc {
	case opt:
		// optimization is the default, so just make sure 1SCF
		// isn't in the header to turn it off
		head = strings.Replace(head, "1SCF", "", -1)
		// also turn off XYZ since it needs to be a ZMAT for opt
		head = strings.Replace(head, "XYZ", "", -1)
	case freq:
		// if AIGIN was needed for the optimization, delete it after
		head = strings.Replace(head, "AIGIN", "", -1)
		lines := strings.Split(
			strings.TrimSpace(head),
			"\n",
		)
		if len(lines) != 3 {
			panic("wrong number of lines in MOPAC header")
		}
		lines[0] += " FORCE"
		tmp := strings.Join(lines, "\n")
		head = tmp + "\n"
	default:
		head = strings.Replace(head, "AIGIN", "", -1)
		head = "1SCF XYZ " + head
	}
	f, err := os.Create(filename)
	defer f.Close()
	if err != nil {
		panic(err)
	}
	// m.Head must end in \n
	fmt.Fprintf(f, "%s%s\n\n", head, geom)

}

// FormatZmat formats a z-matrix for use in a MOPAC input file and
// places it in the Geometry field of m
func (m *Mopac) FormatZmat(geom string) (err error) {
	var out []string
	err = errors.New("improper z-matrix")
	split := strings.Split(geom, "\n")
	unit := regexp.MustCompile(`(?i)\s+(ang|deg)`)
	var (
		i    int
		line string
	)
	for i, line = range split {
		if strings.Contains(line, "=") {
			out = append(out, split[:i]...)
			err = nil
			break
		}
	}
	out = append(out, "")
	// in case there are units in the zmat params, remove them
	for _, line := range split[i:] {
		out = append(out, unit.ReplaceAllString(line, ""))
	}
	m.Geom = strings.Join(out, "\n")
	return
}

func (m *Mopac) FormatGeom(geom string) string {
	return geom
}

func (m *Mopac) Run(proc Procedure, q Queue) (E0 float64) {
	var (
		dir  string
		name string
	)
	switch proc {
	case opt:
		dir = "opt"
		name = "opt"
	case freq:
		dir = "freq"
		name = "freq"
	case none:
		dir = "pts/inp"
		name = "ref"
	}
	dir = filepath.Join(m.Dir, dir)
	infile := filepath.Join(dir, name+".inp")
	pbsfile := filepath.Join(dir, name+".pbs")
	outfile := filepath.Join(dir, name+".out")
	E0, _, _, err := m.ReadOut(outfile)
	if *read && err == nil {
		return
	}
	m.WriteInput(infile, proc)
	q.WritePBS(pbsfile,
		&Job{
			Name: fmt.Sprintf("%s-%s",
				MakeName(Conf.Geometry), proc),
			Filename: infile,
			NumCPUs:  Conf.NumCPUs,
			PBSMem:   Conf.PBSMem,
			Jobs:     []string{infile},
		})
	jobid := q.Submit(pbsfile)
	jobMap := make(map[string]bool)
	jobMap[jobid] = false
	// only wait for opt and ref to run
	for proc != freq && err != nil {
		E0, _, _, err = m.ReadOut(outfile)
		q.Stat(&jobMap)
		if err == ErrFileNotFound && !jobMap[jobid] {
			fmt.Fprintf(os.Stderr, "resubmitting %s for %v\n",
				pbsfile, err)
			jobid = q.Submit(pbsfile)
			jobMap[jobid] = false
		}
		time.Sleep(time.Duration(Conf.SleepInt) * time.Second)
	}
	return
}

func (m *Mopac) HandleOutput(filename string) (
	cart string, zmat string, err error) {
	auxfile := filename + ".inp.aux"
	f, err := os.Open(auxfile)
	defer f.Close()
	if err != nil {
		err = ErrFileNotFound
		return
	}
	scanner := bufio.NewScanner(f)
	var (
		line     string
		fields   []string
		atoms    []string
		coords   []float64
		inatoms  bool
		incoords bool
		fac      float64 = 1
	)
	if SIC {
		fac = ANGBOHR
	}
	for scanner.Scan() {
		line = scanner.Text()
		fields = strings.Fields(line)
		switch {
		case strings.Contains(line, "ATOM_EL"):
			inatoms = true
		case strings.Contains(line, "ATOM_CORE"):
			inatoms = false
		case inatoms:
			atoms = append(atoms, fields...)
		case strings.Contains(line, "ATOM_X_OPT"):
			incoords = true
		case strings.Contains(line, "ATOM_CHARGES"):
			incoords = false
		case incoords:
			for _, f := range fields {
				v, _ := strconv.ParseFloat(f, 64)
				coords = append(coords, v/fac)
			}
		}
	}
	cart = ZipXYZ(atoms, coords)
	// MOPAC doesn't give the optimized Zmat, so just use the Cart
	// again
	zmat = cart
	return
}
func (m *Mopac) UpdateZmat(new string) {
	m.Geom = new
}
func (m *Mopac) FormatCart(geom string) error {
	m.Geom = geom
	return nil
}

func (m *Mopac) ReadOut(filename string) (
	energy float64, time float64, grad []float64, err error) {
	// TODO return the proper errors instead of just panicking
	base := TrimExt(filename)
	filename = base + ".inp.out"
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		err = ErrFileNotFound
		return
	}
	scanner := bufio.NewScanner(f)
	err = ErrEnergyNotFound
	var (
		line   string
		fields []string
		i      int
	)
	for i = 0; scanner.Scan(); i++ {
		line = scanner.Text()
		switch {
		case i == 0 && strings.Contains(strings.ToUpper(line), "PANIC"):
			panic("panic requested in output file")
		case strings.Contains(strings.ToUpper(line), "ERROR"):
			log.Fatalf("file %q contains error\n", filename)
		case strings.Contains(line, "TOTAL JOB TIME"):
			fields = strings.Fields(line)
			time, err = strconv.ParseFloat(fields[3], 64)
			if err != nil {
				panic(err)
			}
		case strings.Contains(line, "== MOPAC DONE =="):
			break
		}
	}
	// should I close old f first? what about deferring double
	// close?
	auxfile := base + ".inp.aux"
	f, err = os.Open(auxfile)
	defer f.Close()
	if err != nil {
		err = ErrFileNotFound
		return
	}
	err = ErrEnergyNotFound
	scanner = bufio.NewScanner(f)
	for scanner.Scan() {
		line = scanner.Text()
		switch {
		case strings.Contains(line, "HEAT_OF_FORMATION"):
			fields = strings.Split(line, "=")
			strVal := fields[1]
			energy, err = strconv.ParseFloat(
				strings.Replace(strVal, "D", "E", -1),
				64,
			)
			energy /= KCALHT
		}
	}
	return
}
func (m *Mopac) ReadFreqs(string) []float64 {
	Warn("ReadFreqs not implemented for MOPAC")
	return nil
}
