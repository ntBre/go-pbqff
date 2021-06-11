package main

import "testing"

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
		a.WriteAnpass(test.write, []float64{0, 0, 0, 0, 0, 0})
		if !compareFile(test.write, test.right) {
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
		a.WriteAnpass2(test.write, ll, []float64{0, 0, 0, 0, 0, 0})
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

func TestFromIntder(t *testing.T) {
	got := FromIntder("testfiles/lin.intder")
	want := "FIXME"
	if got != want {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}
