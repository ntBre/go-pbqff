package main

import (
	"fmt"
	"os"
)

// Calc holds the name of a job to be run and its result's index in
// the output array
type Calc struct {
	Resub    *Calc
	Src      *Energy
	Name     string
	SubFile  string
	ResubID  string
	JobID    string
	Targets  []Target
	Coords   []float64
	ChunkNum int
	Result   float64
	Scale    float64
	noRun    bool
}

func (c Calc) String() string {
	return fmt.Sprintf(`{
  Name:    %s,
  Coords:  %f,
  Targets: %v,
  Result:  %f,
  JobID:   %s,
  ResubID: %s,
  noRun:   %v,
  SubFile: %s,
  ChunkNum: %d,
  Resub:    %p,
  Src:      %p,
  Scale:    %f,
}
`, c.Name, c.Coords, c.Targets, c.Result, c.JobID, c.ResubID,
		c.noRun, c.SubFile, c.ChunkNum, c.Resub, c.Src, c.Scale,
	)
}

// ProtoCalc is a precursor to a Calc with information for setting up
// the Calc itself
type ProtoCalc struct {
	Name  string
	Steps []int
	Index []int
	Coeff float64
	Scale float64
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
		fmt.Fprintf(os.Stderr, "fc2: %p\n", &fc2)
		fmt.Fprintf(os.Stderr, "fc3: %p\n", &fc3)
		fmt.Fprintf(os.Stderr, "fc4: %p\n", &fc4)
		fmt.Fprintf(os.Stderr, "too many additions to %p\n", t.Slice)
		panic("added to CountFloat too many times")
	} else if c.Count == 0 {
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
