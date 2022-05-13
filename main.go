package main

import "github.com/develerik/office-diff/cmd"

var (
	version string
	date    string
)

func main() {
	cmd.Execute(&cmd.Options{
		Version: version,
		Date:    date,
	})
}
