package util

import "strconv"

// ParseUInt64 returns an UInt64 from a string
func ParseUInt64(input string) (uint64, error) {
	return strconv.ParseUint(input, 10, 64)
}

// MustUInt64 returns an UInt64 value without an error
func MustUInt64(input string) uint64 {
	v, err := ParseUInt64(input)
	if err != nil {
		v = 0
	}
	return v
}
