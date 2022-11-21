package mmodule

import (
	"fmt"
	"testing"
)

func Test_normalizeSemver(t *testing.T) {
	type TestCase struct {
		inVersion          string
		expectedOutVersion string
	}

	// Test cases group 1
	alreadyNormalizedTestCases := []TestCase{
		{
			inVersion:          "",
			expectedOutVersion: "",
		},
		{
			inVersion:          "1",
			expectedOutVersion: "1",
		},
		{
			inVersion:          "1.2",
			expectedOutVersion: "1.2",
		},
	}
	const (
		optionalPrefix = "v"
		optionalDot    = "."
	)
	for _, v := range alreadyNormalizedTestCases {
		alreadyNormalizedTestCases = append(
			alreadyNormalizedTestCases,
			TestCase{
				inVersion:          optionalPrefix + v.inVersion,
				expectedOutVersion: optionalPrefix + v.expectedOutVersion,
			},
			TestCase{
				inVersion:          v.inVersion + optionalDot,
				expectedOutVersion: v.expectedOutVersion + optionalDot,
			},
			TestCase{
				inVersion:          optionalPrefix + v.inVersion + optionalDot,
				expectedOutVersion: optionalPrefix + v.expectedOutVersion + optionalDot,
			},
		)
	}

	// Test cases group 2
	toNormalizeTestCases := []TestCase{
		// This test case is not to be normalized, but if we add a dot at the end of the inVersion field
		// normalizeSemver removes it, so this test case is placed in the second test group
		{
			inVersion:          "1.2.3",
			expectedOutVersion: "1.2.3",
		},
		{
			inVersion:          "1.2.3.",
			expectedOutVersion: "1.2.3",
		},
		{
			inVersion:          "1.2.3.4",
			expectedOutVersion: "1.2.3",
		},
		{
			inVersion:          "1.2.3-some.radom.prerelease",
			expectedOutVersion: "1.2.3",
		},
		{
			inVersion:          "1.2.3.4-some.random.prerelease",
			expectedOutVersion: "1.2.3",
		},
		{
			inVersion:          "1.2.3 - some.random.prerelease",
			expectedOutVersion: "1.2.3 ",
		},
		{
			inVersion:          "1.2.3.4.5",
			expectedOutVersion: "1.2.3",
		},
	}
	for _, v := range toNormalizeTestCases {
		toNormalizeTestCases = append(
			toNormalizeTestCases,
			TestCase{
				inVersion:          optionalPrefix + v.inVersion,
				expectedOutVersion: optionalPrefix + v.expectedOutVersion,
			},
		)
	}

	// Test run
	areVersionsEqual := func(actual string, expected string) error {
		if actual != expected {
			return fmt.Errorf("expected \"%s\", got \"%s\"", expected, actual)
		}
		return nil
	}

	for i, tc := range alreadyNormalizedTestCases {
		t.Run(fmt.Sprintf("already normalized test cases: test case #%d", i), func(t *testing.T) {
			actualOutVersion := normalizeSemver(tc.inVersion)
			if err := areVersionsEqual(actualOutVersion, tc.expectedOutVersion); err != nil {
				t.Error(err)
				return
			}
		})
	}

	for i, tc := range toNormalizeTestCases {
		t.Run(fmt.Sprintf("test cases to normalize: test case #%d", i), func(t *testing.T) {
			actualOutVersion := normalizeSemver(tc.inVersion)
			if err := areVersionsEqual(actualOutVersion, tc.expectedOutVersion); err != nil {
				t.Error(err)
				return
			}
		})
	}
}
