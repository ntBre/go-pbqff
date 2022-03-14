package main

import (
	"bytes"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ntBre/anpass"
	"github.com/ntBre/chemutils/spectro"
	"github.com/ntBre/chemutils/summarize"
	symm "github.com/ntBre/chemutils/symmetry"
	"gonum.org/v1/gonum/mat"
)

func TestSIC(t *testing.T) {
	if !testing.Short() {
		t.Skip()
	}
	*test = true
	qsub = "qsub/qsub"
	temp := Conf
	SIC = true
	*overwrite = true
	defer func() {
		SIC = false
		Conf = temp
		*test = false
		qsub = "qsub"
		Global.Submitted = 0
		cenergies = *new([]CountFloat)
	}()
	Conf = ParseInfile("tests/sic/sic.in").ToConfig()
	prog, intder, anpass := initialize("tests/sic/sic.in")
	names := strings.Fields("H O H")
	intder.WritePts("tests/sic/pts/intder.in")
	RunIntder("tests/sic/pts/intder")
	queue := &PBS{Tmpl: MolproPBSTmpl}
	gen := BuildPoints(prog, queue, "tests/sic/pts/file07", names, true)
	E0 := -76.369839620287
	min, _ := Drain(prog, queue, 0, E0, gen)
	energies := FloatsFromCountFloats(cenergies)
	for i := range energies {
		energies[i] -= min
	}
	longLine, _ := DoAnpass(
		anpass, filepath.Join(prog.GetDir(), "freqs"), energies, nil,
	)
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
	if _, _, ok := compfloat(res.Corr, want, 1e-1); !ok {
		t.Errorf("got %v, wanted %v\n", res.Corr, want)
	}
}

// compfloat determines whether a and b are the same to eps precision. If they
// are different, the index in which they differ and the difference between the
// values at that index are returned along with false
func compfloat(a, b []float64, eps float64) (int, float64, bool) {
	if len(a) != len(b) {
		return 0, 0, false
	}
	for i := range a {
		diff := a[i] - b[i]
		if math.Abs(diff) > eps {
			return i, diff, false
		}
	}
	return 0, 0, true
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
		Conf = temp
		*test = false
		qsub = "qsub"
		Global.Submitted = 0
		*nosym = tmpsym
	}()
	tests := []struct {
		name   string
		infile string
		want   []float64
		harm   []float64
		rots   []float64 // vib. avg. rots
		nosym  bool
	}{
		{
			name:   "h2o",
			infile: "tests/cart/h2o/cart.in",
			harm: []float64{
				3943.55, 3833.553, 1651.393,
			},
			rots: []float64{
				14.5044924, 9.2631640, 27.6550391,
			},
			want: []float64{
				3752.9223, 3656.333, 1599.016,
			},
			nosym: false,
		},
		{
			name:   "h2co",
			infile: "tests/cart/h2co/test.in",
			harm: []float64{
				3004.467, 2932.477, 1778.454,
				1534.213, 1269.931, 1186.872,
			},
			rots: []float64{
				1.2914872, 1.1310027, 9.3988073,
			},
			want: []float64{
				2826.6802, 2778.4171, 1747.6452,
				1499.7355, 1247.435, 1167.0952,
			},
			nosym: false,
		},
		{
			name:   "nh3",
			infile: "tests/cart/nh3/test.in",
			harm: []float64{
				3610.241, 3610.202, 3478.359,
				1675.589, 1675.525, 1056.259,
			},
			rots: []float64{
				9.8902307, 6.2261434, 9.889629,
			},
			want: []float64{
				3435.7065, 3435.6384, 3342.0345,
				1628.6497, 1628.6723, 980.7472,
			},
			nosym: false,
		},
	}
	*overwrite = true
	for _, test := range tests[0:] {
		*nosym = test.nosym
		Conf = ParseInfile(test.infile).ToConfig()
		Global.Submitted = 0
		prog, _, _ := initialize(test.infile)
		prog.FormatCart(Conf.Geometry)
		cart := prog.GetGeom()
		queue := &PBS{Tmpl: MolproPBSTmpl}
		names, coords := XYZGeom(cart)
		natoms := len(names)
		other3, other4 := initArrays(natoms)
		cenergies = *new([]CountFloat)
		ncoords := len(coords)
		mol := symm.ReadXYZ(strings.NewReader(cart))
		gen, forces := BuildCartPoints(
			prog, queue, "pts/inp", names, coords, mol,
		)
		min, _ := Drain(prog, queue, ncoords, 0, gen)
		energies := FloatsFromCountFloats(cenergies)
		for i := range energies {
			energies[i] -= min
		}
		nforces := make([]float64, 0)
		for i := 0; i < len(forces[0]); i++ {
			for j := 0; j < len(forces); j++ {
				nforces = append(nforces, float64(forces[j][i]))
			}
		}
		exps := mat.NewDense(len(forces[0]), len(forces), nforces)
		steps := DispToStep(Disps(forces, false))
		stepdat := make([]float64, 0)
		for _, step := range steps {
			stepdat = append(stepdat,
				Step(make([]float64, ncoords), step...)...)
		}
		disps := mat.NewDense(len(stepdat)/ncoords, ncoords, stepdat)
		coeffs, _ := anpass.Fit(disps, energies, exps)
		fcs := anpass.MakeFCs(coeffs, exps)
		Format9903(ncoords, fcs)
		PrintFortFile(fc2, natoms, 6*natoms,
			filepath.Join(prog.GetDir(), "fort.15"))
		PrintFortFile(fc3, natoms, other3, filepath.Join(prog.GetDir(),
			"fort.30"))
		PrintFortFile(fc4, natoms, other4, filepath.Join(prog.GetDir(),
			"fort.40"))
		var buf bytes.Buffer
		for i := range coords {
			if i%3 == 0 && i > 0 {
				fmt.Fprint(&buf, "\n")
			}
			fmt.Fprintf(&buf, " %.10f", coords[i]/ANGBOHR)
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
		if i, v, ok := compfloat(res.Harm, test.harm, 1e-1); !ok {
			t.Errorf("%s harm: got\n%v, wanted\n%v\n"+
				"%dth element differs by %f\n",
				test.name, res.Harm, test.harm,
				i, v)
		}
		if i, v, ok := compfloat(res.Rots[0], test.rots, 1e-5); !ok {
			t.Errorf("%s rots: got\n%v, wanted\n%v\n"+
				"%dth element differs by %f\n",
				test.name, res.Rots[0], test.rots,
				i, v)
		}
		if i, v, ok := compfloat(res.Corr, test.want, 1e-1); !ok {
			t.Errorf("%s fund: got\n%v, wanted\n%v\n"+
				"%dth element differs by %f\n",
				test.name, res.Corr, test.want,
				i, v)
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
	Conf = ParseInfile("tests/grad/grad.in").ToConfig()
	defer func() {
		Conf = temp
		*test = false
		qsub = "qsub"
		Global.Submitted = 0
		GRAD = false
	}()
	SIC = false
	GRAD = true
	prog, _, _ := initialize("tests/grad/grad.in")
	prog.FormatCart(Conf.Geometry)
	cart := prog.GetGeom()
	E0 := 0.0
	names, coords := XYZGeom(cart)
	natoms := len(names)
	other3, other4 := initArrays(natoms)
	ncoords := len(coords)
	queue := &PBS{Tmpl: MolproPBSTmpl}
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
		fmt.Fprintf(&buf, " %.10f", coords[i]/ANGBOHR)
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
	if _, _, ok := compfloat(res.Corr, want, 1e-1); !ok {
		t.Errorf("got %v, wanted %v\n", res.Corr, want)
	}
}

// TestResub is a copy of TestCart, restricted back to the h2o test
func TestResub(t *testing.T) {
	if !testing.Short() {
		t.Skip()
	}
	*test = true
	qsub = "qsub/qsub"
	temp := Conf
	tmpsym := *nosym
	tmpchk := *checkpoint
	defer func() {
		Conf = temp
		*test = false
		qsub = "qsub"
		Global.Submitted = 0
		*nosym = tmpsym
		*checkpoint = tmpchk
	}()
	tests := []struct {
		name   string
		infile string
		want   []float64
		harm   []float64
		rots   []float64 // vib. avg. rots
		nosym  bool
	}{
		{
			name:   "h2o",
			infile: "tests/cart/h2o/cart.in",
			want:   []float64{3753.2, 3656.5, 1598.5},
			harm:   []float64{3943.690, 3833.702, 1650.933},
			rots:   []float64{14.50450, 9.26320, 27.65578},
			nosym:  false,
		},
	}
	*overwrite = true
	for _, test := range tests {
		*nosym = test.nosym
		Conf = ParseInfile(test.infile).ToConfig()
		Global.Submitted = 0
		prog, _, _ := initialize(test.infile)
		prog.FormatCart(Conf.Geometry)
		cart := prog.GetGeom()
		queue := &PBS{Tmpl: MolproPBSTmpl}
		E0 := prog.Run(none, queue)
		names, coords := XYZGeom(cart)
		natoms := len(names)
		initArrays(natoms)
		cenergies = *new([]CountFloat)
		ncoords := len(coords)
		mol := symm.ReadXYZ(strings.NewReader(cart))
		basegen, _ := BuildCartPoints(
			prog, queue, "pts/inp", names, coords, mol)
		counter := 24
		gen := func() ([]Calc, bool) {
			counter--
			if counter == 0 {
				panic("counter hit 0")
			}
			return basegen()
		}
		// TODO catch this panic and resubmit - reset
		// everything above here and call Drain again
		defer func() {
			if r := recover(); r != nil {
				if r != "counter hit 0" {
					panic("wrong panic caught")
				}
				fmt.Printf("caught the panic %q, resubmitting\n", r)
				*checkpoint = true
				*nosym = test.nosym
				Conf = ParseInfile(test.infile).ToConfig()
				Global.Submitted = 0
				prog, _, _ := initialize(test.infile)
				prog.FormatCart(Conf.Geometry)
				cart := prog.GetGeom()
				queue := &PBS{Tmpl: MolproPBSTmpl}
				E0 := prog.Run(none, queue)
				names, coords := XYZGeom(cart)
				natoms := len(names)
				other3, other4 := initArrays(natoms)
				ncoords := len(coords)
				mol := symm.ReadXYZ(strings.NewReader(cart))
				gen, _ := BuildCartPoints(
					prog, queue, "pts/inp",
					names, coords, mol,
				)
				Drain(prog, queue, ncoords, E0, gen)
				PrintFortFile(fc2, natoms, 6*natoms, filepath.Join(prog.GetDir(), "fort.15"))
				PrintFortFile(fc3, natoms, other3, filepath.Join(prog.GetDir(), "fort.30"))
				PrintFortFile(fc4, natoms, other4, filepath.Join(prog.GetDir(), "fort.40"))
				var buf bytes.Buffer
				for i := range coords {
					if i%3 == 0 && i > 0 {
						fmt.Fprint(&buf, "\n")
					}
					fmt.Fprintf(&buf, " %.10f", coords[i]/ANGBOHR)
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
				if i, v, ok := compfloat(res.Harm, test.harm, 1e-1); !ok {
					t.Errorf("%s harm: got\n%v, wanted\n%v\n"+
						"%dth element differs by %f\n",
						test.name, res.Harm, test.harm,
						i, v)
				}
				if i, v, ok := compfloat(res.Rots[0], test.rots, 1e-5); !ok {
					t.Errorf("%s rots: got\n%v, wanted\n%v\n"+
						"%dth element differs by %f\n",
						test.name, res.Rots[0], test.rots,
						i, v)
				}
				if i, v, ok := compfloat(res.Corr, test.want, 1e-1); !ok {
					t.Errorf("%s fund: got\n%v, wanted\n%v\n"+
						"%dth element differs by %f\n",
						test.name, res.Corr, test.want,
						i, v)
				}
			}
		}()
		Drain(prog, queue, ncoords, E0, gen)
	}
}
