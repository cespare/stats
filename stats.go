package main

import (
	"log"

	"github.com/cespare/subcmd"
)

var cmds = []subcmd.Command{
	{
		Name:        "summarize",
		Description: "Display summary statistics for a sequence of numbers",
		Do:          summarize,
	},
}

func main() {
	log.SetFlags(0)
	subcmd.Run(cmds)
}
