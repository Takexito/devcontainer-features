// Package devcontainer запускает devcontainer CLI. Контракт вывода: stdout —
// одна JSON-строка с результатом, всё остальное CLI пишет в stderr.
package devcontainer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type UpOptions struct {
	WorkspaceDir   string
	RemoveExisting bool
	DotfilesRepo   string
	Quiet          bool // stderr CLI показывать только при ошибке
}

type UpResult struct {
	Outcome               string `json:"outcome"`
	ContainerID           string `json:"containerId"`
	RemoteUser            string `json:"remoteUser"`
	RemoteWorkspaceFolder string `json:"remoteWorkspaceFolder"`
	Message               string `json:"message"`
	Description           string `json:"description"`
}

// Up поднимает контейнер. Дотфайлы — флаг CLI, а не свойство devcontainer.json:
// по спецификации это личная настройка человека, а не проекта.
func Up(ctx context.Context, o UpOptions) (UpResult, error) {
	args := []string{"up", "--workspace-folder", o.WorkspaceDir,
		"--dotfiles-repository", o.DotfilesRepo,
		"--dotfiles-install-command", "install.sh",
		"--dotfiles-target-path", "/home/dev/dotfiles",
	}
	if o.RemoveExisting {
		args = append(args, "--remove-existing-container")
	}
	cmd := exec.CommandContext(ctx, "devcontainer", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	if o.Quiet {
		cmd.Stderr = &stderr
	} else {
		cmd.Stderr = os.Stderr
	}
	runErr := cmd.Run()
	res, parseErr := ParseResult(stdout.Bytes())
	if runErr == nil && parseErr == nil && res.Outcome == "success" {
		return res, nil
	}
	if o.Quiet {
		_, _ = os.Stderr.Write(stderr.Bytes())
	}
	var notFound *exec.Error
	if errors.As(runErr, &notFound) {
		return res, fmt.Errorf("devcontainer CLI не найден в PATH: %w", runErr)
	}
	switch {
	case parseErr == nil && res.Message != "":
		msg := res.Message
		if res.Description != "" {
			msg += ": " + res.Description
		}
		return res, fmt.Errorf("devcontainer up: %s", msg)
	case runErr != nil:
		return res, fmt.Errorf("devcontainer up: %w", runErr)
	default:
		return res, fmt.Errorf("devcontainer up: %w", parseErr)
	}
}

// ParseResult берёт последнюю непустую строку stdout — там JSON результата.
func ParseResult(out []byte) (UpResult, error) {
	var res UpResult
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	if last == "" {
		return res, errors.New("пустой вывод")
	}
	if err := json.Unmarshal([]byte(last), &res); err != nil {
		return res, fmt.Errorf("не разобрать результат %q: %w", last, err)
	}
	return res, nil
}

type BuildOptions struct {
	WorkspaceDir string
	ConfigPath   string
	ImageNames   []string
	NoCache      bool
}

// Build собирает образ без --push: с ним CLI уходит в buildx --push и не
// оставляет образ локально, а следующий образ строится поверх этого тега.
func Build(ctx context.Context, o BuildOptions) error {
	args := []string{"build", "--workspace-folder", o.WorkspaceDir, "--config", o.ConfigPath}
	for _, n := range o.ImageNames {
		args = append(args, "--image-name", n)
	}
	if o.NoCache {
		args = append(args, "--no-cache")
	}
	cmd := exec.CommandContext(ctx, "devcontainer", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("devcontainer build: %w", err)
	}
	return nil
}
