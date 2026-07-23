package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/melbahja/goph"

	"al.essio.dev/pkg/shellescape"
)

type dokkuStorageMount struct {
	App           string
	StorageEntry  string
	ContainerPath string
	Phases        []string
	ProcessType   string
	Subpath       string
	Readonly      bool
	VolumeOptions string
	VolumeChown   string
}

func resourceStorageMount() *schema.Resource {
	return &schema.Resource{
		Description:   "Mounts a named Dokku storage entry into an application. Changes affect containers created by a subsequent restart or deployment.",
		CreateContext: resourceStorageMountCreate,
		ReadContext:   resourceStorageMountRead,
		UpdateContext: resourceStorageMountUpdate,
		DeleteContext: resourceStorageMountDelete,
		Schema: map[string]*schema.Schema{
			"app": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Dokku application receiving the mount.",
			},
			"storage_entry": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Named Dokku storage entry to mount.",
			},
			"container_path": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Absolute path where the entry is mounted inside the container.",
			},
			"phases": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{"deploy", "run"}, false),
				},
				Description: "Container phases receiving the mount.",
			},
			"process_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "_default_",
				ForceNew:    true,
				Description: "Process type receiving the mount, or _default_ for every process type.",
			},
			"subpath": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Subdirectory within the storage entry to mount.",
			},
			"readonly": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to mount the entry read-only.",
			},
			"volume_options": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Comma-separated Docker mount options.",
			},
			"volume_chown": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Ownership mode applied by Dokku at mount time.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: importStorageMount,
		},
	}
}

func resourceStorageMountCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	mount := storageMountFromResourceData(d)
	if err := setStorageMount(mount, m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(storageMountID(mount))
	return resourceStorageMountRead(ctx, d, m)
}

func resourceStorageMountRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	wanted := storageMountFromResourceData(d)
	mounts, err := readStorageMounts(wanted.App, m.(*goph.Client))
	if err != nil {
		return diag.FromErr(err)
	}

	for _, mount := range mounts {
		if mount.StorageEntry == wanted.StorageEntry &&
			mount.ContainerPath == wanted.ContainerPath &&
			mount.ProcessType == wanted.ProcessType {
			setStorageMountOnResourceData(d, mount)
			d.SetId(storageMountID(mount))
			return nil
		}
	}

	d.SetId("")
	return nil
}

func resourceStorageMountUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	mount := storageMountFromResourceData(d)
	if err := setStorageMount(mount, m.(*goph.Client)); err != nil {
		return diag.FromErr(err)
	}

	return resourceStorageMountRead(ctx, d, m)
}

func resourceStorageMountDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	mount := storageMountFromResourceData(d)
	res := run(
		m.(*goph.Client),
		fmt.Sprintf(
			"storage:unmount %s %s --container-dir %s",
			shellescape.Quote(mount.App),
			shellescape.Quote(mount.StorageEntry),
			shellescape.Quote(mount.ContainerPath),
		),
	)
	if res.err != nil {
		return diag.FromErr(res.err)
	}

	d.SetId("")
	return nil
}

func setStorageMount(mount dokkuStorageMount, client *goph.Client) error {
	args := []string{
		shellescape.Quote(mount.App),
		shellescape.Quote(mount.StorageEntry),
		"--container-dir",
		shellescape.Quote(mount.ContainerPath),
	}

	phases := append([]string(nil), mount.Phases...)
	sort.Strings(phases)
	for _, phase := range phases {
		args = append(args, "--phase", shellescape.Quote(phase))
	}
	if mount.ProcessType != "_default_" {
		args = append(args, "--process-type", shellescape.Quote(mount.ProcessType))
	}
	if mount.Subpath != "" {
		args = append(args, "--volume-subpath", shellescape.Quote(mount.Subpath))
	}
	if mount.Readonly {
		args = append(args, "--volume-readonly")
	}
	if mount.VolumeOptions != "" {
		args = append(args, "--volume-options", shellescape.Quote(mount.VolumeOptions))
	}
	if mount.VolumeChown != "" {
		args = append(args, "--volume-chown", shellescape.Quote(mount.VolumeChown))
	}

	res := run(client, fmt.Sprintf("storage:mount %s", joinCommandArgs(args)))
	return res.err
}

func readStorageMounts(app string, client *goph.Client) ([]dokkuStorageMount, error) {
	res := run(client, fmt.Sprintf("storage:report %s --format json", shellescape.Quote(app)))
	if res.err != nil {
		return nil, res.err
	}
	return parseStorageMountReport(app, res.stdout)
}

func parseStorageMountReport(app, stdout string) ([]dokkuStorageMount, error) {
	var report map[string]string
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		return nil, fmt.Errorf("parsing storage report for %q: %w", app, err)
	}

	attachments := make(map[string]map[string]string)
	for key, value := range report {
		parts := strings.SplitN(key, ".", 3)
		if len(parts) != 3 || parts[0] != "attachment" {
			continue
		}
		if _, ok := attachments[parts[1]]; !ok {
			attachments[parts[1]] = make(map[string]string)
		}
		attachments[parts[1]][parts[2]] = value
	}

	indices := make([]string, 0, len(attachments))
	for index := range attachments {
		indices = append(indices, index)
	}
	sort.Strings(indices)

	mounts := make([]dokkuStorageMount, 0, len(indices))
	for _, index := range indices {
		attachment := attachments[index]
		readonly, err := strconv.ParseBool(attachment["readonly"])
		if err != nil {
			return nil, fmt.Errorf("parsing storage attachment %s readonly value: %w", index, err)
		}

		processType := attachment["process-type"]
		if processType == "" {
			processType = "_default_"
		}

		mounts = append(mounts, dokkuStorageMount{
			App:           app,
			StorageEntry:  attachment["entry-name"],
			ContainerPath: attachment["container-path"],
			Phases:        splitNonEmpty(attachment["phases"], ","),
			ProcessType:   processType,
			Subpath:       attachment["subpath"],
			Readonly:      readonly,
			VolumeOptions: attachment["volume-options"],
			VolumeChown:   attachment["volume-chown"],
		})
	}
	return mounts, nil
}

func storageMountFromResourceData(d *schema.ResourceData) dokkuStorageMount {
	return dokkuStorageMount{
		App:           d.Get("app").(string),
		StorageEntry:  d.Get("storage_entry").(string),
		ContainerPath: d.Get("container_path").(string),
		Phases:        interfaceSliceToStrSlice(d.Get("phases").(*schema.Set).List()),
		ProcessType:   d.Get("process_type").(string),
		Subpath:       d.Get("subpath").(string),
		Readonly:      d.Get("readonly").(bool),
		VolumeOptions: d.Get("volume_options").(string),
		VolumeChown:   d.Get("volume_chown").(string),
	}
}

func setStorageMountOnResourceData(d *schema.ResourceData, mount dokkuStorageMount) {
	d.Set("app", mount.App)
	d.Set("storage_entry", mount.StorageEntry)
	d.Set("container_path", mount.ContainerPath)
	d.Set("phases", mount.Phases)
	d.Set("process_type", mount.ProcessType)
	d.Set("subpath", mount.Subpath)
	d.Set("readonly", mount.Readonly)
	d.Set("volume_options", mount.VolumeOptions)
	d.Set("volume_chown", mount.VolumeChown)
}

func storageMountID(mount dokkuStorageMount) string {
	return strings.Join([]string{mount.App, mount.StorageEntry, mount.ContainerPath, mount.ProcessType}, "|")
}

func importStorageMount(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	parts := strings.Split(d.Id(), "|")
	if len(parts) != 4 {
		return nil, fmt.Errorf("expected import ID app|storage-entry|container-path|process-type")
	}

	d.Set("app", parts[0])
	d.Set("storage_entry", parts[1])
	d.Set("container_path", parts[2])
	d.Set("process_type", parts[3])
	d.SetId(d.Id())
	return []*schema.ResourceData{d}, nil
}

func splitNonEmpty(value, separator string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, separator)
}
