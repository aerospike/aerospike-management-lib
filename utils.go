package lib

import (
	"fmt"
	"strconv"
	"strings"
)

// CompareVersions compares Aerospike Server versions
// if version1 == version2 returns 0
// else if version1 < version2 returns -1
// else returns 1
func CompareVersions(version1, version2 string) (int, error) {
	if version1 == "" || version2 == "" {
		return 0, fmt.Errorf("wrong versions to compare")
	}

	if version1 == version2 {
		return 0, nil
	}

	// Ignoring extra comment tag... found in git source code build
	v1 := strings.Split(version1, "-")[0]
	v2 := strings.Split(version2, "-")[0]

	if v1 == v2 {
		return 0, nil
	}

	verElems1 := strings.Split(v1, ".")
	verElems2 := strings.Split(v2, ".")

	minLen := len(verElems1)
	if len(verElems2) < minLen {
		minLen = len(verElems2)
	}

	for i := 0; i < minLen; i++ {
		ve1, err := strconv.Atoi(verElems1[i])
		if err != nil {
			return 0, fmt.Errorf("wrong version to compare")
		}

		ve2, err := strconv.Atoi(verElems2[i])
		if err != nil {
			return 0, fmt.Errorf("wrong version to compare")
		}

		if ve1 > ve2 {
			return 1, nil
		} else if ve1 < ve2 {
			return -1, nil
		}
	}

	if len(verElems1) > len(verElems2) {
		return 1, nil
	}

	if len(verElems1) < len(verElems2) {
		return -1, nil
	}

	return 0, nil
}

// CompareVersionsIgnoreRevision compares Aerospike Server versions ignoring
// revisions and builds.
// if version1 == version2 returns 0
// else if version1 < version2 returns -1
// else returns 1
func CompareVersionsIgnoreRevision(version1, version2 string) (int, error) {
	if version1 == "" || version2 == "" {
		return 0, fmt.Errorf("wrong versions to compare")
	}

	if version1 == version2 {
		return 0, nil
	}

	// Ignoring extra comment tag... found in git source code build
	v1 := strings.Split(version1, "-")[0]
	v2 := strings.Split(version2, "-")[0]

	if v1 == v2 {
		return 0, nil
	}

	verElems1 := strings.Split(v1, ".")
	verElems2 := strings.Split(v2, ".")

	minLen := len(verElems1)
	if len(verElems2) < minLen {
		minLen = len(verElems2)
	}

	if minLen > 2 {
		// Force comparison of only major and minor version.
		minLen = 2
	}

	for i := 0; i < minLen; i++ {
		ve1, err := strconv.Atoi(verElems1[i])
		if err != nil {
			return 0, fmt.Errorf("wrong version to compare")
		}

		ve2, err := strconv.Atoi(verElems2[i])
		if err != nil {
			return 0, fmt.Errorf("wrong version to compare")
		}

		if ve1 > ve2 {
			return 1, nil
		} else if ve1 < ve2 {
			return -1, nil
		}
	}

	return 0, nil
}

// ContainsString check whether list contains given string
func ContainsString(list []string, ele string) bool {
	for _, listEle := range list {
		if strings.EqualFold(ele, listEle) {
			return true
		}
	}

	return false
}

func GetKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}
