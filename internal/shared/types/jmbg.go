package types

import (
	"fmt"
	"regexp"
)

// JMBG represents a Serbian personal identification number (13 digits)
// Format: DDMMYYYRRBBBK where:
// - DDMMYYY: date of birth (DD day, MM month, YYY year with region prefix)
// - RR: region code
// - BBB: unique number
// - K: checksum digit
type JMBG string

var jmbgRegex = regexp.MustCompile(`^\d{13}$`)

// ParseJMBG validates and parses a JMBG string
func ParseJMBG(s string) (JMBG, error) {
	if !jmbgRegex.MatchString(s) {
		return "", fmt.Errorf("JMBG must be exactly 13 digits")
	}

	jmbg := JMBG(s)
	if !jmbg.IsValid() {
		return "", fmt.Errorf("invalid JMBG checksum")
	}

	return jmbg, nil
}

// String returns the string representation
func (j JMBG) String() string {
	return string(j)
}

// Masked returns a masked version for display (first 7 digits visible)
func (j JMBG) Masked() string {
	if len(j) < 13 {
		return "***********"
	}
	return string(j)[:7] + "******"
}

// IsValid validates the JMBG checksum
func (j JMBG) IsValid() bool {
	if len(j) != 13 {
		return false
	}

	// Checksum calculation using Mod 11 algorithm
	digits := make([]int, 13)
	for i, c := range j {
		digits[i] = int(c - '0')
	}

	// Weights for JMBG validation
	weights := []int{7, 6, 5, 4, 3, 2, 7, 6, 5, 4, 3, 2}

	sum := 0
	for i := 0; i < 12; i++ {
		sum += digits[i] * weights[i]
	}

	remainder := sum % 11
	checkDigit := 0
	if remainder != 0 {
		checkDigit = 11 - remainder
	}

	// If checkDigit is 10, JMBG is invalid
	if checkDigit == 10 {
		return false
	}

	return digits[12] == checkDigit
}

// IsZero checks if the JMBG is empty
func (j JMBG) IsZero() bool {
	return j == ""
}
