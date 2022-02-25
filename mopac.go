package main

import (
	"bufio"
	"errors"
	"fmt"
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
func (m *Mopac) AugmentHead()        {}

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
	switch proc {
	case opt:
		// optimization is the default, so just make sure 1SCF
		// isn't in the header to turn it off
		m.Head = strings.Replace(m.Head, "1SCF", "", -1)
	case freq:
		panic("proc not implemented for mopac")
	}
	f, err := os.Create(filename)
	defer f.Close()
	if err != nil {
		panic(err)
	}
	// m.Head must end in \n
	fmt.Fprintf(f, "%s%s\n\n", m.Head, m.Geom)

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
	// in case there are units in the zmat params, remove them
	for _, line := range split[i:] {
		out = append(out, unit.ReplaceAllString(line, ""))
	}
	m.Geom = strings.Join(out, "\n")
	return
}

func (m *Mopac) FormatGeom(geom string) string {
	panic("unimplemented")
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

func (m *Mopac) HandleOutput(string) (string, string, error) {
	panic("unimplemented")
}
func (m *Mopac) UpdateZmat(string) {
	panic("unimplemented")
}
func (m *Mopac) FormatCart(geom string) error {
	m.Geom = geom
	return nil
}

func (m *Mopac) ReadOut(filename string) (
	energy float64, time float64, grad []float64, err error) {
	// TODO return the proper errors instead of just panicking
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
		case i == 0 && strings.Contains(strings.ToUpper(line), "ERROR"):
			err = ErrFileContainsError
			return
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
	if i == 0 {
		err = ErrBlankOutput
		return
	}
	// should I close old f first? what about deferring double
	// close?
	auxfile := TrimExt(filename) + ".aux"
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
		}
	}
	return
}
func (m *Mopac) ReadFreqs(string) []float64 {
	panic("unimplemented")
}
