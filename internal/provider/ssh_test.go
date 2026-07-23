package provider

import (
	"errors"
	"testing"
)

func TestRetryableSSHSessionOpenError(t *testing.T) {
	if !isRetryableSSHSessionOpenError(errors.New("ssh: rejected: connect failed (open failed)")) {
		t.Fatal("expected channel-open rejection to be retryable")
	}
	if isRetryableSSHSessionOpenError(errors.New("Process exited with status 1")) {
		t.Fatal("must not retry a command that reached the server")
	}
	if isRetryableSSHSessionOpenError(nil) {
		t.Fatal("must not retry a successful command")
	}
}
