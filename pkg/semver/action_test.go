package semver

import (
	"errors"
	"testing"
)

func TestBumpSemverVersion(t *testing.T) {
	tests := []struct {
		version   string
		increment string
		format    string
		expected  string
		err       error
	}{
		// Valid version increments
		{"1.2.3", "patch", "%major%.%minor%.%patch%", "1.2.4", nil},
		{"1.2.3", "minor", "%major%.%minor%.%patch%", "1.3.0", nil},
		{"1.2.3", "major", "%major%.%minor%.%patch%", "2.0.0", nil},

		// Valid version increments with different formats
		{"1.2.3", "patch", "v%major%.%minor%.%patch%", "v1.2.4", nil},
		{"1.2.3", "minor", "version-%major%.%minor%.%patch%", "version-1.3.0", nil},
		{"1.2.3", "major", "%major%-%minor%-%patch%", "2-0-0", nil},

		// Invalid increment
		{"1.2.3", "invalid", "%major%.%minor%.%patch%", "", ErrInvalidIncrement},

		// Invalid version format (generic error check)
		{"invalid", "patch", "%major%.%minor%.%patch%", "", errors.New("")}, // Any non-nil error

		// Edge case: zero version
		{"0.0.0", "patch", "%major%.%minor%.%patch%", "0.0.1", nil},
		{"0.0.0", "minor", "%major%.%minor%.%patch%", "0.1.0", nil},
		{"0.0.0", "major", "%major%.%minor%.%patch%", "1.0.0", nil},
	}

	for _, test := range tests {
		func(t *testing.T) {
			result, err := BumpSemverVersion(test.version, test.increment, test.format)
			if test.err != nil {
				// If we expect an error, check for non-nil error
				if err == nil {
					t.Errorf("BumpSemverVersion(%q, %q, %q) returned no error, expected error", test.version, test.increment, test.format)
				} else if test.err != ErrInvalidIncrement && err.Error() == "" {
					t.Errorf("BumpSemverVersion(%q, %q, %q) returned error %v, expected any non-nil error", test.version, test.increment, test.format, err)
				}
			} else {
				// If we do not expect an error, ensure result matches the expected value
				if err != nil {
					t.Errorf("BumpSemverVersion(%q, %q, %q) returned error %v, expected no error", test.version, test.increment, test.format, err)
				}
				if result != test.expected {
					t.Errorf("BumpSemverVersion(%q, %q, %q) = %q, expected %q", test.version, test.increment, test.format, result, test.expected)
				}
			}
		}(t)
	}

}
