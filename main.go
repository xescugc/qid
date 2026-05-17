package main

import (
	"log"

	"github.com/xescugc/pikoci/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
