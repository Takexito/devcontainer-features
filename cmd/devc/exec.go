package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/Takexito/devcontainer-features/internal/config"
	"github.com/Takexito/devcontainer-features/internal/docker"
	"github.com/Takexito/devcontainer-features/internal/project"
)

// runExec заходит в контейнер с хоста. Всё после имени — команда, флаги
// не разбираем, чтобы «devc exec app ls -la» ушло в контейнер как есть.
func runExec(ctx context.Context, cfg *config.Config, args []string) error {
	if len(args) == 0 {
		return errors.New("использование: devc exec <name> [cmd ...]")
	}
	name, cmdArgs := args[0], args[1:]
	if len(cmdArgs) > 0 && cmdArgs[0] == "--" {
		cmdArgs = cmdArgs[1:]
	}
	if err := project.ValidateName(name); err != nil {
		return err
	}
	cli := docker.New()
	ids, err := cli.ContainerIDs(ctx, false, docker.FolderFilter(cfg.ProjectDir(name)))
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return fmt.Errorf("контейнер для %q не запущен\nподними его: devc up %s", name, name)
	}
	dargs := []string{"exec", "-u", "dev", "-w", "/workspaces/" + name}
	if isTerminal(os.Stdin) {
		dargs = append(dargs, "-it")
	} else {
		dargs = append(dargs, "-i")
	}
	dargs = append(dargs, ids[0])
	// Login-шелл: /etc/profile.d подхватывает шимы mise и restore-env.
	if len(cmdArgs) == 0 {
		dargs = append(dargs, "bash", "-l")
	} else {
		dargs = append(dargs, "bash", "-lc", `exec "$@"`, "bash")
		dargs = append(dargs, cmdArgs...)
	}
	err = cli.Stream(ctx, dargs...)
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return exitCode(ee.ExitCode())
	}
	return err
}

// isTerminal спрашивает ядро, а не смотрит на тип файла: /dev/null тоже
// символьное устройство, и docker exec -t с ним падает.
func isTerminal(f *os.File) bool {
	var t syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TCGETS, uintptr(unsafe.Pointer(&t)))
	return errno == 0
}
