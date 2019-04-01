package main

import (
	"fmt"
	"github.com/AleksandrKuts/youtubemeter-service/collector"
)

const versionMajor = "0.2"

var (
	version string
)

func main() {
	fmt.Printf("version: %s.%s\n", versionMajor, version)
	collector.StartService(versionMajor, version)
}
