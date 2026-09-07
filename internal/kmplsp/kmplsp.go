// Package kmplsp пересобирает общий том с KMP-сборкой Kotlin LSP из
// ~/projects/kotlin-lsp-kmp. Патч прибит к конкретной сборке сервера и
// сверяет хеши JAR-ов, поэтому собираем в том же образе, на котором работают
// контейнеры: версия сервера гарантированно та, под которую собран патч.
package kmplsp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Takexito/devcontainer-features/internal/docker"
)

const (
	Volume       = "dev-kotlin-lsp-kmp"
	gradleVolume = "kotlin-lsp-kmp-gradle"
	// В android-dev нет Node, а установщик патча написан на нём: mise ставит
	// Node в этот том один раз, дальше сборки идут без загрузки.
	miseVolume = "kotlin-lsp-kmp-mise"
	miseDir    = "/home/dev/.local/share/mise"
)

type Options struct {
	Src   string // каталог kotlin-lsp-kmp с install.sh
	Image string // android-dev
}

// Инсталлятор оставляет предыдущую установку для отката, а она весит ~1.1 ГБ.
// Откат не нужен: пересборка занимает секунды, а если патч не встал,
// post-create в проектах сам возьмёт штатный сервер.
const gcScript = `cd /opt/kmp
for old in current.previous-*; do
  [ -e "$old" ] || continue
  echo ">> убираю копию для отката $old"
  rm -rf -- "$old"
done`

func Rebuild(ctx context.Context, cli *docker.Client, o Options) error {
	install := filepath.Join(o.Src, "install.sh")
	if st, err := os.Stat(install); err != nil || st.Mode()&0o111 == 0 {
		return fmt.Errorf("нет исполняемого %s", install)
	}
	for _, v := range []string{Volume, gradleVolume, miseVolume} {
		if _, err := cli.Run(ctx, "volume", "create", v); err != nil {
			return err
		}
	}
	// Свежий том принадлежит root, а собираем от dev — иначе артефакты
	// приедут с чужим владельцем.
	if err := cli.Stream(ctx, "run", "--rm", "--user", "root",
		"-v", Volume+":/opt/kmp", "-v", gradleVolume+":/home/dev/.gradle", "-v", miseVolume+":"+miseDir,
		o.Image, "chown", "1001:1001", "/opt/kmp", "/home/dev/.gradle", miseDir); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, ">> собираю патч и ставлю в том %s\n", Volume)
	if err := cli.Stream(ctx, "run", "--rm", "--user", "dev",
		"-v", o.Src+":/src", "-v", Volume+":/opt/kmp", "-v", gradleVolume+":/home/dev/.gradle", "-v", miseVolume+":"+miseDir,
		"-w", "/src", "-e", "GRADLE_USER_HOME=/home/dev/.gradle",
		o.Image, "bash", "-lc", "mise x node@22 -- /src/install.sh /opt/kotlin-lsp/current /opt/kmp/current"); err != nil {
		return err
	}
	if err := cli.Stream(ctx, "run", "--rm", "--user", "root", "-v", Volume+":/opt/kmp", o.Image, "bash", "-c", gcScript); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Готово. Контейнеры подхватят сборку при следующем post-create (devc up --rebuild),")
	fmt.Fprintln(os.Stderr, "работающим — перезапустить LSP в редакторе (в Zed: restart language server).")
	return cli.Stream(ctx, "run", "--rm", "-v", Volume+":/opt/kmp:ro", o.Image,
		"sh", "-c", "grep ^server= /opt/kmp/current/kmp-manifest.properties; du -sh /opt/kmp")
}
