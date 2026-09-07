// Package project — проект в ~/projects/<name>: каталог, режим конфига
// (сгенерирован devc, написан руками или отсутствует) и тома из mounts.
package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Takexito/devcontainer-features/internal/stack"
)

type Mode int

const (
	ModeNone      Mode = iota // .devcontainer/devcontainer.json нет
	ModeCustom                // есть, но без маркера devc — не трогаем
	ModeGenerated             // сгенерирован devc, можно перегенерировать
)

type Project struct {
	Name   string
	Dir    string
	Mode   Mode
	Marker stack.Marker
	Image  string // image из конфига; пусто, если сборка из Dockerfile
	Mounts []Mount
	Config []byte // сырой devcontainer.json; nil, если файла нет
}

type Mount struct {
	Source   string
	Target   string
	Type     string
	ReadOnly bool
}

var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// ValidateName: имя идёт в имя контейнера, тома и hostname, поэтому только
// [a-z0-9._-]. Имена dev и dev-* заняты общими томами — rm по префиксу
// снёс бы авторизации всех проектов.
func ValidateName(name string) error {
	if !nameRe.MatchString(name) {
		return fmt.Errorf("имя %q: допустимы только [a-z0-9._-], первый символ — буква или цифра", name)
	}
	if name == "dev" || strings.HasPrefix(name, "dev-") {
		return fmt.Errorf("имя %q пересекается с общими томами dev-* — отказываюсь", name)
	}
	return nil
}

func Load(name, dir string) (*Project, error) {
	p := &Project{Name: name, Dir: dir}
	raw, err := os.ReadFile(p.ConfigPath())
	if errors.Is(err, fs.ErrNotExist) {
		return p, nil
	}
	if err != nil {
		return nil, err
	}
	p.Config = raw
	p.Mode = ModeCustom
	var cfg struct {
		Image          string            `json:"image"`
		Mounts         []json.RawMessage `json:"mounts"`
		Customizations struct {
			Devc *stack.Marker `json:"devc"`
		} `json:"customizations"`
	}
	if err := json.Unmarshal(StripJSONC(raw), &cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", p.ConfigPath(), err)
	}
	p.Image = cfg.Image
	if cfg.Customizations.Devc != nil && cfg.Customizations.Devc.Stack != "" {
		p.Mode = ModeGenerated
		p.Marker = *cfg.Customizations.Devc
	}
	for _, m := range cfg.Mounts {
		if mount, ok := parseMount(m); ok {
			p.Mounts = append(p.Mounts, mount)
		}
	}
	return p, nil
}

func (p *Project) ConfigPath() string {
	return filepath.Join(p.Dir, ".devcontainer", "devcontainer.json")
}

// VolumeNames — тома проекта из mounts: type=volume и не общие dev-*.
// Так находятся и тома с нестандартным префиксом вроде camdict-*.
func (p *Project) VolumeNames() []string {
	var res []string
	for _, m := range p.Mounts {
		if m.Type != "volume" || strings.HasPrefix(m.Source, "dev-") || m.Source == "" {
			continue
		}
		res = append(res, m.Source)
	}
	return res
}

// parseMount принимает обе формы спецификации: строку docker --mount и объект.
func parseMount(raw json.RawMessage) (Mount, bool) {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return parseMountString(s)
	}
	var o struct {
		Source string `json:"source"`
		Target string `json:"target"`
		Type   string `json:"type"`
	}
	if json.Unmarshal(raw, &o) != nil || o.Source == "" {
		return Mount{}, false
	}
	if o.Type == "" {
		o.Type = "volume"
	}
	return Mount{Source: o.Source, Target: o.Target, Type: o.Type}, true
}

func parseMountString(s string) (Mount, bool) {
	m := Mount{Type: "volume"}
	for _, kv := range strings.Split(s, ",") {
		k, v, _ := strings.Cut(strings.TrimSpace(kv), "=")
		switch k {
		case "source", "src":
			m.Source = v
		case "target", "dst", "destination":
			m.Target = v
		case "type":
			m.Type = v
		case "readonly", "ro":
			m.ReadOnly = true
		}
	}
	return m, m.Source != ""
}
