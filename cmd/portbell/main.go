// Command portbell shows every HTTP request that reaches a local port and lets
// you choose the reply. See https://github.com/tejjasdev/portbell.
package main

import (
	"context"
	"os"

	"github.com/tejjasdev/portbell/internal/app"
)

func main() {
	os.Exit(app.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}
