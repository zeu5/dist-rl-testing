package main

import "github.com/zeu5/dist-rl-testing/benchmarks/cmd"

func main() {
	cmd := cmd.RootCommand()
	cmd.Execute()
}
