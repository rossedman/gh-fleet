// Command gh-fleet is a gh CLI extension that applies one change across
// many repositories and tracks the resulting pull requests to completion.
package main

import "github.com/rossedman/gh-fleet/cmd"

// main runs the root command.
func main() {
	cmd.Execute()
}
