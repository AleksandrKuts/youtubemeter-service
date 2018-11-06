package main

import (
	"fmt"
	"github.com/AleksandrKuts/youtubemeter-service/collector/server"
)

const versionMajor = "0.1"

var (
	version string
)

func main() {
	fmt.Printf("version: %s.%s\n", versionMajor, version)
	server.StartService()
}
