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

// ParseInt64 returns an Int64 from a string
func ParseInt64(input string) (int64, error) {
	return strconv.ParseInt(input, 10, 64)
}

// MustInt64 returns an Int64 value without an error
func MustInt64(input string) int64 {
	v, err := ParseInt64(input)
	if err != nil {
		v = 0
	}
	return v
}
