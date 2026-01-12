#!/bin/sh
set -eu

REPO_OWNER="NicoOrz"
REPO_NAME="promethues-DingTalk-Hook"
PROJECT_NAME="promethues-DingTalk-Hook"
BINARY_NAME="prometheus-dingtalk-hook"

SERVICE_NAME="prometheus-dingtalk-hook"

INSTALL_BIN="${INSTALL_BIN:-/usr/local/bin/${BINARY_NAME}}"
CONFIG_DIR="${CONFIG_DIR:-/etc/prometheus-dingtalk-hook}"
CONFIG_FILE="${CONFIG_FILE:-${CONFIG_DIR}/config.yml}"
TEMPLATES_DIR="${TEMPLATES_DIR:-${CONFIG_DIR}/templates}"

DATA_DIR="${DATA_DIR:-/var/lib/prometheus-dingtalk-hook}"

RUN_USER="${RUN_USER:-prometheus-dingtalk-hook}"
RUN_GROUP="${RUN_GROUP:-prometheus-dingtalk-hook}"

DRY_RUN="${DRY_RUN:-0}"

say() { printf '%s\n' "$*"; }
die() { say "错误: $*"; exit 1; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "缺少依赖命令: $1"
}

as_root() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
    return
  fi
  if command -v sudo >/dev/null 2>&1; then
    sudo "$@"
    return
  fi
  die "需要 root 权限（请使用 root 运行，或安装 sudo）"
}

detect_os_arch() {
  os="$(uname -s 2>/dev/null || echo unknown)"
  arch="$(uname -m 2>/dev/null || echo unknown)"

  case "$os" in
    Linux) os="Linux" ;;
    *) die "仅支持 Linux 安装为 systemd 服务（当前: $os）" ;;
  esac

  case "$arch" in
    x86_64|amd64) arch="x86_64" ;;
    aarch64|arm64) arch="arm64" ;;
    armv7l|armv7) arch="armv7" ;;
    *) die "不支持的架构: $arch" ;;
  esac

  say "$os" "$arch"
}

github_latest_tag() {
  # Prefer GitHub redirect (no API token needed).
  # https://github.com/<owner>/<repo>/releases/latest -> .../tag/vX.Y.Z
  url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/latest"
  loc="$(curl -fsSLI "$url" | awk 'tolower($1)=="location:"{print $2}' | tail -n 1 | tr -d '\r')"
  if [ -n "$loc" ]; then
    tag="${loc##*/tag/}"
    tag="${tag##*/}"
    if [ -n "$tag" ]; then
      say "$tag"
      return
    fi
  fi

  # Fallback to GitHub API.
  need_cmd sed
  json="$(curl -fsSL "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest")"
  tag="$(printf '%s' "$json" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]\+\)".*/\1/p' | head -n 1)"
  [ -n "$tag" ] || die "无法获取最新版本号（tag）"
  say "$tag"
}

sha256_check() {
  file="$1"
  checksums="$2"

  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$(dirname "$checksums")" && sha256sum -c "$(basename "$checksums")" --status --ignore-missing) >/dev/null 2>&1 || return 1
    return 0
  fi
  if command -v shasum >/dev/null 2>&1; then
    want="$(grep "  $(basename "$file")\$" "$checksums" | awk '{print $1}' | head -n 1)"
    [ -n "$want" ] || return 1
    got="$(shasum -a 256 "$file" | awk '{print $1}')"
    [ "$want" = "$got" ] || return 1
    return 0
  fi
  return 1
}

ensure_user_group() {
  if getent group "$RUN_GROUP" >/dev/null 2>&1; then
    :
  else
    if command -v groupadd >/dev/null 2>&1; then
      as_root groupadd --system "$RUN_GROUP" >/dev/null 2>&1 || true
    fi
  fi

  if id "$RUN_USER" >/dev/null 2>&1; then
    :
  else
    if command -v useradd >/dev/null 2>&1; then
      as_root useradd --system --no-create-home --home-dir "$DATA_DIR" --shell /usr/sbin/nologin --gid "$RUN_GROUP" "$RUN_USER" >/dev/null 2>&1 || true
    elif command -v adduser >/dev/null 2>&1; then
      as_root adduser --system --no-create-home --home "$DATA_DIR" --shell /usr/sbin/nologin --ingroup "$RUN_GROUP" "$RUN_USER" >/dev/null 2>&1 || true
    fi
  fi
}

write_default_config_if_missing() {
  if [ -f "$CONFIG_FILE" ]; then
    return
  fi
  as_root mkdir -p "$CONFIG_DIR"
  as_root sh -c "cat >\"$CONFIG_FILE\" <<'EOF'
server:
  listen: \"0.0.0.0:8080\"
  path: \"/alert\"
  read_timeout: 5s
  write_timeout: 10s
  idle_timeout: 60s
  max_body_bytes: 4194304

auth:
  # 内网可留空；启用后支持：
  # - Authorization: Bearer <token>
  # - X-Token: <token>
  token: \"\"

template:
  # 模板目录（加载 *.tmpl）。留空则使用内置 default 模板。
  # dir: \"${TEMPLATES_DIR}\"
  dir: \"\"

admin:
  enabled: false
  path_prefix: \"/admin\"
  basic_auth:
    username: \"admin\"
    password: \"change-me\"

reload:
  enabled: false
  interval: 2s

dingtalk:
  timeout: 5s
  robots:
    - name: \"default\"
      webhook: \"https://oapi.dingtalk.com/robot/send?access_token=YOUR_ACCESS_TOKEN\"
      secret: \"\"
      msg_type: \"markdown\"
      # 留空：默认使用 Alertmanager 的 summary。
      title: \"\"

  channels:
    - name: \"default\" # 必须存在
      robots: [\"default\"]
      template: \"default\"
      mention:
        at_all: false
        at_mobiles: []
        at_user_ids: []
      mention_rules:
        - name: \"critical->@all\"
          when:
            labels:
              severity: [\"critical\"]
          mention:
            at_all: true

  routes: []
EOF"
  as_root chmod 0640 "$CONFIG_FILE" || true
  as_root chown root:"$RUN_GROUP" "$CONFIG_FILE" || true
}

install_service() {
  need_cmd systemctl
  unit_path="/etc/systemd/system/${SERVICE_NAME}.service"

  as_root mkdir -p "$DATA_DIR" "$CONFIG_DIR"
  ensure_user_group

  as_root chown -R "$RUN_USER":"$RUN_GROUP" "$DATA_DIR" || true

  as_root sh -c "cat >\"$unit_path\" <<EOF
[Unit]
Description=Prometheus DingTalk Hook
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${RUN_USER}
Group=${RUN_GROUP}
ExecStart=${INSTALL_BIN} -config ${CONFIG_FILE}
Restart=on-failure
RestartSec=2s
WorkingDirectory=${DATA_DIR}
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF"

  as_root systemctl daemon-reload
  as_root systemctl enable --now "${SERVICE_NAME}.service"
}

main() {
  need_cmd curl
  need_cmd tar
  need_cmd awk
  need_cmd grep
  need_cmd uname
  need_cmd mktemp

  os_arch="$(detect_os_arch)"
  os="$(printf '%s' "$os_arch" | awk '{print $1}')"
  arch="$(printf '%s' "$os_arch" | awk '{print $2}')"

  tag="$(github_latest_tag)"
  version="${tag#v}"

  archive="${PROJECT_NAME}_${version}_${os}_${arch}.tar.gz"
  base_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${tag}"

  say "将安装 ${BINARY_NAME} 版本: ${tag}"
  say "目标路径: ${INSTALL_BIN}"
  say "配置路径: ${CONFIG_FILE}"

  if [ "$DRY_RUN" = "1" ]; then
    say "DRY_RUN=1：跳过下载与安装"
    exit 0
  fi

  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT

  say "下载: ${archive}"
  curl -fsSL "${base_url}/${archive}" -o "${tmpdir}/${archive}"
  say "下载: checksums.txt"
  curl -fsSL "${base_url}/checksums.txt" -o "${tmpdir}/checksums.txt"

  if ! sha256_check "${tmpdir}/${archive}" "${tmpdir}/checksums.txt"; then
    die "校验失败：checksums.txt 不匹配（需要 sha256sum 或 shasum）"
  fi

  tar -xzf "${tmpdir}/${archive}" -C "$tmpdir"
  [ -f "${tmpdir}/${BINARY_NAME}" ] || die "归档中未找到二进制文件: ${BINARY_NAME}"

  as_root install -m 0755 "${tmpdir}/${BINARY_NAME}" "$INSTALL_BIN"

  write_default_config_if_missing
  install_service

  say ""
  say "✅ 安装完成"
  say ""
  say "下一步配置："
  say "1) 编辑配置: ${CONFIG_FILE}"
  say "2) 重启服务: systemctl restart ${SERVICE_NAME}"
  say "3) 查看日志: journalctl -u ${SERVICE_NAME} -f"
  say ""
  say "提示：如需自定义模板，可创建目录并配置 template.dir："
  say "  - mkdir -p ${TEMPLATES_DIR}"
  say "  - 将 *.tmpl 放入该目录，然后在 ${CONFIG_FILE} 中设置 template.dir"
}

main "$@"

