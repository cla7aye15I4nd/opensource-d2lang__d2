package d2animate

import "bytes"

// Rule out incompatible artwork at the two ends of a span before decoding
// hundreds of intermediate SVGs. Every native track must survive both ends.
func mayHaveMotion(frames []Frame, svgs [][]byte, hash string, parsed []parsedMotionFrame) bool {
	last := len(frames) - 1
	first, end := &parsed[0], &parsed[last]
	if !first.load(frames[0], svgs[0], hash) || !end.load(frames[last], svgs[last], hash) {
		return false
	}
	if len(frames[0].Picture().Connections) > 0 && len(frames[last].Picture().Connections) > 0 {
		for _, frame := range frames {
			for i := range frame.Picture().Shapes {
				if frame.Picture().Shapes[i].MotionOffset != nil {
					return true // A straight wire may follow this pin.
				}
			}
		}
	}
	scans := 0
	for _, child := range first.children {
		id := child.id
		k, common := end.positions[id]
		if !common || k < 0 || first.positions[id] < 0 || first.shapes[id] == nil || end.shapes[id] == nil {
			continue
		}
		raw := svgs[0][child.start:child.end]
		_, signature, ok := readMotionShape(raw, first.shapes[id])
		if !ok {
			continue
		}
		other := end.children[k]
		final := svgs[last][other.start:other.end]
		if !bytes.Equal(raw, final) {
			if _, sig, ok := readMotionShape(final, end.shapes[id]); ok && sig == signature {
				return true
			}
			continue
		}
		// An object may return to its starting pose. Only rule it out when
		// every intermediate frame contains its exact, uniquely named drawing.
		// Bound full-frame scans on crowded diagrams; the normal index is linear.
		if scans == 4 {
			return true
		}
		scans++
		key := []byte(id)
		for _, frame := range svgs[1:last] {
			if !bytes.Contains(frame, raw) || bytes.Count(frame, key) != 1 || bytes.Contains(frame, []byte("&#")) {
				return true // Numeric XML entities could spell another copy of this ID.
			}
		}
	}
	return false
}
