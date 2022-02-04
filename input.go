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
	"text/template"
)

// ProcessInput extracts keywords from a line of input
func ProcessInput(line string) {
	// map of special extractor functions
	var special = map[string]func(string){
		"energyline": func(s string) {
			Conf.EnergyLine = regexp.MustCompile(s)
		},
	}
	split := strings.SplitN(line, "=", 2)
	key, val := strings.ToLower(split[0]), split[1]
	for _, kword := range reflect.VisibleFields(reflect.TypeOf(Conf)) {
		keyname := kword.Name
		if strings.ToLower(keyname) == key {
			loc := reflect.ValueOf(&Conf).Elem().FieldByName(keyname)
			tn := kword.Type.Name()
			f, ok := special[key]
			switch {
			case ok:
				f(val)
			case key == "deltas":
				Conf.Deltas = Conf.ParseDeltas(val)
			case tn == "string":
				loc.SetString(val)
			case tn == "int":
				v, err := strconv.Atoi(val)
				if err != nil {
					e := fmt.Sprintf(
						"couldn't parse %s as an int on line %s",
						val, line)
					panic(e)
				}
				loc.Set(reflect.ValueOf(v))
			case tn == "float64":
				v, err := strconv.ParseFloat(val, 64)
				if err != nil {
					e := fmt.Sprintf(
						"couldn't parse %s as a float on line %s",
						val, line)
					panic(e)
				}
				loc.SetFloat(v)
			default:
				e := fmt.Sprintf("uncaught type %s on line %q", tn, line)
				panic(e)
			}
			break
		}
	}
}

// ParseInfile parses an input file specified by filename and stores
// the results in the global Conf
func ParseInfile(filename string) {
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
	for scanner.Scan() {
		line = scanner.Text()
		switch {
		case strings.TrimSpace(line) == "":
		case len(line) > 0 && line[0] == '#': // comment
		case strings.Contains(line, "}"):
			inblock = false
			ProcessInput(strings.TrimSpace(block.String()))
			block.Reset()
		case strings.Contains(line, "{"):
			keyword := strings.SplitN(line, "{", 2)[0]
			block.WriteString(keyword)
			inblock = true
		case inblock:
			block.WriteString(line + "\n")
		default:
			ProcessInput(line)
		}
	}
	Conf.ProcessGeom()
	if Conf.Deltas == nil {
		Conf.Deltas = Conf.ParseDeltas("")
	}
	Conf.WhichProgram()
}

type Config struct {
	Cluster     string
	Package     string // quantum chemistry package (molpro|g16)
	Program     string
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
	Intder      string
	Spectro     string
	Ncoords     int
	EnergyLine  *regexp.Regexp
	PBSTmpl     *template.Template
	QueueSystem string
	MolproTmpl  string
	AnpassTmpl  string
	IntderTmpl  string
}

// NewConfig returns a Config with all of the default options set
func NewConfig() Config {
	return Config{
		Cluster:     "maple",
		Package:     "molpro",
		Program:     "sic",
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
		Intder:      "",
		Spectro:     "",
		Ncoords:     0,
		EnergyLine:  regexp.MustCompile(`energy=`),
		PBSTmpl:     pbsMaple,
		QueueSystem: "pbs",
		MolproTmpl:  "molpro.in",
		AnpassTmpl:  "anpass.in",
		IntderTmpl:  "intder.in",
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
	switch c.Program {
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

// TODO flag for reading pbs template file
// - need to update docs to include that
// - dump subcommand to dump the internal default

// if joblimit is not a multiple of chunksize, we should
// increase joblimit until it is
// if some fields are not present, need to error
// - geometry and the cmds I think are the only ones
//   - and the latter only if needed
//     (not intder and anpass in carts)
