package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Takexito/devcontainer-features/internal/config"
	"github.com/Takexito/devcontainer-features/internal/devcontainer"
	"github.com/Takexito/devcontainer-features/internal/docker"
	"github.com/Takexito/devcontainer-features/internal/project"
	"github.com/Takexito/devcontainer-features/internal/stack"
)

func runUp(ctx context.Context, cfg *config.Config, args []string) error {
	fs := newFlagSet("up", "devc up <name> [--stack base|web|rust|android] [--clone URL | --init] [--mem 8g] [--cpus 6] [--rebuild] [--force] [-q]")
	stackName := fs.String("stack", "", "стек: "+strings.Join(stack.Names(), "|"))
	clone := fs.String("clone", "", "склонировать репозиторий в ~/projects/<name>")
	initRepo := fs.Bool("init", false, "создать каталог и git init, если проект с нуля")
	force := fs.Bool("force", false, "перезаписать devcontainer.json, написанный руками")
	rebuild := fs.Bool("rebuild", false, "пересоздать контейнер (после смены конфига или образа)")
	mem := fs.String("mem", "", "лимит памяти, например 8g (по умолчанию из стека)")
	cpus := fs.String("cpus", "", "лимит CPU, например 6")
	quiet := fs.Bool("q", false, "вывод devcontainer up показывать только при ошибке")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		fs.Usage()
		return errors.New("укажи имя проекта")
	}
	name := pos[0]
	if err := project.ValidateName(name); err != nil {
		return err
	}
	dir := cfg.ProjectDir(name)
	if err := ensureDir(ctx, dir, *clone, *initRepo); err != nil {
		return err
	}
	p, err := project.Load(name, dir)
	if err != nil {
		return err
	}
	cli := docker.New()

	// Конфиг, написанный руками, — территория человека: поднимаем как есть.
	if p.Mode == project.ModeCustom && !*force {
		if *stackName != "" {
			return fmt.Errorf("в %s свой devcontainer.json без маркера devc; --force перезапишет его стеком %s", p.ConfigPath(), *stackName)
		}
		fmt.Fprintf(os.Stderr, ">> %s: конфиг написан руками, поднимаю как есть\n", name)
		return bringUp(ctx, cfg, cli, p, p.Image, *rebuild, *quiet)
	}

	sname := *stackName
	if sname == "" && p.Mode == project.ModeGenerated {
		sname = p.Marker.Stack
	}
	if sname == "" {
		return fmt.Errorf("укажи стек: --stack %s", strings.Join(stack.Names(), "|"))
	}
	st, ok := stack.Get(sname)
	if !ok {
		return fmt.Errorf("неизвестный стек %q, есть: %s", sname, strings.Join(stack.Names(), "|"))
	}
	opts := stack.Options{Image: cfg.Images[sname], Mem: *mem, CPUs: *cpus}
	// Лимиты, заданные раньше, помним, пока стек тот же.
	if p.Mode == project.ModeGenerated && p.Marker.Stack == sname {
		if opts.Mem == "" {
			opts.Mem = p.Marker.Mem
		}
		if opts.CPUs == "" {
			opts.CPUs = p.Marker.CPUs
		}
	}
	files, err := stack.Render(name, st, opts)
	if err != nil {
		return err
	}
	changed := p.Config != nil && !bytes.Equal(p.Config, files.Devcontainer)
	if err := writeFiles(dir, files); err != nil {
		return err
	}
	if p.Mode == project.ModeCustom {
		fmt.Fprintf(os.Stderr, ">> %s: конфиг перезаписан стеком %s\n", name, sname)
	}
	if err := checkNameCollision(ctx, cli, name, dir); err != nil {
		return err
	}
	// Тома создаём сами, с метками: devcontainer CLI создал бы их без меток,
	// и devc rm пришлось бы угадывать по префиксу.
	labels := map[string]string{"devc.project": name, "devc.stack": sname}
	for _, v := range files.ProjectVolumes {
		if err := cli.VolumeCreate(ctx, v, labels); err != nil {
			return err
		}
	}
	if changed && !*rebuild {
		if ids, _ := cli.ContainerIDs(ctx, true, docker.FolderFilter(dir)); len(ids) > 0 {
			fmt.Fprintf(os.Stderr, ">> конфиг изменился, а контейнер уже есть: применится после devc up %s --rebuild\n", name)
		}
	}
	return bringUp(ctx, cfg, cli, p, opts.Image, *rebuild, *quiet)
}

func ensureDir(ctx context.Context, dir, cloneURL string, initRepo bool) error {
	_, err := os.Stat(dir)
	exists := err == nil
	switch {
	case cloneURL != "":
		if exists {
			return fmt.Errorf("каталог %s уже есть, --clone некуда", dir)
		}
		fmt.Fprintf(os.Stderr, ">> клонирую %s\n", cloneURL)
		return runGit(ctx, "clone", cloneURL, dir)
	case initRepo:
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return nil
		}
		return runGit(ctx, "-C", dir, "init", "-q")
	case !exists:
		return fmt.Errorf("нет каталога %s: склонируй через --clone URL или начни с нуля через --init", dir)
	}
	return nil
}

func runGit(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w", args[0], err)
	}
	return nil
}

func writeFiles(dir string, f stack.Files) error {
	dc := filepath.Join(dir, ".devcontainer")
	if err := os.MkdirAll(dc, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dc, "devcontainer.json"), f.Devcontainer, 0o644); err != nil {
		return err
	}
	pc := filepath.Join(dc, "post-create.sh")
	if err := os.WriteFile(pc, f.PostCreate, 0o755); err != nil {
		return err
	}
	if err := os.Chmod(pc, 0o755); err != nil {
		return err
	}
	if f.Zed == nil {
		return nil
	}
	// Настройки редактора — территория человека: пишем только если их нет.
	zed := filepath.Join(dir, ".zed", "settings.json")
	if _, err := os.Stat(zed); !errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(zed), 0o755); err != nil {
		return err
	}
	return os.WriteFile(zed, f.Zed, 0o644)
}

// checkNameCollision: контейнер зовётся как проект, и чужой контейнер с тем
// же именем уронил бы docker run непонятной ошибкой.
func checkNameCollision(ctx context.Context, cli *docker.Client, name, dir string) error {
	ids, err := cli.ContainerIDs(ctx, true, "name="+name)
	if err != nil {
		return err
	}
	cts, err := cli.Inspect(ctx, ids...)
	if err != nil {
		return err
	}
	for _, c := range cts {
		if c.Name == name && c.Labels[docker.LabelFolder] != dir {
			return fmt.Errorf("имя контейнера %s уже занято (%s), и это не контейнер проекта %s", name, c.State, dir)
		}
	}
	return nil
}

func bringUp(ctx context.Context, cfg *config.Config, cli *docker.Client, p *project.Project, image string, rebuild, quiet bool) error {
	if image != "" && !cli.ImageExists(ctx, image) {
		fmt.Fprintf(os.Stderr, ">> образа %s нет локально, первый запуск тянет его из реестра — это долго\n", image)
	}
	fmt.Fprintf(os.Stderr, ">> поднимаю контейнер %s\n", p.Name)
	res, err := devcontainer.Up(ctx, devcontainer.UpOptions{
		WorkspaceDir:   p.Dir,
		RemoveExisting: rebuild,
		DotfilesRepo:   cfg.DotfilesRepo,
		Quiet:          quiet,
	})
	if err != nil {
		return err
	}
	id := res.ContainerID
	if len(id) > 12 {
		id = id[:12]
	}
	fmt.Printf("\nГотово: %s (%s)\n", p.Name, id)
	fmt.Printf("  подключиться:   ssh %s.dev\n", p.Name)
	fmt.Printf("  с хоста:        devc exec %s\n", p.Name)
	fmt.Printf("  проект внутри:  /workspaces/%s\n", p.Name)
	fmt.Printf("  снести:         devc rm %s\n", p.Name)
	return nil
}
