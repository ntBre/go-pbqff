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

// Keys in the configuration array
const (
	Cluster Key = iota
	Program
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
)

func (k Key) String() string {
	return []string{
		"Cluster", "Program", "Queue",
		"Delta", "Deltas", "Geometry",
		"GeomType", "Flags", "Deriv",
		"JobLimit", "ChunkSize", "CheckInt",
		"SleepInt", "NumJobs", "IntderCmd",
		"AnpassCmd", "SpectroCmd", "Ncoords",
		"EnergyLine", "PBS",
	}[k]
}

type Keyword struct {
	Re      *regexp.Regexp
	Extract func(string) interface{}
	Value   interface{}
}

type Conf []Keyword

func (c *Conf) At(k int) interface{} {
	return (*c)[Key(k)].Value
}

func (c *Conf) Set(k Key, val interface{}) {
	(*c)[k].Value = val
}

func (c *Conf) Str(k Key) string {
	return (*c)[k].Value.(string)
}

func (c *Conf) FlSlice(k Key) []float64 {
	return (*c)[k].Value.([]float64)
}

func (c *Conf) Int(k Key) int {
	return (*c)[k].Value.(int)
}

func (c *Conf) RE(k Key) *regexp.Regexp {
	return (*c)[k].Value.(*regexp.Regexp)
}

func (c Conf) String() string {
	var buf strings.Builder
	for i, kw := range c {
		fmt.Fprintf(&buf, "%s: %v\n", Key(i), kw.Value)
	}
	return buf.String()
}

func StringKeyword(str string) interface{} {
	return str
}

func kwpanic(str string, err error) {
	panic(
		fmt.Sprintf(
			"%v parsing input line %q\n",
			err, str),
	)
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

// ParseDeltas parses a sequence of step size inputs as a string into
// a slice of floats. Unprovided steps are set to c.Delta. For
// example, the input 1:0.075,4:0.075,7:0.075 yields [0.075, 0.005,
// 0.005, 0.075, 0.005, 0.005, 0.075, 0.005, 0.005], assuming c.Delta
// is 0.005, and c.Ncoord is 9
func ParseDeltas(deltas string) interface{} {
	err := errors.New("invalid deltas input")
	ret := make([]float64, 0)
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
			ret = append(ret, -1.0)
		}
		ret[d-1] = f
	}
	return ret
}

// WhichCluster is a helper function for setting Config.EnergyLine and
// Config.PBS based on the selected Cluster
func (c *Conf) WhichCluster(cluster string) {
	sequoia := regexp.MustCompile(`(?i)sequoia`)
	maple := regexp.MustCompile(`(?i)maple`)
	var (
		pbs   string
		eline *regexp.Regexp
	)
	switch {
	case cluster == "", maple.MatchString(cluster):
		pbs = pbsMaple
	case sequoia.MatchString(cluster):
		eline = regexp.MustCompile(`PBQFF\(2\)`)
		pbs = pbsSequoia
	default:
		panic("unsupported option for keyword cluster")
	}
	c.Set(PBS, pbs)
	c.Set(EnergyLine, eline)
}

// WhichProgram is a helper function for setting Config.EnergyLine
// based on the selected Program
func (c *Conf) WhichProgram(str string) {
	eline := regexp.MustCompile(`energy=`)
	switch str {
	case "cccr":
		eline = regexp.MustCompile(`^\s*CCCRE\s+=`)
	case "cart", "gocart":
		flags |= CART
	case "grad":
		// TODO count points for grad
		flags |= GRAD
	case "molpro", "", "sic": // default if not specified
	default:
		panic("unsupported option for keyword program")
	}
	c.Set(EnergyLine, eline)
}

// TODO use WhichX at end of parseinfile to break recursion

// give each one its own func to check defaults instead of generic
// XKeyword funs
var Config = Conf{
	Cluster: {
		Re:      regexp.MustCompile(`(?i)queuetype=`),
		Extract: StringKeyword,
		Value:   "maple",
	},
	Program: {
		Re:      regexp.MustCompile(`(?i)program=`),
		Extract: StringKeyword,
		Value:   "sic",
	},
	Queue: { // TODO these queues are maple-specific
		Re: regexp.MustCompile(`(?i)queue=`),
		Extract: func(str string) interface{} {
			switch str {
			case "workq", "r410":
			default:
				panic("unsupported option for keyword queue")
			}
			return str
		},
		Value: "",
	},
	Delta: {
		Re:      regexp.MustCompile(`(?i)delta=`),
		Extract: FloatKeyword,
		Value:   0.005,
	},
	Deltas: {
		Re:      regexp.MustCompile(`(?i)deltas=`),
		Extract: ParseDeltas,
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
}

// TODO set Ncoords somewhere, prob in geom parser or after that somehow
// this is how I did it before
// geom := strings.Split(input[Geometry], "\n")
// if input[GeomType] == "xyz" {
// 	conf.Ncoords = 3 * (len(geom) - 2)
// } else {
// 	conf.Ncoords = len(geom)
// }

// also print this after:
// fmt.Printf("%d coords requires %d points\n",
// 	conf.Ncoords, totalPoints(conf.Ncoords))

// check if geometry was given, if not it's an error so just parse
// ncoords from that at the end of ParseInfile

// after that we can make sure deltas is the right shape
