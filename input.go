package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type RawConf map[string]string

func (rc *RawConf) Add(line string) {
	split := strings.SplitN(line, "=", 2)
	key, val := strings.ToLower(split[0]), split[1]
	(*rc)[key] = val
}

func (rc *RawConf) ToConfig() Config {
	ret := NewConfig()
	// map of special extractor functions
	var special = map[string]func(string){
		"energyline": func(s string) {
			switch s {
			// special case for cccr so you don't have to
			// type this
			case "cccr":
				s = `^\s*CCCRE\s+=`
			}
			ret.EnergyLine = regexp.MustCompile(s)
		},
		"queue": func(s string) {
			switch s {
			case "slurm":
				ret.Queue = &Slurm{
					Tmpl: MolproSlurmTmpl,
				}
			}
		},
		"flags": func(s string) {
			ret.Flags = s
			switch s {
			case "noopt":
				OPT = false
			}
		},
		"deltas": func(s string) { return },
		"program": func(s string) {
			ret.Program = s
			switch s {
			case "cart", "gocart":
				CART = true
			case "grad":
				GRAD = true
			case "molpro", "sic":
				SIC = true
			default:
				panic("unsupported option for keyword program")
			}
		},
	}
	for _, kword := range reflect.VisibleFields(reflect.TypeOf(ret)) {
		keyname := kword.Name
		key := strings.ToLower(keyname)
		val, ok := (*rc)[key]
		if ok {
			loc := reflect.ValueOf(&ret).Elem().FieldByName(keyname)
			tn := kword.Type.Name()
			f, ok := special[key]
			switch {
			case ok:
				f(val)
			case tn == "string":
				loc.SetString(val)
			case tn == "int":
				v, err := strconv.Atoi(val)
				if err != nil {
					e := fmt.Sprintf(
						"couldn't parse %s as an int on line %s",
						val, key)
					panic(e)
				}
				loc.Set(reflect.ValueOf(v))
			case tn == "float64":
				v, err := strconv.ParseFloat(val, 64)
				if err != nil {
					e := fmt.Sprintf(
						"couldn't parse %s as a float on line %s",
						val, key)
					panic(e)
				}
				loc.SetFloat(v)
			default:
				e := fmt.Sprintf("uncaught type %s on line %q", tn, key)
				panic(e)
			}
		}
	}
	ret.ProcessGeom()
	ret.ParseDeltas((*rc)["deltas"])
	// Conf.WhichProgram()
	return ret
}

// ParseInfile parses an input file specified by filename into a
// RawConf
func ParseInfile(filename string) *RawConf {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(f)
	var (
		block   strings.Builder
		inblock bool
		line    string
	)
	ret := make(RawConf)
	for scanner.Scan() {
		line = scanner.Text()
		switch {
		case strings.TrimSpace(line) == "":
		case len(line) > 0 && line[0] == '#': // comment
		case strings.Contains(line, "}"):
			inblock = false
			ret.Add(strings.TrimSpace(block.String()))
			block.Reset()
		case strings.Contains(line, "{"):
			keyword := strings.SplitN(line, "{", 2)[0]
			block.WriteString(keyword)
			inblock = true
		case inblock:
			block.WriteString(line + "\n")
		default:
			ret.Add(line)
		}
	}
	return &ret
}

type Config struct {
	EnergyLine *regexp.Regexp
	Intder     string
	WorkQueue  string
	AnpassTmpl string
	MolproTmpl string
	Geometry   string
	GeomType   string
	Flags      string
	Queue      Queue
	Program    string
	Package    string // quantum chemistry package (molpro|g16)
	Cluster    string
	Spectro    string
	IntderTmpl string
	Deltas     []float64
	PBSMem     int
	SleepInt   int // interval in seconds between polling jobs
	Ncoords    int
	ChunkSize  int     // number of jobs submitted in one group
	JobLimit   int     // maximum number of jobs to run at once
	Deriv      int     // derivative level
	NumCPUs    int     // number of CPUs
	Delta      float64 // step size

	// CheckInt is the interval for writing checkpoints. A zero or
	// negative value disables checkpoints
	CheckInt int
}

// NewConfig returns a Config with all of the default options set
func NewConfig() Config {
	return Config{
		Cluster:    "maple",
		Package:    "molpro",
		Program:    "sic",
		WorkQueue:  "",
		Delta:      0.005,
		Deltas:     nil,
		Geometry:   "",
		GeomType:   "zmat",
		Flags:      "",
		Deriv:      4,
		JobLimit:   1024,
		ChunkSize:  8,
		CheckInt:   100,
		SleepInt:   60,
		NumCPUs:    1,
		PBSMem:     8,
		Intder:     "",
		Spectro:    "",
		Ncoords:    0,
		EnergyLine: regexp.MustCompile(`energy=`),
		Queue: &PBS{
			Tmpl: MolproPBSTmpl,
		},
		MolproTmpl: "molpro.in",
		AnpassTmpl: "anpass.in",
		IntderTmpl: "intder.in",
	}
}

func (c Config) String() string {
	s, _ := json.MarshalIndent(Conf, "", "\t")
	return string(s)
}

// WhichCluster is a helper function for setting Config.EnergyLine and
// Config.PBS based on the selected Cluster
func (c *Config) WhichCluster() {
	cluster := c.Cluster
	sequoia := regexp.MustCompile(`(?i)sequoia`)
	maple := regexp.MustCompile(`(?i)maple`)
	switch {
	case cluster == "", maple.MatchString(cluster):
	case sequoia.MatchString(cluster):
		c.EnergyLine = regexp.MustCompile(`PBQFF\(2\`)
	default:
		panic("unsupported option for keyword cluster")
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
		if len(strings.Fields(lines[0])) == 1 {
			start = 2
		}
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
func (c *Config) ParseDeltas(deltas string) {
	if c.Delta == 0 {
		panic("delta unset before parsing deltas")
	}
	err := errors.New("invalid deltas input")
	if deltas != "" {
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
			for d > len(c.Deltas) {
				c.Deltas = append(c.Deltas, c.Delta)
			}
			c.Deltas[d-1] = f
		}
	}
	for len(c.Deltas) < c.Ncoords {
		c.Deltas = append(c.Deltas, c.Delta)
	}
}

// TODO flag for reading pbs template file
// - need to update docs to include that
// - dump subcommand to dump the internal default

// if joblimit is not a multiple of chunksize, we should
// increase joblimit until it is
// if some fields are not present, need to error
// - geometry and the cmds I think are the only ones
//   - and the latter only if needed
//     (not intder and anpass in carts)
