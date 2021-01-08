package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
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

func NewConfig(input [NumKeys]string) (conf Configuration) {
	if input[JobLimit] != "" {
		v, err := strconv.Atoi(input[JobLimit])
		if err == nil {
			jobLimit = v
		}
	}
	if input[ChunkSize] != "" {
		v, err := strconv.Atoi(input[ChunkSize])
		if err == nil {
			chunkSize = v
		}
	}
	if input[Deriv] != "" {
		d, err := strconv.Atoi(input[Deriv])
		if err != nil {
			panic(fmt.Sprintf("%v parsing derivative level input: %q\n",
				err, Config.Deriv))
		}
		nDerivative = d
	}
	if input[NumJobs] != "" {
		d, err := strconv.Atoi(input[NumJobs])
		if err != nil {
			panic(fmt.Sprintf("%v parsing number of jobs input: %q\n",
				err, Config.NumJobs))
		}
		numJobs = d
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
		delta = f
	}
	// always parse deltas to fill with default even if no input
	geom := strings.Split(input[Geometry], "\n")
	if input[GeomType] == "xyz" {
		conf.Ncoords = 3 * (len(geom) - 2)
	} else {
		conf.Ncoords = len(geom)
	}
	var err error
	conf.Deltas, err = ParseDeltas(input[Deltas], conf.Ncoords)
	if err != nil {
		panic(fmt.Sprintf("%v parsing deltas input: %v\n", err, Config.Deltas))
	}
	WhichCluster(input[QueueType])
	switch Config.Program {
	case "cccr":
		energyLine = regexp.MustCompile(`^\s*CCCRE\s+=`)
	case "cart", "gocart":
		flags |= CART
		fmt.Printf("%d coords requires %d points\n", conf.Ncoords, totalPoints(conf.Ncoords))
		energyLine = regexp.MustCompile(`energy=`)
	case "grad":
		flags |= GRAD
		energyLine = regexp.MustCompile(`energy=`)
	case "molpro", "": // default if not specified
		energyLine = regexp.MustCompile(`energy=`)
	default:
		errExit(fmt.Errorf("%s not implemented as a Program", Config.Program), "")
	}
	return Configuration{}
}
