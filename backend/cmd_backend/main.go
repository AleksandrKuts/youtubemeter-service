package main

import (
	"fmt"
	"github.com/AleksandrKuts/youtubemeter-service/backend"
)
const versionMajor = "1.0"

var (
	version string
)

func main() {
	fmt.Printf("version: %s.%s\n", versionMajor, version)
	backend.StartService(versionMajor, version)
}
