package asconfig

import (
	"fmt"

	lib "github.com/aerospike/aerospike-management-lib"
)

// Metadata for restriction check

// unsupportedJumps is set of versions which need special attention or system changes for upgrade/downgrade.
// upgrade/downgrade from A to B is not supported if (A < R and B > R) or (A > R and B < R) for any unsupported
// jump version R. This list should be in ascending order.
var unsupportedJumps = []string{"3.13", "4.2", "4.3", "4.9"}

func setUnsupportedJumps() error {
	// init unsupportedJumps
	return nil
}

// IsValidUpgrade validates fromVersion and toVersion for
// all upgrade/downgrade restrictions
func IsValidUpgrade(fromVersion, toVersion string) error {
	// check version validity
	valid, err := IsSupportedVersion(fromVersion)
	if !valid || err != nil {
		return fmt.Errorf("unsupported aerospike version %s", fromVersion)
	}

	valid, err = IsSupportedVersion(toVersion)
	if !valid || err != nil {
		return fmt.Errorf("unsupported aerospike version %s", toVersion)
	}

	fromBaseVersion, err := BaseVersion(fromVersion)
	if err != nil {
		return err
	}

	toBaseVersion, err := BaseVersion(toVersion)
	if err != nil {
		return err
	}

	return checkUpgradeJump(fromBaseVersion, toBaseVersion)
}

// IsUpgrade returns true if it is upgrade else false
func IsUpgrade(fromVersion, toVersion string) (bool, error) {
	r, err := lib.CompareVersions(fromVersion, toVersion)
	if err != nil {
		return false, fmt.Errorf("failed to compare version fromVersion %s , toVersion %s: %v", fromVersion, toVersion, err)
	}

	if r >= 0 {
		return false, nil
	}

	return true, nil
}

func checkUpgradeJump(fromVersion, toVersion string) error {
	for _, jumpVer := range unsupportedJumps {
		r1, _ := lib.CompareVersionsIgnoreRevision(fromVersion, jumpVer)
		r2, _ := lib.CompareVersionsIgnoreRevision(toVersion, jumpVer)

		if (r1 < 0 && r2 > 0) || (r2 < 0 && r1 > 0) {
			return fmt.Errorf("version change not allowed from %s to %s - jump required to version %s",
				fromVersion, toVersion, jumpVer)
		}
	}

	return nil
}
