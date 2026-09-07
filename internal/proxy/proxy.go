// Package proxy — ProxyCommand для ssh: находит контейнер по имени проекта
// и гонит stdin/stdout в его sshd. Портов на хосте нет: контейнер доступен
// по своему IP в сети docker, а клиенту хватает одной записи Host *.dev.
package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/Takexito/devcontainer-features/internal/docker"
)

const sshdPort = "2222"

// Normalize снимает суффикс .dev: клиент передаёт %h целиком.
func Normalize(host string) string { return strings.TrimSuffix(host, ".dev") }

func Serve(ctx context.Context, cli *docker.Client, dir, name string) error {
	ids, err := cli.ContainerIDs(ctx, false, docker.FolderFilter(dir))
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return fmt.Errorf("контейнер для %q не запущен\nподними его: devc up %s", name, name)
	}
	cts, err := cli.Inspect(ctx, ids[0])
	if err != nil {
		return err
	}
	if len(cts) == 0 || cts[0].IP == "" {
		return fmt.Errorf("у контейнера %q нет IP", name)
	}
	conn, err := net.Dial("tcp", net.JoinHostPort(cts[0].IP, sshdPort))
	if err != nil {
		return fmt.Errorf("sshd в %q: %w", name, err)
	}
	defer func() { _ = conn.Close() }()
	go func() {
		_, _ = io.Copy(conn, os.Stdin)
		// Клиент закрыл stdin — закрываем свою половину, sshd увидит EOF.
		if tc, ok := conn.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}()
	_, err = io.Copy(os.Stdout, conn)
	return err
}
