package provider

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/melbahja/goph"
)

type SshOutput struct {
	stdout string
	// status code will be 0 if there is no error, otherwise
	// the status code extracted form the error
	status int
	err    error
}

// OpenSSH defaults MaxSessions to 10 per connection. Terraform also defaults
// to 10 concurrent resource operations, and individual resource reads often
// open follow-up sessions before other reads finish. Keep a small margin so
// the shared provider connection never exhausts the server's channel limit.
const maxConcurrentSSHSessions = 8

var sshSessionSlots = make(chan struct{}, maxConcurrentSSHSessions)

// Run a command using the provided SSH client
//
// strings to be removed from logging can also be provided via `sensitiveStrings`
func run(client *goph.Client, cmd string, sensitiveStrings ...string) SshOutput {
	cmd = prefixedCommand(cmd)

	cmdSafe := cmd
	for _, toReplace := range sensitiveStrings {
		cmdSafe = strings.Replace(cmdSafe, toReplace, "*******", -1)
	}

	log.Printf("[DEBUG] SSH: %s", cmdSafe)

	sshSessionSlots <- struct{}{}
	defer func() {
		<-sshSessionSlots
	}()

	stdoutRaw, err := client.Run(cmd)

	stdout := string(stdoutRaw)
	for _, toReplace := range sensitiveStrings {
		stdout = strings.Replace(stdout, toReplace, "*******", -1)
	}

	if err != nil {
		status := parseStatusCode(err.Error())
		log.Printf("[DEBUG] SSH: error status %d from %s: %s", status, cmdSafe, err)
		return SshOutput{
			stdout: stdout,
			status: status,
			err:    fmt.Errorf("SSH command failed with status %d: %s: %w", status, stdout, err),
		}
	} else {
		return SshOutput{
			stdout: stdout,
			status: 0,
			err:    nil,
		}
	}
}

func prefixedCommand(cmd string) string {
	prefix := strings.TrimSpace(DOKKU_COMMAND_PREFIX)
	if prefix == "" {
		return cmd
	}
	return prefix + " " + cmd
}

// TODO add some debug logging
func parseStatusCode(str string) int {
	re := regexp.MustCompile("^Process exited with status ([0-9]+)$")
	found := re.FindStringSubmatch(str)

	if found == nil {
		return 0
	}

	i, err := strconv.Atoi(found[1])

	if err != nil {
		return 0
	}

	return i
}
