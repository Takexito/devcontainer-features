package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Takexito/devcontainer-features/internal/config"
	"github.com/Takexito/devcontainer-features/internal/docker"
	"github.com/Takexito/devcontainer-features/internal/stack"
	"github.com/Takexito/devcontainer-features/internal/ui"
)

func runLs(ctx context.Context, cfg *config.Config, args []string) error {
	fs := newFlagSet("ls", "devc ls [-a] [--no-stats]")
	all := fs.Bool("a", false, "включая остановленные")
	noStats := fs.Bool("no-stats", false, "без CPU и памяти (docker stats медленный)")
	if _, err := parse(fs, args); err != nil {
		return err
	}
	cli := docker.New()
	ids, err := cli.ContainerIDs(ctx, *all, "label="+docker.LabelFolder)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		fmt.Println("dev-контейнеров нет")
	} else {
		cts, err := cli.Inspect(ctx, ids...)
		if err != nil {
			return err
		}
		stats := map[string]docker.Stat{}
		if !*noStats {
			// Один вызов на все контейнеры: docker stats медленный.
			if stats, err = cli.Stats(ctx); err != nil {
				fmt.Fprintln(os.Stderr, ">>", err)
			}
		}
		sort.Slice(cts, func(i, j int) bool { return cts[i].Project() < cts[j].Project() })
		rows := [][]string{{"ПРОЕКТ", "СТЕК", "СОСТОЯНИЕ", "АДРЕС", "ЛИМИТ", "CPU", "ПАМЯТЬ"}}
		for _, c := range cts {
			st := stats[c.ShortID()]
			rows = append(rows, []string{
				c.Project(), stackOf(c), c.State, dash(c.IP), dash(memLimit(c.Memory)), dash(st.CPU), dash(st.Mem),
			})
		}
		if err := ui.Table(os.Stdout, rows); err != nil {
			return err
		}
	}
	vols, err := cli.VolumeNames(ctx)
	if err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Общие тома (не сносятся вместе с проектом):")
	n := 0
	for _, v := range vols {
		// Фильтр docker по имени ищет подстроку, поэтому префикс проверяем сами.
		if strings.HasPrefix(v, "dev-") {
			fmt.Println("  " + v)
			n++
		}
	}
	if n == 0 {
		fmt.Println("  нет")
	}
	return nil
}

// stackOf читает стек из метки контейнера; у конфигов, написанных руками,
// маркера нет.
func stackOf(c docker.Container) string {
	if m, ok := stack.FromMetadata(c.Labels["devcontainer.metadata"]); ok {
		return m.Stack
	}
	return "custom"
}

func memLimit(b int64) string {
	switch {
	case b <= 0:
		return ""
	case b%(1<<30) == 0:
		return fmt.Sprintf("%dg", b>>30)
	case b%(1<<20) == 0:
		return fmt.Sprintf("%dm", b>>20)
	}
	return fmt.Sprintf("%d", b)
}

func dash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
