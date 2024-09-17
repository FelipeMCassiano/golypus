/*
Copyright © 2024 Cassiano felipecassianofmc@gmail.com
*/
package main

import (
	"log"

	"github.com/FelipeMCassiano/golypus/internal/commands"
	"go.uber.org/goleak"
)

func main() {
	commands.Execute()
	if err := goleak.Find(); err != nil {
		log.Fatal(err)
		return
	}
}
