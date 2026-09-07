package prune

import (
	"testing"

	"github.com/Takexito/devcontainer-features/internal/docker"
)

func TestPlan(t *testing.T) {
	images := []docker.Image{
		{ID: "sha256:a", Repo: "vsc-old-abc-uid", Tag: "latest"},
		{ID: "sha256:b", Repo: "vsc-live-def-uid", Tag: "latest"},
		{ID: "sha256:c", Repo: "ghcr.io/takexito/android-dev", Tag: "1"},
		{ID: "sha256:c", Repo: "ghcr.io/takexito/android-dev", Tag: "latest"},
		{ID: "sha256:d", Repo: "vsc-shared", Tag: "one"},
		{ID: "sha256:d", Repo: "ghcr.io/takexito/base-dev", Tag: "1"},
		{ID: "sha256:e", Repo: "mcr.microsoft.com/devcontainers/base", Tag: "debian"},
	}
	used := map[string]bool{"sha256:b": true}
	got := Plan(images, used, "ghcr.io/takexito/")
	if len(got) != 1 || got[0].Ref != "vsc-old-abc-uid:latest" {
		t.Errorf("Plan = %+v", got)
	}
}
