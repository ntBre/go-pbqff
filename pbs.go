package main

import "os"

type PBS struct {
	Name     string
	Filename string
}

func (p *PBS) WriteInput(infile, tfile string) {
	f, err := os.Create(infile)
	if err != nil {
		panic(err)
	}
	t := LoadTemplate(tfile)
	t.Execute(f, p)
}
