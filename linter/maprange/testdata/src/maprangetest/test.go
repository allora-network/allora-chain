package maprangetest

func testMapRange() {
	// This should trigger the linter
	m := map[string]int{"a": 1, "b": 2}
	for k, v := range m { // want "range over map detected, which can be non-deterministic"
		_ = k
		_ = v
	}
}

func testNoMapRange() {
	s := []string{"a", "b", "c"}
	for i, v := range s {
		_ = i
		_ = v
	}
}

func testEmptyMapRange() {
	m := map[string]int{}
	for k, v := range m { // want "range over map detected, which can be non-deterministic"
		_ = k
		_ = v
	}
}
