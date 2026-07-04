package database

// KeyName converts an Engine DJ numeric key value to its musical key name
// (e.g. 13 → "Am", 4 → "C"). Returns "" for unknown/unset values.
func KeyName(key int) string {
	if key < 0 || key >= len(keyNames) {
		return ""
	}
	return keyNames[key]
}

// KeyCamelot converts an Engine DJ numeric key value to Camelot notation
// (e.g. 13 → "8A", 4 → "8B"). Returns "" for unknown/unset values.
func KeyCamelot(key int) string {
	if key < 0 || key >= len(camelotKeys) {
		return ""
	}
	return camelotKeys[key]
}

// keyNames maps Engine DJ key integer → musical key name.
// Index 0 is "not set". Indices 1–12 are major keys, 13–24 are minor keys.
var keyNames = []string{
	"",    // 0 – not set
	"A",   // 1
	"B♭",  // 2
	"B",   // 3
	"C",   // 4
	"D♭",  // 5
	"D",   // 6
	"E♭",  // 7
	"E",   // 8
	"F",   // 9
	"G♭",  // 10
	"G",   // 11
	"A♭",  // 12
	"Am",  // 13
	"B♭m", // 14
	"Bm",  // 15
	"Cm",  // 16
	"D♭m", // 17
	"Dm",  // 18
	"E♭m", // 19
	"Em",  // 20
	"Fm",  // 21
	"G♭m", // 22
	"Gm",  // 23
	"A♭m", // 24
}

// camelotKeys maps Engine DJ key integer → Camelot notation.
var camelotKeys = []string{
	"",    // 0 – not set
	"11B", // 1  A
	"6B",  // 2  Bb
	"1B",  // 3  B
	"8B",  // 4  C
	"3B",  // 5  Db
	"10B", // 6  D
	"5B",  // 7  Eb
	"12B", // 8  E
	"7B",  // 9  F
	"2B",  // 10 Gb
	"9B",  // 11 G
	"4B",  // 12 Ab
	"8A",  // 13 Am
	"3A",  // 14 Bbm
	"10A", // 15 Bm
	"5A",  // 16 Cm
	"12A", // 17 Dbm
	"7A",  // 18 Dm
	"2A",  // 19 Ebm
	"9A",  // 20 Em
	"4A",  // 21 Fm
	"11A", // 22 Gbm
	"6A",  // 23 Gm
	"1A",  // 24 Abm
}
