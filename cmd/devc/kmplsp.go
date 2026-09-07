package main

import (
	"context"
	"errors"

	"github.com/Takexito/devcontainer-features/internal/config"
	"github.com/Takexito/devcontainer-features/internal/docker"
	"github.com/Takexito/devcontainer-features/internal/kmplsp"
	"github.com/Takexito/devcontainer-features/internal/ui"
)

func runKmplsp(ctx context.Context, cfg *config.Config, args []string) error {
	fs := newFlagSet("kmplsp", "devc kmplsp [-y]")
	yes := fs.Bool("y", false, "не спрашивать подтверждения")
	if _, err := parse(fs, args); err != nil {
		return err
	}
	// Том общий: пересборка подменяет сервер всем android-проектам разом.
	if !*yes && !ui.Confirm("Пересобрать общий том "+kmplsp.Volume+"?") {
		return errors.New("отменено")
	}
	return kmplsp.Rebuild(ctx, docker.New(), kmplsp.Options{Src: cfg.KmplspSrc, Image: cfg.Images["android"]})
}
