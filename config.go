// Could actually have accessor methods for each of these with the
// right types
package main

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

type Config struct {
	Cluster     string
	Package     string // quantum chemistry package (molpro|g16)
	ChemProg    string
	WorkQueue   string
	Delta       float64 // step size
	Deltas      []float64
	Geometry    string
	GeomType    string
	Flags       string
	Deriv       int // derivative level
	JobLimit    int // maximum number of jobs to run at once
	ChunkSize   int // number of jobs submitted in one group
	CheckInt    int // interval for writing checkpoints
	SleepInt    int // interval in seconds between polling jobs
	NumCPUs     int // number of CPUs
	PBSMem      int
	IntderCmd   string
	SpectroCmd  string
	Ncoords     int
	EnergyLine  *regexp.Regexp
	PBSTmpl     *template.Template
	QueueSystem string
	MolproTmpl  string
	AnpassTmpl  string
	IntderTmpl  string
	NumKeys     string
}

// WhichCluster is a helper function for setting Config.EnergyLine and
// Config.PBS based on the selected Cluster
func (c *Config) WhichCluster() {
	cluster := c.Cluster
	sequoia := regexp.MustCompile(`(?i)sequoia`)
	maple := regexp.MustCompile(`(?i)maple`)
	pbs := new(template.Template)
	switch {
	case cluster == "", maple.MatchString(cluster):
		pbs = pbsMaple
	case sequoia.MatchString(cluster):
		c.EnergyLine = regexp.MustCompile(`PBQFF\(2\`)
		pbs = pbsSequoia
	default:
		panic("unsupported option for keyword cluster")
	}
	c.PBSTmpl = pbs
}

// WhichProgram is a helper function for setting Config.EnergyLine
// based on the selected ChemProg
func (c *Config) WhichProgram() {
	switch c.ChemProg {
	case "cccr":
		c.EnergyLine = regexp.MustCompile(`^\s*CCCRE\s+=`)
	case "cart", "gocart":
		CART = true
	case "grad":
		GRAD = true
	case "molpro", "", "sic": // default if not specified
		SIC = true
	default:
		panic("unsupported option for keyword program")
	}
}

// ProcessGeom uses c.Geometry to update c.Deltas and calculate
// c.Ncoords
func (c *Config) ProcessGeom() (cart bool) {
	var (
		ncoords int
		start   int
		end     int
		incr    int
	)
	if c.Geometry == "" {
		panic("no geometry given")
	}
	lines := strings.Split(c.Geometry, "\n")
	gt := c.GeomType
	switch gt {
	case "xyz", "cart":
		start = 2
		end = len(lines)
		cart = true
		incr = 3
	case "zmat":
		end = len(lines)
		incr = 1
	default:
		panic("unable to determine geometry type")
	}
	for _, line := range lines[start:end] {
		if !strings.Contains(line, "=") {
			ncoords += incr
		}
	}
	c.Ncoords = ncoords
	return
	// we could actually do heavier processing of the geometry
	// here, as seen in the Zmat or XYZ functions I think in main
}

// ParseDeltas parses a sequence of step size inputs as a string into
// a slice of floats. Unprovided steps are set to c.Delta. For
// example, the input 1:0.075,4:0.075,7:0.075 yields [0.075, 0.005,
// 0.005, 0.075, 0.005, 0.005, 0.075, 0.005, 0.005], assuming c.Delta
// is 0.005, and c.Ncoord is 9
func (c *Config) ParseDeltas(deltas string) []float64 {
	err := errors.New("invalid deltas input")
	ret := make([]float64, 0)
	if c.Deltas != nil {
		pairs := strings.Split(deltas, ",")
		for _, p := range pairs {
			sp := strings.Split(p, ":")
			if len(sp) != 2 {
				panic(err)
			}
			d, e := strconv.Atoi(strings.TrimSpace(sp[0]))
			if e != nil || d < 1 {
				panic(err)
			}
			f, e := strconv.ParseFloat(strings.TrimSpace(sp[1]), 64)
			if e != nil || f < 0.0 {
				panic(err)
			}
			for d > len(ret) {
				ret = append(ret, c.Delta)
			}
			ret[d-1] = f
		}
	}
	for len(ret) < c.Ncoords {
		ret = append(ret, c.Delta)
	}
	return ret
}

// NewConfig returns a Config with all of the default options set
func NewConfig() Config {
	return Config{
		Cluster:     "maple",
		Package:     "molpro",
		ChemProg:    "sic",
		WorkQueue:   "",
		Delta:       0.005,
		Deltas:      nil,
		Geometry:    "",
		GeomType:    "zmat",
		Flags:       "",
		Deriv:       4,
		JobLimit:    1024,
		ChunkSize:   8,
		CheckInt:    100,
		SleepInt:    60,
		NumCPUs:     1,
		PBSMem:      8,
		IntderCmd:   "",
		SpectroCmd:  "",
		EnergyLine:  regexp.MustCompile(`energy=`),
		QueueSystem: "pbs",
		MolproTmpl:  "molpro.in",
		AnpassTmpl:  "anpass.in",
		IntderTmpl:  "intder.in",
	}
}
