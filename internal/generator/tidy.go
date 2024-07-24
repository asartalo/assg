package generator

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

var tidyArgs = []string{
	"--show-body-only",
	"auto",
	// "--show-errors",
	// "0",
	"--gnu-emacs",
	"yes",
	"-q",
	"-i",
	"-m",
	"-w",
	"160",
	"--indent-spaces",
	"2",
	"-ashtml",
	"-utf8",
	"--tidy-mark",
	"no",
}

func TidyHtml(pathToTidy string) error {
	args := append(tidyArgs, pathToTidy)
	cmd := exec.Command("tidy", args...)
	cmd.Stdout = os.Stdout

	buf := bytes.NewBufferString("")
	cmd.Stderr = buf

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error running tidy on %s:\n%s", pathToTidy, buf.String())
	}

	return nil
}
