package main

import (
	"log"
	"os"

	"github.com/ctII/filebrowserui/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		log.SetOutput(os.Stderr)
		log.Fatal(err)
	}
}
