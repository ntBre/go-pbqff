package main

import (
	"bufio"
	"fmt"
	"os"
)

type Mopac struct {
	Dir  string
	Head string
	Opt  string
	Body string
	Geom string
	Tail string
}

func (m *Mopac) SetDir(dir string)   { m.Dir = dir }
func (m *Mopac) GetDir() string      { return m.Dir }
func (m *Mopac) SetGeom(geom string) { m.Geom = geom }
func (m *Mopac) GetGeom() string     { return m.Geom }

// Load a MOPAC input file from filename
func (m *Mopac) Load(filename string) error {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(f)
	var line string
	// TODO look up MOPAC input format and get some examples to
	// test on
	for scanner.Scan() {
		line = scanner.Text()
		fmt.Println(line)
	}
	return nil
}
