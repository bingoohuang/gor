package walk

//go:generate stringer -type=Location location.go

// Location ...
type Location uint

const (
	// None ...
	None Location = iota
	// Map ...
	Map
	// MapKey ...
	MapKey
	// MapValue ...
	MapValue
	// Slice ...
	Slice
	// SliceElem ...
	SliceElem
	// Array ...
	Array
	// ArrayElem ...
	ArrayElem
	// Struct ...
	Struct
	// StructField ...
	StructField
	// Loc ...
	Loc
)
