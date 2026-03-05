package util

import (
	"bytes"
	"errors"
	"os/exec"
)

type Pair[T, U any] struct {
	First  T
	Second U
}

func RunCommand(command string) error {
	if len(command) == 0 {
		return errors.New("no command was specified")
	}

	cmd := exec.Command("bash", "-c", command)

  var stderr bytes.Buffer
  cmd.Stderr = &stderr
  _, err := cmd.Output()
	if err != nil {
		return errors.Join(errors.New("an error occurred while running the command"), err, errors.New(stderr.String()))
	}

	return nil
}
