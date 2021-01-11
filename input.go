package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"reflect"
)

// Key is a type for input keyword indices
type Key int

// Keys in the configuration array
const (
	QueueType Key = iota
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
	NumKeys
)

// Regexp combines a *regexp.Regexp and a Key
type Regexp struct {
	*regexp.Regexp
	Name Key
}

// ParseInfile parses an input file specified by filename and stores
// the results in the array Input
func ParseInfile(filename string) (input [NumKeys]string) {
	lines, err := ReadFile(filename)
	if err != nil {
		panic(err)
	}
	Keywords := []Regexp{
		Regexp{regexp.MustCompile(`(?i)queuetype=`), QueueType},
		Regexp{regexp.MustCompile(`(?i)program=`), Program},
		Regexp{regexp.MustCompile(`(?i)queue=`), Queue},
		Regexp{regexp.MustCompile(`(?i)delta=`), Delta},
		Regexp{regexp.MustCompile(`(?i)deltas=`), Deltas},
		Regexp{regexp.MustCompile(`(?i)geomtype=`), GeomType},
		Regexp{regexp.MustCompile(`(?i)flags=`), Flags},
		Regexp{regexp.MustCompile(`(?i)deriv=`), Deriv},
		Regexp{regexp.MustCompile(`(?i)joblimit=`), JobLimit},
		Regexp{regexp.MustCompile(`(?i)chunksize=`), ChunkSize},
		Regexp{regexp.MustCompile(`(?i)checkint=`), CheckInt},
		Regexp{regexp.MustCompile(`(?i)sleepint=`), SleepInt},
		Regexp{regexp.MustCompile(`(?i)numjobs=`), NumJobs},
		Regexp{regexp.MustCompile(`(?i)intder=`), IntderCmd},
		Regexp{regexp.MustCompile(`(?i)anpass=`), AnpassCmd},
		Regexp{regexp.MustCompile(`(?i)spectro=`), SpectroCmd},
	}
	geom := regexp.MustCompile(`(?i)geometry={`)
	for i := 0; i < len(lines); {
		if lines[i][0] == '#' {
			i++
			continue
		}
		if geom.MatchString(lines[i]) {
			i++
			geomlines := make([]string, 0)
			for !strings.Contains(lines[i], "}") {
				geomlines = append(geomlines, lines[i])
				i++
			}
			input[Geometry] = strings.Join(geomlines, "\n")
		} else {
			for _, kword := range Keywords {
				if kword.MatchString(lines[i]) {
					split := strings.Split(lines[i], "=")
					input[kword.Name] = split[len(split)-1]
				}
			}
			i++
		}
	}
	return
}

// ParseDeltas parses a sequence of step sizes input as a string into
// a slice of floats. Unprovided steps are set to c.Delta. For
// example, the input 1:0.075,4:0.075,7:0.075 yields [0.075, 0.005,
// 0.005, 0.075, 0.005, 0.005, 0.075, 0.005, 0.005], assuming c.Delta
// is 0.005, and c.Ncoord is 9
func (c *Configuration) ParseDeltas(deltas string) (err error) {
	// assume problem
	err = errors.New("invalid deltas input")
	c.Deltas = make([]float64, c.Ncoords)
	// set up defaults
	for i := range c.Deltas {
		c.Deltas[i] = c.Delta
	}
	if len(deltas) == 0 {
		err = nil
		return
	}
	pairs := strings.Split(deltas, ",")
	for _, p := range pairs {
		sp := strings.Split(p, ":")
		if len(sp) != 2 {
			return
		}
		d, e := strconv.Atoi(strings.TrimSpace(sp[0]))
		if e != nil || d > c.Ncoords || d < 1 {
			return
		}
		f, e := strconv.ParseFloat(strings.TrimSpace(sp[1]), 64)
		if e != nil || f < 0.0 {
			return
		}
		c.Deltas[d-1] = f
	}
	err = nil
	return
}

// WhichCluster is a helper function for setting global variables
// depending on the QueueType keyword
func WhichCluster(q string) {
	sequoia := regexp.MustCompile(`(?i)sequoia`)
	maple := regexp.MustCompile(`(?i)maple`)
	switch {
	case q == "", maple.MatchString(q):
		pbs = pbsMaple
	case sequoia.MatchString(q):
		energyLine = regexp.MustCompile(`PBQFF\(2\)`)
		pbs = pbsSequoia
	default:
		panic("no queue selected " + q)
	}
}

type Configuration struct {
	QueueType  string
	Program    string
	Queue      string
	Delta      float64
	Deltas     []float64
	Geometry   string
	GeomType   string
	Flags      string
	Deriv      int
	JobLimit   int
	ChunkSize  int
	CheckInt   int
	SleepInt   int
	NumJobs    int
	IntderCmd  string
	AnpassCmd  string
	SpectroCmd string
	Ncoords    int
}

func (c Configuration) String() string {
	ret, _ := json.MarshalIndent(c, "", "\t")
	return string(ret)
}

func Defaults() Configuration {
	return Configuration{
		QueueType: "maple",
		Program:   "molpro",
		Delta:     0.005,
		GeomType:  "zmat",
		Deriv:     4,
		JobLimit:  1024,
		ChunkSize: 64,
		CheckInt:  100,
		SleepInt:  1,
		NumJobs:   8,
	}
}

func parseInt(str string) int {
	v, err := strconv.Atoi(str)
	if err != nil {
		e := fmt.Sprintf(
			"%v parsing input line %q\n",
			err, str)
		panic(e)
	}
	return v
}

// if joblimit is not a multiple of chunksize, we should
// increase joblimit until it is
// if some fields are not present, need to error
// - geometry and the cmds I think are the only ones
//   - and the latter only if needed
//     (not intder and anpass in carts)

func NewConfig(input [NumKeys]string) (conf Configuration) {
	conf = Defaults()
	conf.Geometry = input[Geometry]
	conf.IntderCmd = input[IntderCmd]
	conf.AnpassCmd = input[AnpassCmd]
	conf.SpectroCmd = input[SpectroCmd]
	WhichCluster(input[QueueType])
	if input[JobLimit] != "" {
		conf.JobLimit = parseInt(input[JobLimit])
	}
	if input[ChunkSize] != "" {
		conf.ChunkSize = parseInt(input[ChunkSize])
	}
	if input[Deriv] != "" {
		conf.Deriv = parseInt(input[Deriv])
	}
	if input[NumJobs] != "" {
		conf.NumJobs = parseInt(input[NumJobs])
	}
	if s := input[SleepInt]; s != "" {
		d, err := strconv.Atoi(s)
		if err != nil {
			panic(fmt.Sprintf("%v parsing sleep interval: %q\n", err, s))
		}
		sleep = d
	}
	switch input[CheckInt] {
	case "no":
		nocheck = true
	case "":
	default:
		d, err := strconv.Atoi(input[CheckInt])
		if err != nil {
			panic(fmt.Sprintf("%v parsing checkpoint interval: %q\n",
				err, input[CheckInt]))
		}
		checkAfter = d
	}
	if input[Delta] != "" {
		f, err := strconv.ParseFloat(input[Delta], 64)
		if err != nil {
			panic(fmt.Sprintf("%v parsing delta input: %q\n", err, input[Delta]))
		}
		Config.Delta = f
	}
	// always parse deltas to fill with default even if no input
	geom := strings.Split(input[Geometry], "\n")
	if input[GeomType] == "xyz" {
		conf.Ncoords = 3 * (len(geom) - 2)
	} else {
		conf.Ncoords = len(geom)
	}
	var err error
	err = conf.ParseDeltas(input[Deltas])
	if err != nil {
		panic(fmt.Sprintf("%v parsing deltas input: %v\n",
			err, Config.Deltas))
	}
	switch Config.Program {
	case "cccr":
		energyLine = regexp.MustCompile(`^\s*CCCRE\s+=`)
	case "cart", "gocart":
		flags |= CART
		fmt.Printf("%d coords requires %d points\n",
			conf.Ncoords, totalPoints(conf.Ncoords))
		energyLine = regexp.MustCompile(`energy=`)
	case "grad":
		flags |= GRAD
		energyLine = regexp.MustCompile(`energy=`)
	case "molpro", "": // default if not specified
		energyLine = regexp.MustCompile(`energy=`)
	default:
		errExit(fmt.Errorf("%s not implemented as a Program", Config.Program), "")
	}
	return
}
