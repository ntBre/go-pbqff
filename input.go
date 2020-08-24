package main

import (
	"regexp"
	"strings"
)

// Key is a type for input keyword indices
type Key int

// Keys in the configuration array
const (
	QueueType Key = iota
	Program
	Geometry
	GeomType
	Flags
	Deriv
	JobLimit
	ChunkSize
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
func ParseInfile(filename string) {
	lines, err := ReadFile(filename)
	if err != nil {
		panic(err)
	}
	Keywords := []Regexp{
		Regexp{regexp.MustCompile(`(?i)queuetype=`), QueueType},
		Regexp{regexp.MustCompile(`(?i)program=`), Program},
		Regexp{regexp.MustCompile(`(?i)geomtype=`), GeomType},
		Regexp{regexp.MustCompile(`(?i)flags=`), Flags},
		Regexp{regexp.MustCompile(`(?i)deriv=`), Deriv},
		Regexp{regexp.MustCompile(`(?i)joblimit=`), JobLimit},
		Regexp{regexp.MustCompile(`(?i)chunksize=`), ChunkSize},
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
			Input[Geometry] = strings.Join(geomlines, "\n")
		} else {
			for _, kword := range Keywords {
				if kword.MatchString(lines[i]) {
					split := strings.Split(lines[i], "=")
					Input[kword.Name] = split[len(split)-1]
				}
			}
			i++
		}
	}
}
