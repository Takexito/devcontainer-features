// devc — dev-контейнеры на build01: поднять проект в контейнере, посмотреть,
// что запущено, снести, собрать образы. Один бинарник вместо россыпи скриптов.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Takexito/devcontainer-features/internal/config"
)

// exitCode — код возврата дочернего процесса, который надо отдать как есть,
// без сообщения (devc exec).
type exitCode int

func (e exitCode) Error() string { return fmt.Sprintf("код возврата %d", int(e)) }

type command func(ctx context.Context, cfg *config.Config, args []string) error

var commands = map[string]command{
	"up":     runUp,
	"ls":     runLs,
	"rm":     runRm,
	"exec":   runExec,
	"proxy":  runProxy,
	"build":  runBuild,
	"prune":  runPrune,
	"kmplsp": runKmplsp,
}

const usageText = `devc — dev-контейнеры на build01

  devc up <name> [--stack base|web|rust|android] [--clone URL | --init]
                 [--mem 8g] [--cpus 6] [--rebuild] [--force] [-q]
  devc ls [-a] [--no-stats]
  devc rm <name> [-y]
  devc exec <name> [cmd ...]
  devc proxy <name>
  devc build <base|web|rust|android|all> [--push] [--no-cache]
  devc prune [--builder] [-y]
  devc kmplsp [-y]

Подключение с клиента: ssh <name>.dev — при одной записи в ssh-конфиге:
  Host *.dev
      User dev
      ProxyCommand ssh vps devproxy %h

Переопределения: DEVC_PROJECTS_DIR, DEVC_DOTFILES_REPO, DEVC_REGISTRY,
DEVC_IMAGE_<STACK>, DEVC_REPO_DIR, DEVC_KMPLSP_SRC.
`

func main() {
	args := os.Args[1:]
	// devproxy — симлинк на devc: клиентский ssh-конфиг зовёт его по имени.
	if filepath.Base(os.Args[0]) == "devproxy" {
		args = append([]string{"proxy"}, args...)
	}
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(2)
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Print(usageText)
		return
	}
	run, ok := commands[args[0]]
	if !ok {
		fmt.Fprintf(os.Stderr, "devc: неизвестная команда %q\n\n%s", args[0], usageText)
		os.Exit(2)
	}
	cfg, err := config.Load()
	if err == nil {
		err = run(context.Background(), cfg, args[1:])
	}
	if err == nil {
		return
	}
	var code exitCode
	if errors.As(err, &code) {
		os.Exit(int(code))
	}
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(2)
	}
	fmt.Fprintln(os.Stderr, "devc:", err)
	os.Exit(1)
}
