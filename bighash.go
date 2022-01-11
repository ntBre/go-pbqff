package main

// Status is a type describing the status of geometry in the
// BigHash. The three options are
// 1) the geometry has not been placed into the map
// 2) the geometry is in the map, but the calculation isn't finished
// 3) the geometry is in the map, and the calculation finished
type Status int

const (
	NotPresent Status = iota
	NotCalculated
	Done
)

type Energy struct {
	Status Status
	Value  float64
}

type BigHash map[string]*Energy

var Table BigHash = make(BigHash)

// somewhere I need to normalize the geometry for lookup, ie perform
// all the symmetry operations, should be in the initial lookup
// because we never check the geom again after that

// Lookup normalizes the geometry, performs the map lookup, and then
// returns the normalized geometry along with the current status and
// the value
func (bh BigHash) Lookup(geom string) (norm string, status Status, value float64) {
	norm = Normalize(geom)
	e, ok := bh[geom]
	if !ok {
		bh[geom] = &Energy{
			Status: NotCalculated,
		}
		return
	}
	return norm, e.Status, e.Value
}

func Normalize(geom string) string { return geom }

func (bh BigHash) At(geom string) *Energy {
	return bh[geom]
}
