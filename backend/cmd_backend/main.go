package main

import (
	"fmt"
	"github.com/AleksandrKuts/youtubemeter-service/backend"
)
const versionMajor = "2.1"

var (
	version string
)

func main() {
	fmt.Printf("version: %s.%s\n", versionMajor, version)
	backend.StartService(versionMajor, version)
}
