package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStripJSONC(t *testing.T) {
	in := `{
  // комментарий
  "a": "http://x/y", /* блок */ "b": [1, 2,],
  "c": "с \"кавычкой\" и // не комментарием",
}`
	var v map[string]any
	if err := json.Unmarshal(StripJSONC([]byte(in)), &v); err != nil {
		t.Fatalf("%v\n%s", err, StripJSONC([]byte(in)))
	}
	if v["a"] != "http://x/y" || len(v["b"].([]any)) != 2 || !strings.Contains(v["c"].(string), "//") {
		t.Errorf("v = %+v", v)
	}
}

func TestValidateName(t *testing.T) {
	for _, ok := range []string{"my-app", "app2", "a.b_c"} {
		if err := ValidateName(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "-x", "My App", "dev", "dev-claude", "a/b", "a b"} {
		if err := ValidateName(bad); err == nil {
			t.Errorf("%q: ожидалась ошибка", bad)
		}
	}
}

func write(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".devcontainer"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".devcontainer", "devcontainer.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadModes(t *testing.T) {
	dir := t.TempDir()
	p, err := Load("app", dir)
	if err != nil || p.Mode != ModeNone || p.Config != nil {
		t.Fatalf("пустой каталог: %+v, %v", p, err)
	}

	write(t, dir, `{
  "name": "cpa",
  "build": { "dockerfile": "Dockerfile" },
  "mounts": [
    "source=dev-cargo,target=/home/dev/.cargo,type=volume",
    "source=cpa-target,target=/home/dev/target,type=volume",
    { "source": "cpa-ssh", "target": "/home/dev/.ssh", "type": "volume" },
    "source=dev-kotlin-lsp-kmp,target=/opt/kmp,type=volume,readonly",
    "source=${localEnv:HOME}/.ssh/authorized_keys,target=/etc/ssh/authorized_keys.host,type=bind,readonly"
  ], // хвостовая запятая и комментарий
}`)
	p, err = Load("app", dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Mode != ModeCustom || p.Image != "" {
		t.Errorf("custom: %+v", p)
	}
	if got := strings.Join(p.VolumeNames(), ","); got != "cpa-target,cpa-ssh" {
		t.Errorf("VolumeNames = %q", got)
	}

	write(t, dir, `{"image":"ghcr.io/takexito/rust-dev:1","customizations":{"devc":{"stack":"rust","generator":1,"mem":"8g"}}}`)
	p, err = Load("app", dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Mode != ModeGenerated || p.Marker.Stack != "rust" || p.Marker.Mem != "8g" || p.Image != "ghcr.io/takexito/rust-dev:1" {
		t.Errorf("generated: %+v", p)
	}

	write(t, dir, `{ this is not json `)
	if _, err := Load("app", dir); err == nil {
		t.Error("битый JSON должен быть ошибкой")
	}
}
