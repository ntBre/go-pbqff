package main

import (
	"regexp"
	"strings"
)

type Key int

const (
	QueueType Key = iota
	Program
	Geometry
	GeomType
	Method
	Basis
	Charge
	Spin
	NumKeys
)

type Regexp struct {
	*regexp.Regexp
	Name Key
}

func ParseInfile(filename string) {
	lines := ReadFile(filename)
	Keywords := []Regexp{
		Regexp{regexp.MustCompile(`(?i)queuetype=`), QueueType},
		Regexp{regexp.MustCompile(`(?i)program=`), Program},
		Regexp{regexp.MustCompile(`(?i)method=`), Method},
		Regexp{regexp.MustCompile(`(?i)geomtype=`), GeomType},
		Regexp{regexp.MustCompile(`(?i)basis=`), Basis},
		Regexp{regexp.MustCompile(`(?i)charge=`), Charge},
		Regexp{regexp.MustCompile(`(?i)spin=`), Spin},
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
					Input[kword.Name] = strings.ToUpper(split[len(split)-1])
				}
			}
			i++
		}
	}
}
