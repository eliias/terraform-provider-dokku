package provider

import (
	"testing"

	"github.com/blang/semver"
)

func TestProviderSchemaIsValid(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("provider schema is invalid: %v", err)
	}
}

func TestProviderSupportsSSHAgentAuthentication(t *testing.T) {
	provider := Provider()

	if provider.Schema["ssh_cert"].Required {
		t.Fatal("ssh_cert must be optional when SSH agent authentication is available")
	}

	sshAgentSocket, ok := provider.Schema["ssh_agent_socket"]
	if !ok {
		t.Fatal("provider schema does not contain ssh_agent_socket")
	}
	if !sshAgentSocket.Optional {
		t.Fatal("ssh_agent_socket must be optional")
	}
}

func TestPrefixedCommand(t *testing.T) {
	t.Cleanup(func() {
		DOKKU_COMMAND_PREFIX = ""
	})

	DOKKU_COMMAND_PREFIX = ""
	if got := prefixedCommand("version"); got != "version" {
		t.Fatalf("unexpected command without prefix: %q", got)
	}

	DOKKU_COMMAND_PREFIX = "dokku"
	if got := prefixedCommand("version"); got != "dokku version" {
		t.Fatalf("unexpected command with prefix: %q", got)
	}
}

func TestSupportedDokkuVersions(t *testing.T) {
	compatible, err := semver.ParseRange(testedDokkuVersions)
	if err != nil {
		t.Fatalf("invalid tested Dokku version range: %v", err)
	}

	for _, version := range []string{"0.30.0", "0.36.9", "0.38.25"} {
		if !compatible(semver.MustParse(version)) {
			t.Errorf("expected Dokku %s to be supported", version)
		}
	}

	for _, version := range []string{"0.29.9", "0.39.0"} {
		if compatible(semver.MustParse(version)) {
			t.Errorf("expected Dokku %s to be outside the supported range", version)
		}
	}
}
