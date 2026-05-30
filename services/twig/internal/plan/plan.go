package plan

// Overlap reports whether two half-open intervals [aStart, aStart+aDur) and
// [bStart, bStart+bDur) overlap. Touching at the boundary is NOT an overlap.
func Overlap(aStart, aDur, bStart, bDur int) bool {
	return aStart < bStart+bDur && bStart < aStart+aDur
}
