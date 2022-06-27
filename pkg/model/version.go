package model

import "fmt"

type Version struct {
	Major int
	Minor int
}

func (v Version) String() string {
	return fmt.Sprintf("v%d.%d", v.Major, v.Minor)
}

func (v Version) NextMajor() Version {
	return Version{Major: v.Major + 1, Minor: 0}
}

func (v Version) NextMinor() Version {
	return Version{Major: v.Major, Minor: v.Minor + 1}
}
