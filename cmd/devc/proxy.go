package main

import (
	"context"
	"errors"

	"github.com/Takexito/devcontainer-features/internal/config"
	"github.com/Takexito/devcontainer-features/internal/docker"
	"github.com/Takexito/devcontainer-features/internal/proxy"
)

func runProxy(ctx context.Context, cfg *config.Config, args []string) error {
	if len(args) != 1 {
		return errors.New("использование: devc proxy <name>[.dev]")
	}
	name := proxy.Normalize(args[0])
	return proxy.Serve(ctx, docker.New(), cfg.ProjectDir(name), name)
}
