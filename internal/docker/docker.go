// Package docker — тонкая обёртка над docker CLI. Все вызовы идут через две
// функции, которые тесты подменяют: Run возвращает stdout, Stream наследует
// терминал (push, exec, долгие run).
package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// LabelFolder — метка devcontainer CLI с каталогом проекта на хосте.
// Это первичный ключ всей системы: по ней ищут контейнер ls, rm, proxy и exec.
const LabelFolder = "devcontainer.local_folder"

// FolderFilter — фильтр docker ps по каталогу проекта (точное совпадение).
func FolderFilter(dir string) string { return "label=" + LabelFolder + "=" + dir }

type Client struct {
	Run    func(ctx context.Context, args ...string) ([]byte, error)
	Stream func(ctx context.Context, args ...string) error
}

func New() *Client { return &Client{Run: run, Stream: stream} }

func run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func stream(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker %s: %w", args[0], err)
	}
	return nil
}

// Container — то, что devc нужно знать о контейнере.
type Container struct {
	ID       string
	Name     string
	State    string
	Image    string // имя образа из конфига контейнера
	ImageID  string
	Labels   map[string]string
	IP       string
	Memory   int64
	NanoCPUs int64
}

// Project — имя проекта: basename каталога из метки.
func (c Container) Project() string {
	dir := c.Labels[LabelFolder]
	if dir == "" {
		return ""
	}
	return filepath.Base(dir)
}

// ShortID — 12 символов, как в выводе docker ps и docker stats.
func (c Container) ShortID() string {
	if len(c.ID) > 12 {
		return c.ID[:12]
	}
	return c.ID
}

func (c *Client) ContainerIDs(ctx context.Context, all bool, filters ...string) ([]string, error) {
	args := []string{"ps", "-q"}
	if all {
		args = append(args, "-a")
	}
	for _, f := range filters {
		args = append(args, "--filter", f)
	}
	out, err := c.Run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return lines(out), nil
}

type inspectEntry struct {
	ID    string `json:"Id"`
	Name  string `json:"Name"`
	Image string `json:"Image"`
	State struct {
		Status string `json:"Status"`
	} `json:"State"`
	Config struct {
		Image  string            `json:"Image"`
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
	HostConfig struct {
		Memory   int64 `json:"Memory"`
		NanoCPUs int64 `json:"NanoCpus"`
	} `json:"HostConfig"`
	NetworkSettings struct {
		Networks map[string]struct {
			IPAddress string `json:"IPAddress"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

func (c *Client) Inspect(ctx context.Context, ids ...string) ([]Container, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	out, err := c.Run(ctx, append([]string{"inspect"}, ids...)...)
	if err != nil {
		return nil, err
	}
	return ParseInspect(out)
}

func ParseInspect(out []byte) ([]Container, error) {
	var entries []inspectEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("docker inspect: разбор вывода: %w", err)
	}
	res := make([]Container, 0, len(entries))
	for _, e := range entries {
		ct := Container{
			ID:       e.ID,
			Name:     strings.TrimPrefix(e.Name, "/"),
			State:    e.State.Status,
			Image:    e.Config.Image,
			ImageID:  e.Image,
			Labels:   e.Config.Labels,
			Memory:   e.HostConfig.Memory,
			NanoCPUs: e.HostConfig.NanoCPUs,
		}
		// Сеть у dev-контейнеров одна (bridge); берём первый непустой адрес,
		// обходя сети в стабильном порядке.
		names := make([]string, 0, len(e.NetworkSettings.Networks))
		for n := range e.NetworkSettings.Networks {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			if ip := e.NetworkSettings.Networks[n].IPAddress; ip != "" {
				ct.IP = ip
				break
			}
		}
		res = append(res, ct)
	}
	return res, nil
}

// Stat — строка docker stats; ключ карты — короткий ID.
type Stat struct{ CPU, Mem string }

func (c *Client) Stats(ctx context.Context) (map[string]Stat, error) {
	out, err := c.Run(ctx, "stats", "--no-stream", "--format", "{{.ID}}\t{{.CPUPerc}}\t{{.MemUsage}}")
	if err != nil {
		return nil, err
	}
	return ParseStats(out), nil
}

func ParseStats(out []byte) map[string]Stat {
	res := map[string]Stat{}
	for _, l := range lines(out) {
		f := strings.Split(l, "\t")
		if len(f) != 3 {
			continue
		}
		res[f[0]] = Stat{CPU: f[1], Mem: strings.ReplaceAll(f[2], " ", "")}
	}
	return res
}

func (c *Client) ContainerRm(ctx context.Context, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := c.Run(ctx, append([]string{"rm", "-f"}, ids...)...)
	return err
}

func (c *Client) VolumeExists(ctx context.Context, name string) bool {
	_, err := c.Run(ctx, "volume", "inspect", "--format", "{{.Name}}", name)
	return err == nil
}

// VolumeCreate создаёт том с метками. Существующий не трогает: docker
// отказывается менять метки у живого тома, а нам важно только, что он есть.
func (c *Client) VolumeCreate(ctx context.Context, name string, labels map[string]string) error {
	if c.VolumeExists(ctx, name) {
		return nil
	}
	args := []string{"volume", "create"}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "--label", k+"="+labels[k])
	}
	_, err := c.Run(ctx, append(args, name)...)
	return err
}

func (c *Client) VolumeNames(ctx context.Context, filters ...string) ([]string, error) {
	args := []string{"volume", "ls", "-q"}
	for _, f := range filters {
		args = append(args, "--filter", f)
	}
	out, err := c.Run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return lines(out), nil
}

func (c *Client) VolumeRm(ctx context.Context, names ...string) error {
	if len(names) == 0 {
		return nil
	}
	_, err := c.Run(ctx, append([]string{"volume", "rm"}, names...)...)
	return err
}

type Image struct{ ID, Repo, Tag string }

func (c *Client) Images(ctx context.Context) ([]Image, error) {
	out, err := c.Run(ctx, "images", "--no-trunc", "--format", "{{.ID}}\t{{.Repository}}\t{{.Tag}}")
	if err != nil {
		return nil, err
	}
	return ParseImages(out), nil
}

func ParseImages(out []byte) []Image {
	var res []Image
	for _, l := range lines(out) {
		f := strings.Split(l, "\t")
		if len(f) != 3 {
			continue
		}
		res = append(res, Image{ID: f[0], Repo: f[1], Tag: f[2]})
	}
	return res
}

func (c *Client) ImageExists(ctx context.Context, ref string) bool {
	_, err := c.Run(ctx, "image", "inspect", "--format", "{{.Id}}", ref)
	return err == nil
}

func (c *Client) ImageRm(ctx context.Context, refs ...string) error {
	if len(refs) == 0 {
		return nil
	}
	_, err := c.Run(ctx, append([]string{"rmi"}, refs...)...)
	return err
}

// UsedImageIDs — полные ID образов, на которые ссылается хоть один контейнер.
// Именно полные: имя из docker ps усечено и для сверки не годится.
func (c *Client) UsedImageIDs(ctx context.Context) (map[string]bool, error) {
	ids, err := c.ContainerIDs(ctx, true)
	if err != nil {
		return nil, err
	}
	used := map[string]bool{}
	if len(ids) == 0 {
		return used, nil
	}
	out, err := c.Run(ctx, append([]string{"inspect", "--format", "{{.Image}}"}, ids...)...)
	if err != nil {
		return nil, err
	}
	for _, l := range lines(out) {
		used[l] = true
	}
	return used, nil
}

func lines(out []byte) []string {
	var res []string
	for _, l := range strings.Split(string(out), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			res = append(res, l)
		}
	}
	return res
}
