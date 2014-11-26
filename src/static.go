package main

import (
	"flag"
	"fmt"
)

func printUsage() {
	fmt.Println(`Stout Static Deploy Tool
Supports two commands, deploy and rollback.

Example Usage:

To deploy the current folder to the root of my-bucket:

stout deploy --bucket my-bucket --key AWS_KEY --secret AWS_SECRET

To rollback to a specific deploy:

stout rollback c4a22bf94de1 --bucket my-bucket --key AWS_KEY --secret AWS_SECRET

See the README for full configuration.
`)
}

func main() {
	flag.Parse()

	command := flag.Arg(0)

	switch command {
	case "help":
		printUsage()
	case "deploy":
		deployCmd()
	case "rollback":
		rollbackCmd()
	default:
		fmt.Println("Command not understood")
		fmt.Println("")
		printUsage()
	}
}
