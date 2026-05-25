package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"keyprox/internal/keyprox"
)

func main() {
	if err := keyprox.Execute(context.Background()); err != nil {
		var exitErr *keyprox.ExitError
		if errors.As(err, &exitErr) {
			fmt.Fprintln(os.Stderr, exitErr.Unwrap())
			os.Exit(exitErr.ExitCode())
		}

		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
