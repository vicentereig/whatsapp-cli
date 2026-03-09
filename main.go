package main

import "github.com/vicentereig/whatsapp-cli/cmd"

// version is overridden at build time via -ldflags "-X main.version=X.Y.Z"
var version = "1.3.1"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
