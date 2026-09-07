package docker

import (
	"context"
	"strings"
	"testing"
)

const inspectJSON = `[{
  "Id": "sha256full0123456789abcdef",
  "Name": "/my-app",
  "Image": "sha256:img",
  "State": {"Status": "running"},
  "Config": {"Image": "ghcr.io/takexito/rust-dev:1", "Labels": {"devcontainer.local_folder": "/home/x/projects/my-app"}},
  "HostConfig": {"Memory": 8589934592, "NanoCpus": 6000000000},
  "NetworkSettings": {"Networks": {"bridge": {"IPAddress": "172.17.0.5"}}}
}]`

func TestParseInspect(t *testing.T) {
	cts, err := ParseInspect([]byte(inspectJSON))
	if err != nil {
		t.Fatal(err)
	}
	if len(cts) != 1 {
		t.Fatalf("len = %d", len(cts))
	}
	c := cts[0]
	if c.Name != "my-app" || c.Project() != "my-app" || c.IP != "172.17.0.5" || c.State != "running" {
		t.Errorf("container = %+v", c)
	}
	if c.Memory != 8589934592 || c.NanoCPUs != 6000000000 || c.Image != "ghcr.io/takexito/rust-dev:1" {
		t.Errorf("limits/image = %+v", c)
	}
	if c.ShortID() != "sha256full01" {
		t.Errorf("ShortID = %q", c.ShortID())
	}
}

func TestParseStats(t *testing.T) {
	out := "abc123456789\t0.50%\t1.2GiB / 8GiB\nbad line\n"
	s := ParseStats([]byte(out))
	if s["abc123456789"] != (Stat{CPU: "0.50%", Mem: "1.2GiB/8GiB"}) {
		t.Errorf("stats = %+v", s)
	}
	if len(s) != 1 {
		t.Errorf("len = %d", len(s))
	}
}

func TestParseImages(t *testing.T) {
	out := "sha256:a\tvsc-x\tlatest\nsha256:b\tghcr.io/takexito/base-dev\t1\n"
	im := ParseImages([]byte(out))
	if len(im) != 2 || im[0] != (Image{ID: "sha256:a", Repo: "vsc-x", Tag: "latest"}) {
		t.Errorf("images = %+v", im)
	}
}

func TestVolumeCreateLabelsSorted(t *testing.T) {
	var calls []string
	c := &Client{Run: func(_ context.Context, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		if args[0] == "volume" && args[1] == "inspect" {
			return nil, context.Canceled // тома нет
		}
		return nil, nil
	}}
	err := c.VolumeCreate(context.Background(), "app-ssh", map[string]string{"devc.stack": "rust", "devc.project": "app"})
	if err != nil {
		t.Fatal(err)
	}
	want := "volume create --label devc.project=app --label devc.stack=rust app-ssh"
	if len(calls) != 2 || calls[1] != want {
		t.Errorf("calls = %q", calls)
	}
}

func TestVolumeCreateSkipsExisting(t *testing.T) {
	var calls int
	c := &Client{Run: func(_ context.Context, args ...string) ([]byte, error) {
		calls++
		return []byte("app-ssh\n"), nil
	}}
	if err := c.VolumeCreate(context.Background(), "app-ssh", nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, ожидался только inspect", calls)
	}
}
