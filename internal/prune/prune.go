// Package prune — уборка образов vsc-*, которые devcontainer CLI оставляет
// после каждого up и каждого прогона тестов фич.
package prune

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Takexito/devcontainer-features/internal/docker"
)

type Candidate struct{ Ref, ID string }

// Plan выбирает образы vsc-*, на которые не ссылается ни один контейнер.
// Свои образы из реестра (protect — префикс репозитория) не трогаем никогда:
// они контейнерами напрямую не используются, а качать заново долго.
// Удаляем по repo:tag, а не по ID: один ID может носить несколько тегов.
func Plan(images []docker.Image, used map[string]bool, protect string) []Candidate {
	protected := map[string]bool{}
	for _, im := range images {
		if strings.HasPrefix(im.Repo, protect) {
			protected[im.ID] = true
		}
	}
	var out []Candidate
	for _, im := range images {
		if !strings.HasPrefix(im.Repo, "vsc-") || used[im.ID] || protected[im.ID] {
			continue
		}
		out = append(out, Candidate{Ref: im.Repo + ":" + im.Tag, ID: im.ID})
	}
	return out
}

// DiskUsage — реальные цифры с диска. docker system df здесь врёт в обе
// стороны: хранилище раздвоено между /var/lib/docker и /var/lib/containerd.
func DiskUsage(ctx context.Context) string {
	out, err := exec.CommandContext(ctx, "sudo", "-n", "du", "-sh", "/var/lib/docker", "/var/lib/containerd").Output()
	if err != nil {
		return fmt.Sprintf("(недоступно: %v)", err)
	}
	return strings.TrimRight(string(out), "\n")
}
