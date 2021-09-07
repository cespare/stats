package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cespare/subcmd"
)

var cmds = []subcmd.Command{
	{
		Name:        "summarize",
		Description: "Display summary statistics for a sequence of numbers",
		Do:          summarize,
	},
}

const version = "0.1.1"

func main() {
	log.SetFlags(0)
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "-version", "--version", "-v":
			fmt.Println(version)
			return
		}
	}
	subcmd.Run(cmds)
}
