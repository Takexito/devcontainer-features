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
	MobileMCP    string            // адрес mobile-mcp на хосте; пусто — не прописывать
	MobileMCPMac string            // адрес mobile-mcp на Mac по tailnet; пусто — не прописывать
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
	// Шлюз docker0: сервис слушает 172.17.0.1:8971, ufw пускает туда только
	// docker0. Mac по умолчанию не прописан — его tailnet-адрес знает хозяин.
	c.MobileMCP = env("DEVC_MOBILE_MCP_URL", "http://172.17.0.1:8971/mcp")
	c.MobileMCPMac = env("DEVC_MOBILE_MCP_MAC_URL", "")
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
