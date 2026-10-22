#!/usr/bin/env bash
# 本地 Docker 测试环境管理脚本。
#
#   scripts/docker-local.sh up        生成配置并构建/启动（默认）
#   scripts/docker-local.sh start     启动已有镜像，不重新构建
#   scripts/docker-local.sh rebuild   重新构建 app 镜像并重建容器
#   scripts/docker-local.sh stop      停止容器（保留数据）
#   scripts/docker-local.sh down      删除容器与网络（保留 .docker-data）
#   scripts/docker-local.sh restart   重启 app
#   scripts/docker-local.sh logs      跟踪 app 日志
#   scripts/docker-local.sh status    查看容器状态
#   scripts/docker-local.sh shell     进入 app 容器
#   scripts/docker-local.sh reset     清空 .docker-data 与配置后重新开始
#
# 可用 DJ_HTTP_PORT 覆盖宿主机端口，默认 8080。

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

CONFIG_FILE="$ROOT_DIR/config.docker.yml"
CONFIG_TEMPLATE="$ROOT_DIR/config.docker.yml.example"
DATA_DIR="$ROOT_DIR/.docker-data"
HTTP_PORT="${DJ_HTTP_PORT:-8080}"

compose() {
  docker compose -f "$ROOT_DIR/docker-compose.yml" "$@"
}

require_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    echo "未找到 docker，请先安装 Docker Desktop 或 Docker Engine。" >&2
    exit 1
  fi
  if ! docker compose version >/dev/null 2>&1; then
    echo "未找到 docker compose v2 插件。" >&2
    exit 1
  fi
}

random_secret() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32
    return
  fi
  LC_ALL=C tr -dc 'a-f0-9' </dev/urandom | head -c 64
}

random_password() {
  if command -v openssl >/dev/null 2>&1; then
    printf 'Dj%s' "$(openssl rand -hex 12)"
    return
  fi
  printf 'Dj%s' "$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 20)"
}

ensure_config() {
  if [[ -f "$CONFIG_FILE" ]]; then
    return
  fi
  if [[ ! -f "$CONFIG_TEMPLATE" ]]; then
    echo "缺少配置模板: $CONFIG_TEMPLATE" >&2
    exit 1
  fi

  local app_secret jwt_secret user_jwt_secret admin_password
  app_secret="$(random_secret)"
  jwt_secret="$(random_secret)"
  user_jwt_secret="$(random_secret)"
  admin_password="$(random_password)"

  sed \
    -e "s|__APP_SECRET_KEY__|$app_secret|" \
    -e "s|__JWT_SECRET__|$jwt_secret|" \
    -e "s|__USER_JWT_SECRET__|$user_jwt_secret|" \
    -e "s|__ADMIN_PASSWORD__|$admin_password|" \
    "$CONFIG_TEMPLATE" >"$CONFIG_FILE"

  chmod 600 "$CONFIG_FILE"

  echo "已生成 $CONFIG_FILE"
  echo "后台账号: admin"
  echo "后台密码: $admin_password"
  echo "后台地址: http://localhost:${HTTP_PORT}/admin"
  echo "（该文件含随机密钥，已被 .gitignore 忽略）"
}

wait_healthy() {
  local url="http://127.0.0.1:${HTTP_PORT}/health"
  for _ in $(seq 1 60); do
    if curl -fsS --max-time 2 "$url" >/dev/null 2>&1; then
      echo "服务已就绪: http://localhost:${HTTP_PORT}"
      echo "后台地址:   http://localhost:${HTTP_PORT}/admin"
      return
    fi
    sleep 2
  done
  echo "等待健康检查超时，请执行 scripts/docker-local.sh logs 查看日志。" >&2
  exit 1
}

case "${1:-up}" in
  init)
    require_docker
    ensure_config
    ;;
  up)
    require_docker
    ensure_config
    mkdir -p "$DATA_DIR/db" "$DATA_DIR/uploads" "$DATA_DIR/logs" "$DATA_DIR/redis"
    compose up -d --build
    wait_healthy
    ;;
  start)
    require_docker
    ensure_config
    compose up -d
    wait_healthy
    ;;
  rebuild)
    require_docker
    ensure_config
    compose build app
    compose up -d --force-recreate app
    wait_healthy
    ;;
  stop)
    require_docker
    compose stop
    ;;
  down)
    require_docker
    compose down
    ;;
  restart)
    require_docker
    compose restart app
    wait_healthy
    ;;
  logs)
    require_docker
    compose logs -f --tail=200 app
    ;;
  status)
    require_docker
    compose ps
    ;;
  shell)
    require_docker
    compose exec app sh
    ;;
  reset)
    require_docker
    echo "这会删除 .docker-data/（数据库、上传文件、日志）并重新生成配置。"
    read -r -p "输入 RESET 确认: " answer
    if [[ "$answer" != "RESET" ]]; then
      echo "已取消。"
      exit 0
    fi
    compose down
    rm -rf "$DATA_DIR" "$CONFIG_FILE"
    ensure_config
    mkdir -p "$DATA_DIR/db" "$DATA_DIR/uploads" "$DATA_DIR/logs" "$DATA_DIR/redis"
    compose up -d --build
    wait_healthy
    ;;
  *)
    sed -n '2,20p' "$0"
    exit 1
    ;;
esac
