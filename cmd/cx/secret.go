package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// readSecret prompts on stderr so that piping cx's stdout stays clean, and
// suppresses terminal echo so the token is never painted on screen.
func readSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)

	restoreEcho := suppressEcho()
	defer restoreEcho()

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	fmt.Fprintln(os.Stderr)
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func suppressEcho() func() {
	if !stdinIsTerminal() {
		return func() {}
	}
	if err := stty("-echo"); err != nil {
		return func() {}
	}
	return func() { _ = stty("echo") }
}

func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func stty(mode string) error {
	cmd := exec.Command("stty", mode)
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
