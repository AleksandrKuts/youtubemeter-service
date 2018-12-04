package main

import (
	"fmt"
	"github.com/AleksandrKuts/youtubemeter-service/backend/server"
)
const versionMajor = "1.0"

var (
	version string
)

func main() {
	fmt.Printf("version: %s.%s\n", versionMajor, version)
	server.StartService(versionMajor, version)
}
