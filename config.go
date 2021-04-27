// Could actually have accessor methods for each of these with the
// right types
package main

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Key is a type for input keyword indices
type Key int

// Keys in the configuration array. To add a new Keyword, add a Key
// here and to the String method below, then add its field to Conf
// below. If it requires other Keywords to fully process, add a method
// on Config and call it at the end of ParseInfile in input.go.
const (
	Cluster Key = iota
	ChemProg
	Queue
	Delta
	Deltas
	Geometry
	GeomType
	Flags
	Deriv
	JobLimit
	ChunkSize
	CheckInt
	SleepInt
	NumJobs
	IntderCmd
	AnpassCmd
	SpectroCmd
	Ncoords
	EnergyLine
	PBS
	MolproTmpl
	AnpassTmpl
	IntderTmpl
	NumKeys
)

func (k Key) String() string {
	return []string{
		"Cluster",
		"ChemProg",
		"Queue",
		"Delta",
		"Deltas",
		"Geometry",
		"GeomType",
		"Flags",
		"Deriv",
		"JobLimit",
		"ChunkSize",
		"CheckInt",
		"SleepInt",
		"NumJobs",
		"IntderCmd",
		"AnpassCmd",
		"SpectroCmd",
		"Ncoords",
		"EnergyLine",
		"PBS",
		"MolproTmpl",
		"AnpassTmpl",
		"IntderTmpl",
	}[k]
}

// If generics are ever added, this becomes

// type Keyword[T any] struct {
//   *regexp.Regexp
//   Extract func(string) T
//   Value T
// }

// and I can get rid of all these stupid conversion methods

// At would become something like

// func (c *Config) [T any] At(k Key) T {

// Nope, no parameterized methods, need a different approach. I guess I
// would just use Config[Key].Value since that's basically what at was
// doing anyway, and the main advantage was not having to do the type
// casting myself.

type Keyword struct {
	Re      *regexp.Regexp
	Extract func(string) interface{}
	Value   interface{}
}

type Config [NumKeys]Keyword

// At returns the Value of c at k
func (c *Config) At(k Key) interface{} {
	return (*c)[k].Value
}

// Set sets the Value of c at k
func (c *Config) Set(k Key, val interface{}) {
	(*c)[k].Value = val
}

func (c *Config) Str(k Key) string {
	return (*c)[k].Value.(string)
}

func (c *Config) Float(k Key) float64 {
	return (*c)[k].Value.(float64)
}

func (c *Config) FlSlice(k Key) []float64 {
	return (*c)[k].Value.([]float64)
}

func (c *Config) Int(k Key) int {
	return (*c)[k].Value.(int)
}

func (c *Config) RE(k Key) *regexp.Regexp {
	return (*c)[k].Value.(*regexp.Regexp)
}

func (c Config) String() string {
	var buf strings.Builder
	for i, kw := range c {
		fmt.Fprintf(&buf, "%s: %v\n", Key(i), kw.Value)
	}
	return buf.String()
}

// WhichCluster is a helper function for setting Config.EnergyLine and
// Config.PBS based on the selected Cluster
func (c *Config) WhichCluster() {
	cluster := c.Str(Cluster)
	sequoia := regexp.MustCompile(`(?i)sequoia`)
	maple := regexp.MustCompile(`(?i)maple`)
	var pbs string
	switch {
	case cluster == "", maple.MatchString(cluster):
		pbs = pbsMaple
	case sequoia.MatchString(cluster):
		c.Set(EnergyLine, regexp.MustCompile(`PBQFF\(2\)`))
		pbs = pbsSequoia
	default:
		panic("unsupported option for keyword cluster")
	}
	c.Set(PBS, pbs)
}

// WhichProgram is a helper function for setting Config.EnergyLine
// based on the selected ChemProg
func (c *Config) WhichProgram() {
	switch c.Str(ChemProg) {
	case "cccr":
		c.Set(EnergyLine, regexp.MustCompile(`^\s*CCCRE\s+=`))
	case "cart", "gocart":
		flags |= CART
	case "grad":
		// TODO count points for grad
		flags |= GRAD
	case "molpro", "", "sic": // default if not specified
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
	if c.At(Geometry) == nil {
		panic("no geometry given")
	}
	lines := strings.Split(c.Str(Geometry), "\n")
	gt := c.At(GeomType)
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
	c.Set(Ncoords, ncoords)
	return
	// we could actually do heavier processing of the geometry
	// here, as seen in the Zmat or XYZ functions I think in main
}

// ParseDeltas parses a sequence of step size inputs as a string into
// a slice of floats. Unprovided steps are set to c.Delta. For
// example, the input 1:0.075,4:0.075,7:0.075 yields [0.075, 0.005,
// 0.005, 0.075, 0.005, 0.005, 0.075, 0.005, 0.005], assuming c.Delta
// is 0.005, and c.Ncoord is 9
func (c *Config) ParseDeltas() {
	err := errors.New("invalid deltas input")
	ret := make([]float64, 0)
	if c.At(Deltas) == nil {
		panic("no deltas to parse")
	}
	pairs := strings.Split(c.Str(Deltas), ",")
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
			ret = append(ret, c.Float(Delta))
		}
		ret[d-1] = f
	}
	for len(ret) < c.Int(Ncoords) {
		ret = append(ret, c.Float(Delta))
	}
	c.Set(Deltas, ret)
}

func kwpanic(str string, err error) {
	panic(
		fmt.Sprintf(
			"%v parsing input line %q\n",
			err, str),
	)
}

func StringKeyword(str string) interface{} {
	return str
}

func FloatKeyword(str string) interface{} {
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		kwpanic(str, err)
	}
	return f
}

func IntKeyword(str string) interface{} {
	v, err := strconv.Atoi(str)
	if err != nil {
		kwpanic(str, err)
	}
	return v
}

var Conf = Config{
	Cluster: {
		Re:      regexp.MustCompile(`(?i)queuetype=`),
		Extract: StringKeyword,
		Value:   "maple",
	},
	ChemProg: {
		Re:      regexp.MustCompile(`(?i)program=`),
		Extract: StringKeyword,
		Value:   "sic",
	},
	Queue: { // TODO these queues are maple-specific
		Re: regexp.MustCompile(`(?i)queue=`),
		Extract: func(str string) interface{} {
			switch str {
			case "workq", "r410", "":
			default:
				panic("unsupported option for keyword queue")
			}
			return str
		},
		// possible problem using this in template if ""
		// doesn't satisify {{if .Field}} = False
		// just delete and leave nil if not
		Value: "",
	},
	Delta: {
		Re:      regexp.MustCompile(`(?i)delta=`),
		Extract: FloatKeyword,
		Value:   0.005,
	},
	Deltas: {
		Re:      regexp.MustCompile(`(?i)deltas=`),
		Extract: StringKeyword,
	},
	Geometry: {
		Re:      regexp.MustCompile(`(?i)geometry=`),
		Extract: StringKeyword,
	},
	GeomType: {
		Re:      regexp.MustCompile(`(?i)geomtype=`),
		Extract: StringKeyword,
		Value:   "zmat",
	},
	Flags: {
		Re: regexp.MustCompile(`(?i)flags=`),
		Extract: func(str string) interface{} {
			switch str {
			case "noopt":
				flags = flags &^ OPT
			default:
				panic("unsupported option for keyword flag")
			}
			return str
		},
	},
	Deriv: {
		Re:      regexp.MustCompile(`(?i)deriv=`),
		Extract: IntKeyword,
		Value:   4,
	},
	JobLimit: {
		Re:      regexp.MustCompile(`(?i)joblimit=`),
		Extract: IntKeyword,
		Value:   1024,
	},
	ChunkSize: {
		Re:      regexp.MustCompile(`(?i)chunksize=`),
		Extract: IntKeyword,
		Value:   64,
	},
	CheckInt: {
		Re: regexp.MustCompile(`(?i)checkint=`),
		Extract: func(str string) interface{} {
			switch str {
			case "no":
				nocheck = true
				return 0
			default:
				return IntKeyword(str)
			}
		},
		Value: 100,
	},
	SleepInt: {
		Re:      regexp.MustCompile(`(?i)sleepint=`),
		Extract: IntKeyword,
		Value:   60,
	},
	NumJobs: {
		Re:      regexp.MustCompile(`(?i)numjobs=`),
		Extract: IntKeyword,
		Value:   8,
	},
	IntderCmd: {
		Re:      regexp.MustCompile(`(?i)intder=`),
		Extract: StringKeyword,
	},
	AnpassCmd: {
		Re:      regexp.MustCompile(`(?i)anpass=`),
		Extract: StringKeyword,
	},
	SpectroCmd: {
		Re:      regexp.MustCompile(`(?i)spectro=`),
		Extract: StringKeyword,
	},
	EnergyLine: {
		Value: regexp.MustCompile(`energy=`),
	},
	MolproTmpl: {
		Re:      regexp.MustCompile(`(?i)molprotmpl=`),
		Extract: StringKeyword,
		Value:   "molpro.in",
	},
	AnpassTmpl: {
		Re:      regexp.MustCompile(`(?i)anpasstmpl=`),
		Extract: StringKeyword,
		Value:   "anpass.in",
	},
	IntderTmpl: {
		Re:      regexp.MustCompile(`(?i)intdertmpl=`),
		Extract: StringKeyword,
		Value:   "intder.in",
	},
}
