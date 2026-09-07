// Package stack описывает стеки dev-контейнеров: образ, окружение, тома и
// фрагменты post-create. Всё, чем rust-контейнер отличается от android,
// лежит здесь, а команды об этом не знают.
package stack

// Volume — том контейнера. Проектный получает имя «<проект>-<Name>» и
// сносится вместе с проектом; общий (dev-*) переживает любой проект.
type Volume struct {
	Name     string
	Target   string
	Shared   bool
	ReadOnly bool
}

type Stack struct {
	Name        string
	Env         map[string]string
	Volumes     []Volume
	PostCreate  string   // bash-фрагмент после общего блока post-create
	Report      []string // строки в конце post-create, обычно echo ">> ..."
	Zed         string   // содержимое .zed/settings.json; пусто — не нужен
	DefaultMem  string
	DefaultCPUs string
}

// Marker хранится в customizations.devc сгенерированного devcontainer.json:
// по нему devc up без --stack знает, что перегенерировать, а devc ls
// показывает стек, читая метку контейнера.
type Marker struct {
	Stack     string `json:"stack"`
	Generator int    `json:"generator"`
	Mem       string `json:"mem,omitempty"`
	CPUs      string `json:"cpus,omitempty"`
}

// Generator — версия формата генерируемых файлов.
const Generator = 1

var names = []string{"base", "web", "rust", "android"}

// Names — стеки в порядке показа и сборки образов.
func Names() []string { return append([]string(nil), names...) }

func Get(name string) (Stack, bool) {
	st, ok := registry[name]
	return st, ok
}

// Лимиты — потолок против одного разогнавшегося Gradle или rustc на хосте
// с 15 GiB и 8 vCPU, а не резервирование: сумма по контейнерам больше хоста.
var registry = map[string]Stack{
	"base": {
		Name:        "base",
		DefaultMem:  "4g",
		DefaultCPUs: "4",
	},
	"web": {
		Name: "web",
		// corepack при первом вызове yarn спрашивает разрешение на загрузку,
		// а в контейнере спрашивать некого.
		Env: map[string]string{"COREPACK_ENABLE_DOWNLOAD_PROMPT": "0"},
		// Хранилище pnpm в общий том не выносим: том и workspace — разные
		// mount-ы, link() между ними даёт EXDEV, и pnpm сам уходит от такого
		// хранилища в node_modules/.pnpm-store проекта. Кеш npm (cacache)
		// копированием не страдает и безопасен для параллельного доступа.
		Volumes: []Volume{
			{Name: "dev-npm", Target: "/home/dev/.npm", Shared: true},
		},
		Report:      []string{`echo ">> node $(node --version), pnpm $(pnpm --version)"`},
		DefaultMem:  "4g",
		DefaultCPUs: "4",
	},
	"rust": {
		Name: "rust",
		Env:  map[string]string{"CARGO_TARGET_DIR": "/home/dev/target"},
		Volumes: []Volume{
			// Артефакты сборки свои у проекта: cargo разных проектов иначе
			// дерутся за блокировку каталога.
			{Name: "target", Target: "/home/dev/target"},
			// Реестр crates общий: cargo держит на нём файловые блокировки,
			// параллельные сборки его не портят.
			{Name: "dev-cargo-registry", Target: "/usr/local/cargo/registry", Shared: true},
		},
		Report: []string{
			`echo ">> $(rustc --version), $(cargo nextest --version | head -1)"`,
			`echo ">> $(rust-analyzer --version)"`,
		},
		DefaultMem:  "8g",
		DefaultCPUs: "6",
	},
	"android": {
		Name: "android",
		// Кеши Gradle и konan свои у каждого проекта, иначе демоны дерутся
		// за блокировки.
		Env: map[string]string{
			"ANDROID_USER_HOME": "/home/dev/.android",
			"GRADLE_USER_HOME":  "/home/dev/.gradle",
			"KONAN_DATA_DIR":    "/home/dev/.konan",
		},
		Volumes: []Volume{
			{Name: "gradle", Target: "/home/dev/.gradle"},
			{Name: "konan", Target: "/home/dev/.konan"},
			{Name: "android", Target: "/home/dev/.android"},
			{Name: "dev-kotlin-lsp-kmp", Target: "/opt/kmp", Shared: true, ReadOnly: true},
		},
		PostCreate: androidPostCreate,
		Report: []string{
			`echo ">> $(java -version 2>&1 | head -1)"`,
			`echo ">> Kotlin LSP: $LSP_KIND"`,
		},
		Zed:         androidZed,
		DefaultMem:  "10g",
		DefaultCPUs: "6",
	},
}
