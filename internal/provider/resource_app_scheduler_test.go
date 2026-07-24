package provider

import (
	"encoding/json"
	"testing"
)

func TestParseSchedulerReports(t *testing.T) {
	t.Parallel()

	var generic dokkuAppSchedulerReport
	if err := json.Unmarshal([]byte(`{"selected":"","shell":"","computed-selected":"docker-local","computed-shell":"sh"}`), &generic); err != nil {
		t.Fatal(err)
	}
	if generic.Selected != "" || generic.ComputedSelected != "docker-local" || generic.ComputedShell != "sh" {
		t.Fatalf("unexpected generic scheduler report: %#v", generic)
	}

	var dockerLocal dokkuAppDockerLocalSchedulerReport
	if err := json.Unmarshal([]byte(`{"init-process":"","parallel-schedule-count":"","computed-init-process":"true","computed-parallel-schedule-count":"1"}`), &dockerLocal); err != nil {
		t.Fatal(err)
	}
	if dockerLocal.InitProcess != "" || dockerLocal.ComputedInitProcess != "true" || dockerLocal.ComputedParallelScheduleCount != "1" {
		t.Fatalf("unexpected docker-local scheduler report: %#v", dockerLocal)
	}
}
