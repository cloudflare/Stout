package git

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
)

func Exists() bool {
	_, err := ioutil.ReadDir(".git")
	if err != nil {
		return false
	}

	return true
}

func Exec(args ...string) error {
	cmd := exec.Command("git", args...)
	_, err := cmd.Output()

	return err
}

func ChainExecLog(commands [][]string) error {
	for _, command := range commands {
		cmd := exec.Command("git", command...)
		out, err := cmd.CombinedOutput()

		fmt.Println("git", strings.Join(command, " "))
		fmt.Println(string(out))

		if err != nil {
			return err
		}
	}

	return nil
}
