package main

import "github.com/develerik/office-diff/cmd"

// TODO: detect moved files (maybe hashing)
// TODO: don't add binary content to diff (see git example)

func main() {
	cmd.Execute()
}
