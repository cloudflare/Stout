package main

import (
	"flag"
	"fmt"
)

func printUsage() {
	fmt.Println(`Stout Static Deploy Tool
Supports three commands, create, deploy and rollback.

Example Usage:

To create a site which will be hosted at my.awesome.website:

stout create --bucket my.awesome.website --key AWS_KEY --secret AWS_SECRET

To deploy the current folder to the root of the my.awesome.website site:

stout deploy --bucket my.awesome.website --key AWS_KEY --secret AWS_SECRET

To rollback to a specific deploy:

stout rollback --bucket my.awesome.website --key AWS_KEY --secret AWS_SECRET c4a22bf94de1

See the README for more configuration information.
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
