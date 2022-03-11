package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/ntBre/anpass"
)

func TestLoadAnpass(t *testing.T) {
	tests := []struct {
		file string
		fmt1 string
		fmt2 string
	}{
		{
			file: "testfiles/load/anpass.small",
			fmt1: "%12.8f",
			fmt2: "%20.12f",
		},
		{
			file: "testfiles/anpass.prob",
			fmt1: "%12.8f",
			fmt2: "%20.12f",
		},
	}
	for _, test := range tests {
		got, _ := LoadAnpass(test.file)
		if got.Fmt1 != test.fmt1 {
			t.Errorf("got %#v, wanted %#v\n", got.Fmt1, test.fmt1)
		}
		if got.Fmt2 != test.fmt2 {
			t.Errorf("got %#v, wanted %#v\n", got.Fmt2, test.fmt2)
		}
	}
}

func TestWriteAnpass(t *testing.T) {
	tests := []struct {
		load  string
		write string
		right string
	}{
		{
			load:  "testfiles/load/anpass.small",
			write: "testfiles/write/anpass.test",
			right: "testfiles/right/anpass.test",
		},
	}
	for _, test := range tests {
		a, _ := LoadAnpass(test.load)
		a.WriteAnpass(test.write, []float64{0, 0, 0, 0, 0, 0}, nil)
		if !compareFile(test.write, test.right) {
			fmt.Printf("(diff %q %q)\n", test.write, test.right)
			t.Errorf("mismatch between %s and %s\n", test.write, test.right)
		}
	}
}

func TestWriteAnpass2(t *testing.T) {
	tests := []struct {
		load  string
		lline string
		write string
		right string
	}{
		{
			load:  "testfiles/load/anpass.small",
			lline: "testfiles/read/anpass1.out",
			write: "testfiles/write/anpass2.test",
			right: "testfiles/right/anpass2.test",
		},
	}
	for _, test := range tests {
		a, _ := LoadAnpass(test.load)
		ll, _ := GetLongLine(test.lline)
		a.WriteAnpass2(test.write, ll, []float64{0, 0, 0, 0, 0, 0}, nil)
		if !compareFile(test.write, test.right) {
			t.Errorf("mismatch between %s and %s\n", test.write, test.right)
		}
	}
}

func TestGetLongLine(t *testing.T) {
	got, _ := GetLongLine("testfiles/read/anpass1.out")
	want := `     -0.000879072913     -0.000974769219     -0.000489284859     -0.000744291296      0.000772915057      0.000000000000     -0.000002937018`
	if got != want {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func loadvec(filename string) (ret []float64) {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 {
			v, err := strconv.ParseFloat(
				strings.TrimSpace(line), 64,
			)
			if err != nil {
				panic(err)
			}
			ret = append(ret, v)
		}
	}
	return
}

func TestFromIntder(t *testing.T) {
	energies := loadvec("testfiles/read/ally.dat")
	got := FromIntder("testfiles/read/ally.in", energies, true)
	f, err := os.Open("testfiles/read/ally.anpass.mid")
	if err != nil {
		panic(err)
	}
	byts, _ := io.ReadAll(f)
	want := string(byts)
	if got != want {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

// load a fort.9903 file produced by anpass. borrowed from anpass package
func load9903(filename string) (ret []anpass.FC) {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		hold := new(anpass.FC)
		if len(fields) == 5 {
			for i := range fields[:4] {
				hold.Coord[i], _ = strconv.Atoi(fields[i])
			}
			hold.Val, _ = strconv.ParseFloat(fields[4], 64)
			ret = append(ret, *hold)
		}
	}
	return
}

func TestFormat9903(t *testing.T) {
	initArrays(3)
	fcs := load9903("testfiles/load/anpass.9903")
	Format9903(9, fcs)
	want := []float64{
		0.000015066623,
		-0.003795299437,
		-0.002528188120,
		-0.000226897901,
		0.003228815360,
		0.001970232132,
		-0.000082519062,
		0.000252427783,
		0.000206156134,

		-0.003795299437,
		5.831072988783,
		3.659158992343,
		0.002896727442,
		-5.345602947917,
		-3.171968084495,
		0.000210528845,
		-0.485766171906,
		-0.487720511535,

		-0.002528188120,
		3.659158992343,
		3.398417880833,
		0.002235343484,
		-4.147172754881,
		-3.579800654096,
		-0.000411609150,
		0.487471423749,
		0.181124721694,

		-0.000226897901,
		0.002896727442,
		0.002235343484,
		-0.000012609181,
		-0.006294359154,
		-0.000279054040,
		-0.000035643028,
		0.003194478242,
		-0.002232267666,

		0.003228815360,
		-5.345602947917,
		-4.147172754881,
		-0.006294359154,
		10.691805533212,
		0.000609093939,
		0.002639715086,
		-5.346673358608,
		4.146462093362,

		0.001970232132,
		-3.171968084495,
		-3.579800654096,
		-0.000279054040,
		0.000609093939,
		7.158200782594,
		-0.001411983068,
		3.171778256731,
		-3.578418153998,

		-0.000082519062,
		0.000210528845,
		-0.000411609150,
		-0.000035643028,
		0.002639715086,
		-0.001411983068,
		0.000029954860,
		-0.002787139159,
		0.001872575891,

		0.000252427783,
		-0.485766171906,
		0.487471423749,
		0.003194478242,
		-5.346673358608,
		3.171778256731,
		-0.002787139159,
		5.832165997020,
		-3.658796612424,

		0.000206156134,
		-0.487720511535,
		0.181124721694,
		-0.002232267666,
		4.146462093362,
		-3.578418153998,
		0.001872575891,
		-3.658796612424,
		3.397216632634,
	}
	for i := range want {
		want[i] *= ANGBOHR * ANGBOHR / ATTO_JOULES
	}
	got := FloatsFromCountFloats(fc2)
	if _, _, ok := compfloat(got, want, 1e-12); !ok {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}
