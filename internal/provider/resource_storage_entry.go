package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/melbahja/goph"

	"al.essio.dev/pkg/shellescape"
)

type dokkuStorageEntry struct {
	Name          string `json:"name"`
	Scheduler     string `json:"scheduler"`
	HostPath      string `json:"host_path"`
	SchemaVersion int    `json:"schema_version"`
}

func resourceStorageEntry() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a named docker-local storage entry. Destroying an entry may remove its underlying storage; use lifecycle.prevent_destroy for production data.",
		CreateContext: resourceStorageEntryCreate,
		ReadContext:   resourceStorageEntryRead,
		DeleteContext: resourceStorageEntryDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,43}[a-z0-9])?$`), "must be a DNS-1123 label of at most 45 characters"),
				Description:  "Globally unique Dokku storage entry name.",
			},
			"host_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Host directory or Docker volume backing the entry. Dokku uses its default storage directory when omitted.",
			},
			"scheduler": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"docker-local"}, false),
				Description:  "Scheduler that owns the storage entry. Currently only docker-local is supported.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceStorageEntryCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	args := []string{shellescape.Quote(name)}

	if hostPath, ok := d.GetOk("host_path"); ok {
		args = append(args, shellescape.Quote(hostPath.(string)))
	}
	if scheduler, ok := d.GetOk("scheduler"); ok {
		args = append(args, "--scheduler", shellescape.Quote(scheduler.(string)))
	}

	res := run(m.(*goph.Client), fmt.Sprintf("storage:create %s", joinCommandArgs(args)))
	if res.err != nil {
		return diag.FromErr(res.err)
	}

	d.SetId(name)
	return resourceStorageEntryRead(ctx, d, m)
}

func resourceStorageEntryRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	name := d.Id()
	if name == "" {
		name = d.Get("name").(string)
	}

	entry, exists, err := readStorageEntry(name, m.(*goph.Client))
	if err != nil {
		return diag.FromErr(err)
	}
	if !exists {
		d.SetId("")
		return nil
	}

	d.SetId(entry.Name)
	d.Set("name", entry.Name)
	d.Set("host_path", entry.HostPath)
	d.Set("scheduler", entry.Scheduler)
	return nil
}

func resourceStorageEntryDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	res := run(m.(*goph.Client), fmt.Sprintf("storage:destroy %s --force", shellescape.Quote(d.Id())))
	if res.err != nil {
		return diag.FromErr(res.err)
	}

	d.SetId("")
	return nil
}

func readStorageEntry(name string, client *goph.Client) (dokkuStorageEntry, bool, error) {
	res := run(client, fmt.Sprintf("storage:info %s --format json", shellescape.Quote(name)))
	if res.err != nil {
		if res.status > 0 {
			return dokkuStorageEntry{}, false, nil
		}
		return dokkuStorageEntry{}, false, res.err
	}

	var entry dokkuStorageEntry
	if err := json.Unmarshal([]byte(res.stdout), &entry); err != nil {
		return dokkuStorageEntry{}, false, fmt.Errorf("parsing storage entry %q: %w", name, err)
	}
	return entry, true, nil
}

func joinCommandArgs(args []string) string {
	result := ""
	for _, arg := range args {
		if result != "" {
			result += " "
		}
		result += arg
	}
	return result
}
