package provider

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"al.essio.dev/pkg/shellescape"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/melbahja/goph"
)

var (
	dockerSizePattern      = regexp.MustCompile(`(?i)^[1-9][0-9]*(b|k|kb|m|mb|g|gb)?$`)
	dockerSizePartsPattern = regexp.MustCompile(`(?i)^([1-9][0-9]*)(b|k|kb|m|mb|g|gb)?$`)
)

func databaseServiceMemorySchema(service string) *schema.Schema {
	return &schema.Schema{
		Type:         schema.TypeInt,
		Optional:     true,
		Computed:     true,
		ForceNew:     true,
		ValidateFunc: validation.IntAtLeast(1),
		Description:  fmt.Sprintf("Container memory limit in megabytes. The %s plugin accepts this only when creating or cloning a service, so changing it requires replacing the database service and should be protected with lifecycle.prevent_destroy.", service),
	}
}

func databaseServiceShmSizeSchema(service string) *schema.Schema {
	return &schema.Schema{
		Type:             schema.TypeString,
		Optional:         true,
		Computed:         true,
		ForceNew:         true,
		ValidateFunc:     validation.StringMatch(dockerSizePattern, "must be a positive Docker size such as 256m"),
		DiffSuppressFunc: suppressEquivalentDockerSizes,
		Description:      fmt.Sprintf("Shared-memory size for the %s container, for example `256m`. The plugin accepts this only when creating, cloning, or upgrading a service, so changing it requires replacing the database service and should be protected with lifecycle.prevent_destroy.", service),
	}
}

func readDatabaseServiceLimits(service *DokkuGenericService, client *goph.Client) error {
	script := `if [ -f /sys/fs/cgroup/memory.max ]; then memory_file=/sys/fs/cgroup/memory.max; else memory_file=/sys/fs/cgroup/memory/memory.limit_in_bytes; fi
printf 'terraform-memory=%s\n' "$(cat "$memory_file")"
set -- $(df -B1 /dev/shm | tail -n 1)
printf 'terraform-shm=%s\n' "$2"`
	cmd := fmt.Sprintf("%s:enter %s sh -c %s", service.CmdName, service.Name, shellescape.Quote(script))
	res := run(client, cmd)
	if res.err != nil {
		return res.err
	}

	return setDatabaseServiceLimitsFromOutput(service, res.stdout)
}

func setDatabaseServiceLimitsFromOutput(service *DokkuGenericService, output string) error {
	memoryRead := false
	shmRead := false

	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "terraform-memory="):
			memoryRead = true
			memory := strings.TrimPrefix(line, "terraform-memory=")
			if memory == "max" {
				service.MemoryMB = 0
				continue
			}

			bytes, err := strconv.ParseInt(memory, 10, 64)
			if err != nil {
				return fmt.Errorf("parse %s memory limit %q: %w", service.CmdName, memory, err)
			}
			if bytes >= 1<<60 {
				service.MemoryMB = 0
				continue
			}
			service.MemoryMB = int(bytes / (1024 * 1024))

		case strings.HasPrefix(line, "terraform-shm="):
			shmRead = true
			bytes, err := strconv.ParseInt(strings.TrimPrefix(line, "terraform-shm="), 10, 64)
			if err != nil {
				return fmt.Errorf("parse %s shared-memory size: %w", service.CmdName, err)
			}
			service.ShmSize = formatDockerSize(bytes)
		}
	}

	if !memoryRead || !shmRead {
		return fmt.Errorf("read %s resource limits: expected memory and shared-memory output", service.CmdName)
	}
	service.LimitsRead = true
	return nil
}

func suppressEquivalentDockerSizes(_, old, new string, _ *schema.ResourceData) bool {
	oldBytes, oldOK := parseDockerSize(old)
	newBytes, newOK := parseDockerSize(new)
	return oldOK && newOK && oldBytes == newBytes
}

func parseDockerSize(value string) (int64, bool) {
	match := dockerSizePartsPattern.FindStringSubmatch(value)
	if match == nil {
		return 0, false
	}

	number, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		return 0, false
	}

	multiplier := int64(1)
	switch strings.ToLower(match[2]) {
	case "k", "kb":
		multiplier = 1024
	case "m", "mb":
		multiplier = 1024 * 1024
	case "g", "gb":
		multiplier = 1024 * 1024 * 1024
	}
	if number > math.MaxInt64/multiplier {
		return 0, false
	}
	return number * multiplier, true
}

func formatDockerSize(bytes int64) string {
	const (
		gib = int64(1024 * 1024 * 1024)
		mib = int64(1024 * 1024)
		kib = int64(1024)
	)

	switch {
	case bytes%gib == 0:
		return fmt.Sprintf("%dg", bytes/gib)
	case bytes%mib == 0:
		return fmt.Sprintf("%dm", bytes/mib)
	case bytes%kib == 0:
		return fmt.Sprintf("%dk", bytes/kib)
	default:
		return strconv.FormatInt(bytes, 10)
	}
}
