package main

import (
	"bytes"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ntBre/chemutils/spectro"
	"github.com/ntBre/chemutils/summarize"
	symm "github.com/ntBre/chemutils/symmetry"
)

func TestSIC(t *testing.T) {
	if !testing.Short() {
		t.Skip()
	}
	*test = true
	qsub = "qsub/qsub"
	temp := Conf
	Conf = NewConfig()
	defer func() {
		flags = 0
		Conf = temp
		*test = false
		qsub = "qsub"
		submitted = 0
		cenergies = *new([]CountFloat)
	}()
	prog, intder, anpass := initialize("tests/sic/sic.in")
	names := strings.Fields("H O H")
	intder.WritePts("tests/sic/pts/intder.in")
	RunIntder("tests/sic/pts/intder")
	queue := PBS{SinglePt: pbsMaple, ChunkPts: ptsMaple}
	gen := BuildPoints(prog, queue, "tests/sic/pts/file07", names, true)
	E0 := -76.369839620287
	min, _ := Drain(prog, queue, 0, E0, gen)
	energies := FloatsFromCountFloats(cenergies)
	for i := range energies {
		energies[i] -= min
	}
	fmt.Println(filepath.Join(prog.GetDir(), "freqs"))
	longLine, _ := DoAnpass(anpass, filepath.Join(prog.GetDir(), "freqs"), energies, nil)
	coords, _ := DoIntder(intder, names, longLine, prog.GetDir(), false)
	spec, err := spectro.Load("tests/sic/spectro.in")
	if err != nil {
		errExit(err, "loading spectro input")
	}
	spec.FormatGeom(names, coords)
	spec.WriteInput("tests/sic/freqs/spectro.in")
	err = spec.DoSpectro("tests/sic/freqs/")
	if err != nil {
		errExit(err, "running spectro")
	}
	res := summarize.SpectroFile(
		filepath.Join("tests/sic/", "freqs", "spectro2.out"))
	want := []float64{3753.2, 3656.5, 1598.8}
	if !compfloat(res.Corr, want, 1e-1) {
		t.Errorf("got %v, wanted %v\n", res.Corr, want)
	}
}

func compfloat(a, b []float64, eps float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(a[i]-b[i]) > eps {
			return false
		}
	}
	return true
}

func TestCart(t *testing.T) {
	if !testing.Short() {
		t.Skip()
	}
	*test = true
	qsub = "qsub/qsub"
	temp := Conf
	tmpsym := *nosym
	defer func() {
		flags = 0
		Conf = temp
		*test = false
		qsub = "qsub"
		submitted = 0
		*nosym = tmpsym
	}()
	tests := []struct {
		name   string
		infile string
		want   []float64
		harm   []float64
		nosym  bool
	}{
		{
			name:   "h2o",
			infile: "tests/cart/h2o/cart.in",
			want:   []float64{3753.2, 3656.5, 1598.5},
			harm:   []float64{3943.690, 3833.702, 1650.933},
			nosym:  false,
		},
		{
			name:   "h2co",
			infile: "tests/cart/h2co/test.in",
			want: []float64{
				2826.6, 2778.4, 1747.8,
				1499.4, 1246.8, 1167.0,
			},
			harm: []float64{
				3004.590, 2932.596, 1778.656,
				1534.098, 1269.765, 1186.913,
			},
			nosym: false,
		},
		{
			name:   "nh3",
			infile: "tests/cart/nh3/test.in",
			want: []float64{
				3435.8, 3435.7, 3341.7,
				1628.3, 1628.0, 979.6,
			},
			harm: []float64{
				3610.420, 3610.299, 3478.498,
				1675.554, 1675.300, 1056.025,
			},
			nosym: true,
		},
	}
	for _, test := range tests {
		*nosym = test.nosym
		Conf = NewConfig()
		submitted = 0
		prog, _, _ := initialize(test.infile)
		prog.FormatCart(Conf.Str(Geometry))
		cart := prog.GetGeom()
		queue := PBS{SinglePt: pbsMaple, ChunkPts: ptsMaple}
		E0 := prog.Run(none, queue)
		names, coords := XYZGeom(cart)
		natoms := len(names)
		other3, other4 := initArrays(natoms)
		ncoords := len(coords)
		mol := symm.ReadXYZ(strings.NewReader(cart))
		gen := BuildCartPoints(prog, queue, "pts/inp", names, coords, mol)
		Drain(prog, queue, ncoords, E0, gen)
		PrintFortFile(fc2, natoms, 6*natoms, filepath.Join(prog.GetDir(), "fort.15"))
		PrintFortFile(fc3, natoms, other3, filepath.Join(prog.GetDir(), "fort.30"))
		PrintFortFile(fc4, natoms, other4, filepath.Join(prog.GetDir(), "fort.40"))
		var buf bytes.Buffer
		for i := range coords {
			if i%3 == 0 && i > 0 {
				fmt.Fprint(&buf, "\n")
			}
			fmt.Fprintf(&buf, " %.10f", coords[i]/angbohr)
		}
		specin := filepath.Join(prog.GetDir(), "spectro.in")
		spec, err := spectro.Load(specin)
		if err != nil {
			errExit(err, "loading spectro input")
		}
		spec.FormatGeom(names, buf.String())
		spec.WriteInput(specin)
		err = spec.DoSpectro(prog.GetDir())
		if err != nil {
			errExit(err, "running spectro")
		}
		res := summarize.SpectroFile(filepath.Join(prog.GetDir(), "spectro2.out"))
		if !compfloat(res.Harm, test.harm, 1e-1) {
			t.Errorf("harm: got %v, wanted %v\n", res.Harm, test.harm)
		}
		// TODO also test rots for cubic fc accuracy
		if !compfloat(res.Corr, test.want, 1e-1) {
			t.Errorf("fund: got %v, wanted %v\n", res.Corr, test.want)
		}
	}
}

func TestGrad(t *testing.T) {
	if !testing.Short() {
		t.Skip()
	}
	*test = true
	qsub = "qsub/qsub"
	temp := Conf
	Conf = NewConfig()
	defer func() {
		Conf = temp
		*test = false
		qsub = "qsub"
		submitted = 0
		flags = 0
	}()
	prog, _, _ := initialize("tests/grad/grad.in")
	prog.FormatCart(Conf.Str(Geometry))
	cart := prog.GetGeom()
	E0 := 0.0
	names, coords := XYZGeom(cart)
	natoms := len(names)
	other3, other4 := initArrays(natoms)
	ncoords := len(coords)
	queue := PBS{SinglePt: pbsMaple, ChunkPts: ptsMaple}
	mol := symm.ReadXYZ(strings.NewReader(cart))
	gen := BuildGradPoints(prog, queue, "pts/inp", names, coords, mol)
	Drain(prog, queue, ncoords, E0, gen)
	PrintFortFile(fc2, natoms, 6*natoms, filepath.Join(prog.GetDir(), "fort.15"))
	PrintFortFile(fc3, natoms, other3, filepath.Join(prog.GetDir(), "fort.30"))
	PrintFortFile(fc4, natoms, other4, filepath.Join(prog.GetDir(), "fort.40"))
	var buf bytes.Buffer
	for i := range coords {
		if i%3 == 0 && i > 0 {
			fmt.Fprint(&buf, "\n")
		}
		fmt.Fprintf(&buf, " %.10f", coords[i]/angbohr)
	}
	specin := filepath.Join(prog.GetDir(), "spectro.in")
	spec, err := spectro.Load(specin)
	if err != nil {
		errExit(err, "loading spectro input")
	}
	spec.FormatGeom(names, buf.String())
	spec.WriteInput(specin)
	err = spec.DoSpectro(prog.GetDir())
	if err != nil {
		errExit(err, "running spectro in test")
	}
	res := summarize.SpectroFile(filepath.Join(prog.GetDir(), "spectro2.out"))
	want := []float64{3739.1, 3651.1, 1579.4}
	if !compfloat(res.Corr, want, 1e-1) {
		t.Errorf("got %v, wanted %v\n", res.Corr, want)
	}
}
