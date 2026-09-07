package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/Takexito/devcontainer-features/internal/config"
	"github.com/Takexito/devcontainer-features/internal/docker"
	"github.com/Takexito/devcontainer-features/internal/prune"
	"github.com/Takexito/devcontainer-features/internal/ui"
)

func runPrune(ctx context.Context, cfg *config.Config, args []string) error {
	fs := newFlagSet("prune", "devc prune [--builder] [-y]")
	builder := fs.Bool("builder", false, "заодно снести кеш сборки (docker builder prune -a)")
	yes := fs.Bool("y", false, "не спрашивать подтверждения")
	if _, err := parse(fs, args); err != nil {
		return err
	}
	cli := docker.New()
	before := prune.DiskUsage(ctx)
	images, err := cli.Images(ctx)
	if err != nil {
		return err
	}
	used, err := cli.UsedImageIDs(ctx)
	if err != nil {
		return err
	}
	cands := prune.Plan(images, used, cfg.Registry+"/")
	fmt.Printf("Диск сейчас:\n%s\n", before)
	if len(cands) == 0 && !*builder {
		fmt.Println("неиспользуемых vsc-* образов нет")
		return nil
	}
	fmt.Println("Будет удалено:")
	for _, c := range cands {
		fmt.Printf("  образ  %s\n", c.Ref)
	}
	if *builder {
		fmt.Println("  кеш сборки (docker builder prune -a)")
	}
	if !*yes && !ui.Confirm("Продолжить?") {
		return errors.New("отменено")
	}
	refs := make([]string, 0, len(cands))
	for _, c := range cands {
		refs = append(refs, c.Ref)
	}
	if err := cli.ImageRm(ctx, refs...); err != nil {
		return err
	}
	if *builder {
		if err := cli.Stream(ctx, "builder", "prune", "-a", "-f"); err != nil {
			return err
		}
	}
	fmt.Printf("Диск после:\n%s\n", prune.DiskUsage(ctx))
	return nil
}
