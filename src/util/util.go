package util

import (
	"errors"
	"os/exec"
	"strings"
)

type Pair[T, U any] struct {
	First  T
	Second U
}

func RunCommand(command string) error {
	words := strings.Split(command, " ")

	if len(words) == 0 {
		return errors.New("no command was specified")
	}

	cmd := exec.Command(words[0], words[1:]...)

	if err := cmd.Run(); err != nil {
		return errors.Join(errors.New("an error occurred while running the command"), err)
	}

	return nil
}
