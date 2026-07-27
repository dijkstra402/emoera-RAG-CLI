package main

import (
	"fmt"
	"os"

	"github.com/dijkstra402/emoera-RAG-CLI/cmd"
	"github.com/dijkstra402/emoera-RAG-CLI/internal/apperr"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(apperr.ExitCode(err))
	}
}
