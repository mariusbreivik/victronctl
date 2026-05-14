package main

import (
	"log"

	"github.com/mariusbreivik/victronctl/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
