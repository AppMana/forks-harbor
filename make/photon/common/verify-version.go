package main

import (
	"fmt"
	"os"

	"github.com/goharbor/harbor/src/pkg/version"
)

func main() {
	if len(os.Args) != 3 || version.ReleaseVersion == "" || version.GitCommit == "" ||
		version.ReleaseVersion != os.Args[1] || version.GitCommit != os.Args[2] {
		panic("Harbor release metadata was not linked correctly")
	}
	fmt.Printf("Verified Harbor release %s, commit %s\n", version.ReleaseVersion, version.GitCommit)
}
