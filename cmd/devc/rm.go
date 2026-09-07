package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Takexito/devcontainer-features/internal/config"
	"github.com/Takexito/devcontainer-features/internal/docker"
	"github.com/Takexito/devcontainer-features/internal/project"
	"github.com/Takexito/devcontainer-features/internal/ui"
)

// runRm сносит контейнер и тома проекта. Каталог с кодом не трогает.
// Общие тома dev-* не трогает тоже: на них авторизации агентов и gh.
func runRm(ctx context.Context, cfg *config.Config, args []string) error {
	fs := newFlagSet("rm", "devc rm <name> [-y]")
	yes := fs.Bool("y", false, "не спрашивать подтверждения")
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
	cli := docker.New()

	containers, err := cli.ContainerIDs(ctx, true, docker.FolderFilter(dir))
	if err != nil {
		return err
	}
	volumes, err := projectVolumes(ctx, cli, name, dir)
	if err != nil {
		return err
	}
	if len(containers) == 0 && len(volumes) == 0 {
		fmt.Printf("нечего сносить для %q\n", name)
		return nil
	}
	fmt.Println("Будет удалено:")
	cts, err := cli.Inspect(ctx, containers...)
	if err != nil {
		return err
	}
	for _, c := range cts {
		fmt.Printf("  контейнер  %s (%s)\n", c.Name, c.State)
	}
	for _, v := range volumes {
		fmt.Printf("  том        %s\n", v)
	}
	fmt.Printf("Каталог %s останется на месте.\n", dir)
	if !*yes && !ui.Confirm("Продолжить?") {
		return errors.New("отменено")
	}
	if err := cli.ContainerRm(ctx, containers...); err != nil {
		return err
	}
	if err := cli.VolumeRm(ctx, volumes...); err != nil {
		return err
	}
	fmt.Println("готово")
	return nil
}

// projectVolumes собирает тома проекта из трёх источников: метки devc.project,
// mounts в devcontainer.json (ловит нестандартные префиксы вроде camdict-*)
// и легаси-префикс <name>-. Оставляет только существующие и никогда dev-*.
func projectVolumes(ctx context.Context, cli *docker.Client, name, dir string) ([]string, error) {
	want := map[string]bool{}
	labeled, err := cli.VolumeNames(ctx, "label=devc.project="+name)
	if err != nil {
		return nil, err
	}
	for _, v := range labeled {
		want[v] = true
	}
	if p, err := project.Load(name, dir); err == nil {
		for _, v := range p.VolumeNames() {
			want[v] = true
		}
	} else {
		fmt.Fprintf(os.Stderr, ">> %v — тома из mounts не учтены\n", err)
	}
	all, err := cli.VolumeNames(ctx)
	if err != nil {
		return nil, err
	}
	existing := map[string]bool{}
	for _, v := range all {
		existing[v] = true
		if strings.HasPrefix(v, name+"-") {
			want[v] = true
		}
	}
	var res []string
	for v := range want {
		if existing[v] && !strings.HasPrefix(v, "dev-") {
			res = append(res, v)
		}
	}
	sort.Strings(res)
	return res, nil
}
