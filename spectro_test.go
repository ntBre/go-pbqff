package main

import (
	"reflect"
	"strings"
	"testing"
)

var (
	names  = []string{"Al", "O", "O", "Al"}
	coords = `0.000000000        2.391678166        0.000000000
     -2.274263181        0.000000000        0.000000000
      2.274263181        0.000000000        0.000000000
      0.000000000       -2.391678166        0.000000000
`
)

func TestLoadSpectro(t *testing.T) {
	got, _ := LoadSpectro("testfiles/load/spectro.in",
		[]string{"N", "C", "O", "H"},
		`0.0000000000       -0.0115666469        2.4598228639
      0.0000000000      -0.0139207809       0.2726915161
      0.0000000000       0.1184234620      -2.1785371074
      0.0000000000      -1.5591967852      -2.8818447886
`)
	want := &Spectro{
		Head: `# SPECTRO ##########################################
    1    1    3    2    0    0    1    4    0    1    0    0    0    0    0
    1    1    1    0    0    1    0    0    0    0    0    0    0    0    0
# GEOM #######################################
`,
		Geometry: `   4   1
 7.00      0.0000000000     -0.0115666469      2.4598228639
 6.00      0.0000000000     -0.0139207809      0.2726915161
 8.00      0.0000000000      0.1184234620     -2.1785371074
 1.00      0.0000000000     -1.5591967852     -2.8818447886
`,
		Body: `# WEIGHT ###### 
    4    
    1   14.003074
    2   12.0    
    3   15.9949146
    4    1.007825 
# CURVIL ##########################################
    1    2      
    2    3      
    3    4      
    3    2    4
    4    3    2    1
    4    3    2    1
`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%#v\nwanted\n%#v\n", got, want)
	}
}

func TestWriteSpectroInput(t *testing.T) {
	tests := []struct {
		load   string
		names  []string
		coords string
		write  string
		right  string
	}{
		{
			load:   "testfiles/load/spectro.in",
			names:  names,
			coords: coords,
			write:  "testfiles/write/spectro.in",
			right:  "testfiles/right/spectro.in",
		},
	}
	for _, test := range tests {
		spec, _ := LoadSpectro(test.load, test.names, test.coords)
		spec.WriteInput(test.write)
		if !compareFile(test.write, test.right) {
			t.Errorf("mismatch between %s and %s\n", test.write, test.right)
		}
	}
}

func TestReadSpectroOutput(t *testing.T) {
	t.Run("all resonances present", func(t *testing.T) {
		spec, _ := LoadSpectro("testfiles/load/spectro.in", names, coords)
		spec.ReadOutput("testfiles/read/spectro.out")
	})
	t.Run("no fermi 2 resonances present", func(t *testing.T) {
		spec, _ := LoadSpectro("testfiles/load/spectro.in", names, coords)
		spec.ReadOutput("testfiles/read/spectro.prob")
	})
	t.Run("no coriolis resonances present", func(t *testing.T) {
		spec, _ := LoadSpectro("testfiles/load/spectro.in", names, coords)
		spec.ReadOutput("testfiles/read/spectro.nocoriol")
	})
}

func polyEqual(p1, p2 string) bool {
	if len(p1) != len(p2) {
		return false
	}
	sp1 := strings.Split(p1, "\n")
	sp2 := strings.Split(p2, "\n")
	if sp1[0] != sp2[0] || sp1[1] != sp2[1] {
		return false
	}
	sp1 = sp1[2:]
	sp2 = sp2[2:]
	var found bool
	for i := range sp1 {
		found = false
		for j := range sp2 {
			if sp1[i] == sp2[j] {
				found = true
				sp2 = append(sp2[:j], sp2[j+1:]...)
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func TestCheckPolyad(t *testing.T) {
	spec, _ := LoadSpectro("testfiles/load/spectro.in", names, coords)
	spec.Nfreqs = 6
	spec.ReadOutput("testfiles/read/spectro.out")
	got := spec.Polyad
	want := `    1
    5
    0    0    0    1    0    0
    0    1    0    0    0    0
    0    0    0    2    0    0
    0    0    0    0    2    0
    0    0    1    1    0    0
`
	if !polyEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestMakeKey(t *testing.T) {
	got := MakeKey([]int{1, 2, 3})
	want := "1 2 3"
	if got != want {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestResinLine(t *testing.T) {
	t.Run("One frequency on lhs", func(t *testing.T) {
		got := ResinLine(6, 2, 2)
		want := "    0    2    0    0    0    0\n"
		if got != want {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
	t.Run("two frequencies on lhs", func(t *testing.T) {
		got := ResinLine(6, 1, 2, 1)
		want := "    1    1    0    0    0    0\n"
		if got != want {
			t.Errorf("got %v, wanted %v\n", got, want)
		}
	})
}
