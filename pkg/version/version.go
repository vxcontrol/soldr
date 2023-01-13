package version

import (
	"fmt"
)

// PackageVer is semantic version of vxapi
var PackageVer string

// PackageRev is revision of vxapi build
var PackageRev string

// IsDevelop defines developer mode
var IsDevelop string

func GetBinaryVersion() string {
	version := "develop"
	if PackageVer != "" {
		version = PackageVer
	}
	if PackageRev != "" {
		version = fmt.Sprintf("%s-%s", version, PackageRev)
	}
	return version
}
