package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HOME", "/home/x")
	for _, k := range []string{"DEVC_PROJECTS_DIR", "DEVC_REGISTRY", "DEVC_IMAGE_RUST", "DEVC_REPO_DIR"} {
		t.Setenv(k, "")
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.ProjectsDir != "/home/x/projects" {
		t.Errorf("ProjectsDir = %q", c.ProjectsDir)
	}
	if c.Images["rust"] != "ghcr.io/takexito/rust-dev:1" {
		t.Errorf("Images[rust] = %q", c.Images["rust"])
	}
	if c.RepoDir != "/home/x/projects/devcontainer-features" {
		t.Errorf("RepoDir = %q", c.RepoDir)
	}
	if got := c.ProjectDir("app"); got != "/home/x/projects/app" {
		t.Errorf("ProjectDir = %q", got)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("DEVC_PROJECTS_DIR", "/srv/p")
	t.Setenv("DEVC_REGISTRY", "registry.local/me")
	t.Setenv("DEVC_IMAGE_WEB", "custom/web:2")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Images["web"] != "custom/web:2" {
		t.Errorf("Images[web] = %q", c.Images["web"])
	}
	if c.Images["base"] != "registry.local/me/base-dev:1" {
		t.Errorf("Images[base] = %q", c.Images["base"])
	}
	if c.KmplspSrc != "/srv/p/kotlin-lsp-kmp" {
		t.Errorf("KmplspSrc = %q", c.KmplspSrc)
	}
}
