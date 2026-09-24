package stack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type Options struct {
	Image string
	Mem   string // пусто — DefaultMem стека
	CPUs  string // пусто — DefaultCPUs стека
	// Адреса mobile-mcp. Пусто — сервер в .mcp.json не попадает: иначе проект
	// ловил бы «failed to connect», пока второй конец не поднят.
	MobileMCPURL    string // redroid на хосте, через шлюз docker0
	MobileMCPMacURL string // симуляторы iOS и телефон на Mac, по tailnet
}

// Files — то, что devc up кладёт в проект, плюс тома, которые надо создать
// с метками до запуска (devcontainer CLI создаёт тома без меток).
type Files struct {
	Devcontainer   []byte
	PostCreate     []byte
	Zed            []byte   // nil — стеку не нужен
	MCP            []byte   // .mcp.json; nil — стеку не нужен
	ClaudeServers  []string // имена серверов для enabledMcpjsonServers
	Gitignore      []string // строки, которых не должно не хватать в .gitignore
	ProjectVolumes []string
	Marker         Marker
}

var (
	memRe  = regexp.MustCompile(`^[0-9]+[kmgKMG]?$`)
	cpusRe = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)
)

// Порядок полей — порядок в файле: encoding/json пишет структуру как есть.
type devcontainerJSON struct {
	Name                string            `json:"name"`
	Image               string            `json:"image"`
	Customizations      customizations    `json:"customizations"`
	UpdateRemoteUserUID bool              `json:"updateRemoteUserUID"`
	ContainerEnv        map[string]string `json:"containerEnv"`
	Mounts              []string          `json:"mounts"`
	RunArgs             []string          `json:"runArgs"`
	RemoteUser          string            `json:"remoteUser"`
	PostCreateCommand   string            `json:"postCreateCommand"`
}

type customizations struct {
	Devc Marker `json:"devc"`
}

// Render собирает файлы проекта для стека. Детерминирован: одинаковый вход
// даёт байт в байт одинаковый выход, на этом держится проверка «конфиг
// изменился» в devc up.
func Render(name string, st Stack, o Options) (Files, error) {
	mem, cpus := o.Mem, o.CPUs
	if mem == "" {
		mem = st.DefaultMem
	}
	if cpus == "" {
		cpus = st.DefaultCPUs
	}
	if !memRe.MatchString(mem) {
		return Files{}, fmt.Errorf("--mem %q: ожидается число с суффиксом k/m/g, например 8g", mem)
	}
	if !cpusRe.MatchString(cpus) {
		return Files{}, fmt.Errorf("--cpus %q: ожидается число, например 6 или 1.5", cpus)
	}
	if o.Image == "" {
		return Files{}, fmt.Errorf("стек %s: не задан образ", st.Name)
	}

	vols := volumes(name, st)
	env := map[string]string{
		"CLAUDE_CONFIG_DIR": "/home/dev/.claude",
		"PROJECT_NAME":      name,
	}
	for k, v := range st.Env {
		env[k] = v
	}

	f := Files{Marker: Marker{Stack: st.Name, Generator: Generator, Mem: mem, CPUs: cpus}}
	mounts := make([]string, 0, len(vols)+1)
	for _, v := range vols {
		m := fmt.Sprintf("source=%s,target=%s,type=volume", v.Name, v.Target)
		if v.ReadOnly {
			m += ",readonly"
		}
		mounts = append(mounts, m)
		if !v.Shared {
			f.ProjectVolumes = append(f.ProjectVolumes, v.Name)
		}
	}
	// Второй источник ключей для sshd — authorized_keys хоста: новое
	// устройство достаточно прописать на VPS.
	mounts = append(mounts, "source=${localEnv:HOME}/.ssh/authorized_keys,target=/etc/ssh/authorized_keys.host,type=bind,readonly")

	dc := devcontainerJSON{
		Name:           name,
		Image:          o.Image,
		Customizations: customizations{Devc: f.Marker},
		// uid dev (1001) совпадает с хостовым: ремап не нужен, а без него
		// devcontainer up не строит производный образ vsc-*-uid.
		UpdateRemoteUserUID: false,
		ContainerEnv:        env,
		Mounts:              mounts,
		RunArgs: []string{
			"--name=" + name,
			"--hostname=" + name,
			"--restart=unless-stopped",
			"--memory=" + mem,
			"--memory-swap=" + mem, // равен --memory: контейнер не уходит в своп
			"--cpus=" + cpus,
		},
		RemoteUser:        "dev",
		PostCreateCommand: "bash .devcontainer/post-create.sh",
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(dc); err != nil {
		return Files{}, err
	}
	f.Devcontainer = buf.Bytes()
	f.PostCreate = []byte(postCreate(name, st, vols))
	if st.Zed != "" {
		f.Zed = []byte(st.Zed)
	}
	if st.MobileMCP {
		mcp, names, err := mobileMCP(o)
		if err != nil {
			return Files{}, err
		}
		if mcp != nil {
			f.MCP, f.ClaudeServers = mcp, names
			f.Gitignore = []string{"/.mcp.json", "/.claude/settings.local.json"}
		}
	}
	return f, nil
}

// mcpServer — запись в .mcp.json. Транспорт именно sse, не http: сервер
// mobile-mcp отвечает на GET /mcp кадром «event: endpoint» с sessionId, то
// есть говорит на старом двухэндпоинтном SSE, а не на Streamable HTTP.
type mcpServer struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// mobileMCP — .mcp.json с удалёнными серверами mobile-mcp. Сервер живёт на
// хосте (node и adb там уже есть, из android-образа node убран намеренно), а
// контейнер ходит к нему по http через шлюз docker0.
func mobileMCP(o Options) ([]byte, []string, error) {
	servers := map[string]mcpServer{}
	var names []string
	if o.MobileMCPURL != "" {
		servers["mobile"] = mcpServer{Type: "sse", URL: o.MobileMCPURL}
		names = append(names, "mobile")
	}
	if o.MobileMCPMacURL != "" {
		servers["mobile-mac"] = mcpServer{Type: "sse", URL: o.MobileMCPMacURL}
		names = append(names, "mobile-mac")
	}
	if len(servers) == 0 {
		return nil, nil, nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	// encoding/json сортирует ключи map, вывод детерминирован.
	if err := enc.Encode(struct {
		MCPServers map[string]mcpServer `json:"mcpServers"`
	}{servers}); err != nil {
		return nil, nil, err
	}
	sort.Strings(names)
	return buf.Bytes(), names, nil
}

// volumes — тома в порядке монтирования: проектные (ssh, mise, стековые),
// затем общие (стековые, авторизации агентов, gh и glab).
func volumes(name string, st Stack) []Volume {
	res := []Volume{
		{Name: name + "-ssh", Target: "/home/dev/.ssh"},
		// Тулчейны, которые mise ставит по mise.toml проекта, переживают
		// пересборку контейнера.
		{Name: name + "-mise", Target: "/home/dev/.local/share/mise"},
	}
	var shared []Volume
	for _, v := range st.Volumes {
		if v.Shared {
			shared = append(shared, v)
			continue
		}
		res = append(res, Volume{Name: name + "-" + v.Name, Target: v.Target, ReadOnly: v.ReadOnly})
	}
	res = append(res, shared...)
	// Авторизации агентов, gh и glab общие: логин один раз на все проекты.
	res = append(res,
		Volume{Name: "dev-claude", Target: "/home/dev/.claude", Shared: true},
		Volume{Name: "dev-codex", Target: "/home/dev/.codex", Shared: true},
		Volume{Name: "dev-gh", Target: "/home/dev/.config/gh", Shared: true},
		Volume{Name: "dev-glab", Target: "/home/dev/.config/glab-cli", Shared: true},
	)
	return res
}

const postCreateHead = `set -euo pipefail
WS="/workspaces/${PROJECT_NAME:?}"

# Свежий том docker принадлежит root. Рекурсивно чиним только чужие каталоги:
# общий cargo-registry или ~/.claude обходить на каждом старте незачем.
own() {
  for d in "$@"; do
    [ -e "$d" ] || continue
    [ "$(stat -c %u "$d")" = "$(id -u)" ] || sudo chown -R dev:dev "$d"
  done
}
`

const postCreateEnv = `# Недостающих родителей точек монтирования docker тоже создаёт от root.
sudo chown dev:dev "$HOME/.local" "$HOME/.local/share" "$HOME/.config" 2>/dev/null || true
chmod 700 "$HOME/.ssh" 2>/dev/null || true

# sshd не наследует окружение контейнера: ssh-сессии берут PATH из
# /etc/environment, который devcontainer CLI заполняет до этого хука, а sshd
# читает через PAM. Добавляем туда бинарники пользователя и шимы mise.
if ! grep -q 'mise/shims' /etc/environment; then
  sudo sed -i 's|^PATH="|PATH="/home/dev/.local/bin:/home/dev/.local/share/mise/shims:|' /etc/environment
fi

# JDK из mise.toml проекта. Шимам хватает java в PATH, но JAVA_HOME они не
# выставляют, а Gradle-обвязки и IDE-серверы ищут JDK именно по нему. Ставим
# JDK сразу, а не при первом вызове, иначе JAVA_HOME указывать некуда.
if [ -n "$(cd "$WS" && mise current java 2>/dev/null)" ]; then
  (cd "$WS" && mise install -q java)
  JH=$(cd "$WS" && mise where java)
  sudo sed -i '/^JAVA_HOME=/d' /etc/environment
  echo "JAVA_HOME=\"$JH\"" | sudo tee -a /etc/environment >/dev/null
  echo ">> JAVA_HOME=$JH (из mise.toml)"
fi
`

const postCreateTail = `
# SSH приземляется в домашний каталог, а проект примонтирован в /workspaces.
ln -sfn "$WS" "$HOME/$PROJECT_NAME"

# Ключ хоста храним в томе проекта. Иначе при каждой пересборке он меняется,
# и клиент отказывается подключаться, приняв это за подмену.
KEYDIR="$HOME/.ssh/hostkeys"
mkdir -p "$KEYDIR"
if [ ! -f "$KEYDIR/ssh_host_ed25519_key" ]; then
  ssh-keygen -q -t ed25519 -N '' -C "$PROJECT_NAME" -f "$KEYDIR/ssh_host_ed25519_key"
fi
sudo chown root:root "$KEYDIR"/ssh_host_ed25519_key*
sudo chmod 600 "$KEYDIR/ssh_host_ed25519_key"

# Фича sshd оставляет включёнными пароли и root-логин; паролей нет, но
# выключаем явно. Второй источник ключей — authorized_keys хоста: новое
# устройство достаточно прописать на VPS, в контейнер дублировать не нужно.
{
  printf 'PasswordAuthentication no\nPermitRootLogin no\nKbdInteractiveAuthentication no\n'
  printf 'AuthorizedKeysFile .ssh/authorized_keys /etc/ssh/authorized_keys.host\n'
  printf 'HostKey %s/ssh_host_ed25519_key\n' "$KEYDIR"
} | sudo tee /etc/ssh/sshd_config.d/99-harden.conf >/dev/null
sudo pkill -HUP -x sshd 2>/dev/null || true

echo ">> $(mise --version)"
`

func postCreate(name string, st Stack, vols []Volume) string {
	own := make([]string, 0, len(vols))
	for _, v := range vols {
		if !v.ReadOnly {
			own = append(own, fmt.Sprintf("%q", v.Target))
		}
	}
	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	fmt.Fprintf(&b, "# Сгенерировано devc, стек %s. Не править руками: файл пересоздаётся\n", st.Name)
	fmt.Fprintf(&b, "# командой «devc up %s». Тулчейн уже в образе, здесь только то, что\n", name)
	b.WriteString("# зависит от конкретного контейнера.\n")
	b.WriteString(postCreateHead)
	fmt.Fprintf(&b, "own %s\n", strings.Join(own, " "))
	b.WriteString(postCreateEnv)
	if st.PostCreate != "" {
		b.WriteString("\n")
		b.WriteString(st.PostCreate)
	}
	b.WriteString(postCreateTail)
	for _, r := range st.Report {
		b.WriteString(r)
		b.WriteString("\n")
	}
	return b.String()
}
