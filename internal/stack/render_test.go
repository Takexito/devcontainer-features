package stack

import (
	"bytes"
	"errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "перезаписать golden-файлы")

func golden(t *testing.T, path string, got []byte) {
	t.Helper()
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v (go test -update, чтобы создать)", path, err)
	}
	if !bytes.Equal(want, got) {
		t.Errorf("%s отличается от golden; go test -update после проверки глазами\n--- got ---\n%s", path, got)
	}
}

func TestRenderGolden(t *testing.T) {
	for _, name := range Names() {
		st, ok := Get(name)
		if !ok {
			t.Fatalf("стек %s не зарегистрирован", name)
		}
		// Адреса фиксированные: golden должен быть стабильным.
		f, err := Render("demo", st, Options{
			Image:           "ghcr.io/takexito/" + name + "-dev:1",
			MobileMCPURL:    "http://172.17.0.1:8971/mcp",
			MobileMCPMacURL: "http://100.64.0.2:8971/mcp",
		})
		if err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join("testdata", name)
		golden(t, filepath.Join(dir, "devcontainer.json"), f.Devcontainer)
		golden(t, filepath.Join(dir, "post-create.sh"), f.PostCreate)
		zed := filepath.Join(dir, "zed.json")
		if f.Zed != nil {
			golden(t, zed, f.Zed)
		} else if _, err := os.Stat(zed); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s: у стека %s нет настроек Zed, а golden есть", zed, name)
		}
		mcp := filepath.Join(dir, "mcp.json")
		if f.MCP != nil {
			golden(t, mcp, f.MCP)
		} else if _, err := os.Stat(mcp); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s: стеку %s mobile-mcp не нужен, а golden есть", mcp, name)
		}
	}
}

func TestRenderMobileMCPOnlyAndroid(t *testing.T) {
	for _, name := range Names() {
		st, _ := Get(name)
		f, err := Render("app", st, Options{Image: "img", MobileMCPURL: "http://172.17.0.1:8971/mcp"})
		if err != nil {
			t.Fatal(err)
		}
		if (f.MCP != nil) != (name == "android") {
			t.Errorf("стек %s: MCP != nil = %v", name, f.MCP != nil)
		}
	}
}

// Без адресов .mcp.json не пишется: иначе проект ловил бы «failed to connect».
func TestRenderMobileMCPEmpty(t *testing.T) {
	st, _ := Get("android")
	f, err := Render("app", st, Options{Image: "img"})
	if err != nil {
		t.Fatal(err)
	}
	if f.MCP != nil || f.ClaudeServers != nil || f.Gitignore != nil {
		t.Errorf("без адресов ничего не должно генерироваться: %q %v %v", f.MCP, f.ClaudeServers, f.Gitignore)
	}
}

func TestRenderMobileMCPServers(t *testing.T) {
	st, _ := Get("android")
	f, err := Render("app", st, Options{Image: "img", MobileMCPURL: "http://172.17.0.1:8971/mcp"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(f.ClaudeServers, ",") != "mobile" {
		t.Errorf("ClaudeServers = %v", f.ClaudeServers)
	}
	// Транспорт sse, не http: сервер отвечает кадром «event: endpoint».
	if !bytes.Contains(f.MCP, []byte(`"type": "sse"`)) {
		t.Errorf("ожидался транспорт sse:\n%s", f.MCP)
	}
	if bytes.Contains(f.MCP, []byte("mobile-mac")) {
		t.Errorf("mac-сервер не задавали, а он есть:\n%s", f.MCP)
	}
}

func TestRenderProjectVolumes(t *testing.T) {
	st, _ := Get("rust")
	f, err := Render("app", st, Options{Image: "img"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"app-ssh", "app-mise", "app-target"}
	if strings.Join(f.ProjectVolumes, ",") != strings.Join(want, ",") {
		t.Errorf("ProjectVolumes = %v, want %v", f.ProjectVolumes, want)
	}
	if strings.Contains(strings.Join(f.ProjectVolumes, ","), "dev-") {
		t.Errorf("общий том попал в проектные: %v", f.ProjectVolumes)
	}
}

func TestRenderLimits(t *testing.T) {
	st, _ := Get("base")
	f, err := Render("app", st, Options{Image: "img", Mem: "2g", CPUs: "1.5"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"--memory=2g"`, `"--memory-swap=2g"`, `"--cpus=1.5"`, `"mem": "2g"`, `"cpus": "1.5"`} {
		if !bytes.Contains(f.Devcontainer, []byte(want)) {
			t.Errorf("нет %s в:\n%s", want, f.Devcontainer)
		}
	}
	if f.Marker.Mem != "2g" || f.Marker.CPUs != "1.5" || f.Marker.Stack != "base" || f.Marker.Generator != Generator {
		t.Errorf("Marker = %+v", f.Marker)
	}
}

func TestRenderValidation(t *testing.T) {
	st, _ := Get("base")
	cases := []Options{
		{Image: "img", Mem: "8 gb"},
		{Image: "img", CPUs: "six"},
		{Image: ""},
	}
	for _, o := range cases {
		if _, err := Render("app", st, o); err == nil {
			t.Errorf("Render(%+v): ожидалась ошибка", o)
		}
	}
}

func TestRenderReadOnlyNotOwned(t *testing.T) {
	st, _ := Get("android")
	f, err := Render("app", st, Options{Image: "img"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(f.Devcontainer, []byte("source=dev-kotlin-lsp-kmp,target=/opt/kmp,type=volume,readonly")) {
		t.Errorf("readonly-том не в mounts:\n%s", f.Devcontainer)
	}
	if bytes.Contains(f.PostCreate, []byte(`"/opt/kmp"`)) {
		t.Errorf("readonly-том попал в own():\n%s", f.PostCreate)
	}
}

func TestFromMetadata(t *testing.T) {
	label := `[{"id":"ghcr.io/x"},{"customizations":{"vscode":{},"devc":{"stack":"rust","generator":1,"mem":"8g"}},"remoteUser":"dev"}]`
	m, ok := FromMetadata(label)
	if !ok || m.Stack != "rust" || m.Mem != "8g" {
		t.Errorf("FromMetadata = %+v, %v", m, ok)
	}
	if _, ok := FromMetadata(`[{"customizations":{"vscode":{}}}]`); ok {
		t.Error("без devc должен быть false")
	}
	if _, ok := FromMetadata("not json"); ok {
		t.Error("мусор должен быть false")
	}
	if m, ok := FromMetadata(`{"customizations":{"devc":{"stack":"web","generator":1}}}`); !ok || m.Stack != "web" {
		t.Errorf("одиночный объект: %+v, %v", m, ok)
	}
}
