// Package config читает настройки devc из переменных окружения.
// Файла конфигурации нет намеренно: настроек мало, а переменную можно
// переопределить на один запуск, ничего не правя.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Stacks — стеки в порядке сборки образов: base первым, остальные от него.
var Stacks = []string{"base", "web", "rust", "android"}

type Config struct {
	ProjectsDir  string
	DotfilesRepo string
	Registry     string
	RepoDir      string
	KmplspSrc    string
	Images       map[string]string // стек → образ
}

func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("домашний каталог: %w", err)
	}
	c := &Config{Images: map[string]string{}}
	c.ProjectsDir = env("DEVC_PROJECTS_DIR", filepath.Join(home, "projects"))
	c.DotfilesRepo = env("DEVC_DOTFILES_REPO", "https://github.com/Takexito/dotfiles")
	c.Registry = env("DEVC_REGISTRY", "ghcr.io/takexito")
	c.RepoDir = env("DEVC_REPO_DIR", filepath.Join(c.ProjectsDir, "devcontainer-features"))
	c.KmplspSrc = env("DEVC_KMPLSP_SRC", filepath.Join(c.ProjectsDir, "kotlin-lsp-kmp"))
	for _, s := range Stacks {
		c.Images[s] = env("DEVC_IMAGE_"+strings.ToUpper(s), c.Registry+"/"+s+"-dev:1")
	}
	return c, nil
}

// ProjectDir — каталог проекта на хосте; внутри контейнера он же /workspaces/<name>.
func (c *Config) ProjectDir(name string) string { return filepath.Join(c.ProjectsDir, name) }

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
