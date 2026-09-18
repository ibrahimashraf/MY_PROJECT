package galloc

// Dummy implementation of galloc for lean-ctx build

func AllocateAligned(size, alignment uint64) []byte {
	// naive non-cgo allocation
	buf := make([]byte, size+alignment)
	// mock returning the slice
	return buf
}

func Free(b []byte) {
	// no-op in go
}
