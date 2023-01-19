package semver

import (
	"strings"

	"github.com/Masterminds/semver/v3"
)

// Constant enum as a return code from compare two semantic version
const (
	CompareVersionError = iota - 4
	SourceVersionInvalid
	SourceVersionEmpty
	SourceVersionGreat
	VersionsEqual
	TargetVersionGreat
	TargetVersionEmpty
	TargetVersionInvalid
)

// GetPureSemVer is function to get the most value data from semantic version string
// only major, minor and patch numbers
func GetPureSemVer(version string) string {
	min := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}
	worev := strings.Split(version, "-")[0]
	parts := strings.Split(worev, ".")
	wobnum := strings.Join(parts[:min(len(parts), 3)], ".")
	return strings.Trim(wobnum, " ")
}

// CompareVersions is function to check and compare two semantic versions
// comparing mechanism is using only major, minor and patch versions
// the function may return next values:
//
//	-4 means internal error in compare mechanism
//	-3 means that sourceVersion has invalid format
//	-2 means that sourceVersion is empty
//	-1 means that sourceVersion is greater than targetVersion
//	 0 means that two versions are equal
//	 1 means that targetVersion is greater than sourceVersion
//	 2 means that targetVersion is empty
//	 3 means that targetVersion has invalid format
func CompareVersions(sourceVersion, targetVersion string) int {
	targetPureVersion := GetPureSemVer(targetVersion)
	if targetPureVersion == "" {
		return TargetVersionEmpty
	}
	targetVersionSemver, err := semver.NewVersion(targetPureVersion)
	if err != nil {
		return TargetVersionInvalid
	}

	sourcePureVersion := GetPureSemVer(sourceVersion)
	if sourcePureVersion == "" {
		return SourceVersionEmpty
	}
	sourceVersionSemver, err := semver.NewVersion(sourcePureVersion)
	if err != nil {
		return SourceVersionInvalid
	}

	comparisonValue := targetVersionSemver.Compare(sourceVersionSemver)
	switch comparisonValue {
	case -1:
		return SourceVersionGreat
	case 0:
		return VersionsEqual
	case 1:
		return TargetVersionGreat
	default:
		return CompareVersionError
	}
}
