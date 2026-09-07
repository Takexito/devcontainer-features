package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Takexito/devcontainer-features/internal/config"
	"github.com/Takexito/devcontainer-features/internal/devcontainer"
	"github.com/Takexito/devcontainer-features/internal/docker"
)

// runBuild собирает образы из images/<стек>. Порядок важен: web, rust и
// android строятся поверх локального тега base-dev.
func runBuild(ctx context.Context, cfg *config.Config, args []string) error {
	fs := newFlagSet("build", "devc build <base|web|rust|android|all> [--push] [--no-cache]")
	push := fs.Bool("push", false, "отправить оба тега в реестр после сборки")
	noCache := fs.Bool("no-cache", false, "собирать без кеша")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		fs.Usage()
		return errors.New("укажи образ или all")
	}
	var targets []string
	switch pos[0] {
	case "all":
		targets = config.Stacks
	default:
		for _, s := range config.Stacks {
			if s == pos[0] {
				targets = []string{s}
			}
		}
		if targets == nil {
			return fmt.Errorf("неизвестный образ %q, есть: %s, all", pos[0], strings.Join(config.Stacks, ", "))
		}
	}
	if *push {
		if err := checkRegistryAuth(cfg.Registry); err != nil {
			return err
		}
	}
	cli := docker.New()
	for _, t := range targets {
		if t != "base" && !cli.ImageExists(ctx, cfg.Images["base"]) {
			return fmt.Errorf("образа %s нет локально — сначала devc build base", cfg.Images["base"])
		}
		dir := filepath.Join(cfg.RepoDir, "images", t)
		cfgPath := filepath.Join(dir, "devcontainer.json")
		if _, err := os.Stat(cfgPath); err != nil {
			return fmt.Errorf("нет конфига образа: %w", err)
		}
		repo, _ := splitRef(cfg.Images[t])
		names := []string{repo + ":1", repo + ":latest"}
		fmt.Fprintf(os.Stderr, ">> собираю %s\n", names[0])
		if err := devcontainer.Build(ctx, devcontainer.BuildOptions{
			WorkspaceDir: dir, ConfigPath: cfgPath, ImageNames: names, NoCache: *noCache,
		}); err != nil {
			return err
		}
		if !*push {
			continue
		}
		for _, n := range names {
			fmt.Fprintf(os.Stderr, ">> отправляю %s\n", n)
			if err := cli.Stream(ctx, "push", n); err != nil {
				return err
			}
		}
	}
	return nil
}

// splitRef делит ссылку на репозиторий и тег; двоеточие порта в хосте
// стоит до последнего «/», поэтому ищем после него.
func splitRef(ref string) (repo, tag string) {
	slash := strings.LastIndex(ref, "/")
	if i := strings.LastIndex(ref, ":"); i > slash {
		return ref[:i], ref[i+1:]
	}
	return ref, ""
}

// checkRegistryAuth смотрит, есть ли вход в реестр, до долгой сборки, а не
// после: push без авторизации падает последним шагом.
func checkRegistryAuth(registry string) error {
	host, _, _ := strings.Cut(registry, "/")
	hint := fmt.Sprintf("нет входа в %s: gh auth token | docker login %s -u Takexito --password-stdin", host, host)
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(home, ".docker", "config.json"))
	if err != nil {
		return errors.New(hint)
	}
	var cfg struct {
		Auths       map[string]json.RawMessage `json:"auths"`
		CredHelpers map[string]string          `json:"credHelpers"`
		CredsStore  string                     `json:"credsStore"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("~/.docker/config.json: %w", err)
	}
	if _, ok := cfg.Auths[host]; ok {
		return nil
	}
	if _, ok := cfg.CredHelpers[host]; ok || cfg.CredsStore != "" {
		return nil
	}
	return errors.New(hint)
}
