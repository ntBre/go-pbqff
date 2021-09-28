package main

import (
	"os"
)

// Calc holds the name of a job to be run and its result's index in
// the output array
type Calc struct {
	Name     string
	Targets  []Target
	Result   float64
	JobID    string
	ResubID  string
	noRun    bool
	SubFile  string
	ChunkNum int
	Resub    *Calc
	Src      *Source
	Scale    float64
}

// CountFloat combines a value with a counter that keeps track of how
// many times it has been modified, and a boolean Loaded to see if it
// was loaded from a checkpoint file
type CountFloat struct {
	Val    float64
	Count  int
	Loaded bool
}

// Add modifies the underlying value of c and decrements its counter
func (c *CountFloat) Add(t Target, scale float64, plus float64) {
	c.Val += plus
	c.Count--
	if c.Count < 0 {
		panic("added to CountFloat too many times")
	} else if c.Count == 0 && t.Slice != &e2d {
		c.Val *= scale
	}
}

// Done reports whether or not c's count has reached zero
func (c *CountFloat) Done() bool { return c.Count == 0 }

// FloatsFromCountFloats converts a slice of CountFloats to the
// corresponding Float64s
func FloatsFromCountFloats(cfs []CountFloat) (floats []float64) {
	for _, cf := range cfs {
		floats = append(floats, cf.Val)
	}
	return
}

// A Source is CountFloat slice and an index in that slice
type Source struct {
	Slice *[]CountFloat
	Index int
}

// Len returns the length of s's underlying slice
func (s *Source) Len() int { return len(*s.Slice) }

// Value returns s's underlying value
func (s *Source) Value() float64 {
	return (*s.Slice)[s.Index].Val
}

// Target combines a coefficient, target array, and the index into
// that array
type Target struct {
	Coeff float64
	Slice *[]CountFloat
	Index int
}

// GarbageHeap is a slice of Basenames to be deleted
type GarbageHeap struct {
	heap []string // list of basenames
}

// Add a filename to the heap
func (g *GarbageHeap) Add(basename string) {
	g.heap = append(g.heap, basename+".inp", basename+".out")
}

// Len returns the length of g's underlying slice
func (g *GarbageHeap) Len() int {
	return len(g.heap)
}

// Dump deletes the globbed files in the heap using an appended *
func (g *GarbageHeap) Dump() {
	for _, f := range g.heap {
		os.Remove(f)
	}
	g.heap = []string{}
}

// Program is an interface for using different quantum chemical
// programs in the place of Molpro. TODO this is a massive interface,
// how many of these are really necessary?
type Program interface {
	WriteInput(string, Procedure)
	FormatZmat(string) error
	SetDir(string)
	GetDir() string
	Run(Procedure) float64
	HandleOutput(string) (string, string, error)
	UpdateZmat(string)
	FormatCart(string) error
	GetGeometry() string
	BuildPoints(string, []string,
		*[]CountFloat, bool) func() ([]Calc, bool)
	BuildCartPoints(string, []string, []float64) func() ([]Calc, bool)
	BuildGradPoints(string, []string, []float64) func() ([]Calc, bool)
	ReadOut(string) (float64, float64, []float64, error)
	ReadFreqs(string) []float64
}
