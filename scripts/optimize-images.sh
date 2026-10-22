#!/usr/bin/env bash
# 为 uploads 下的商品图/横幅生成 WebP 变体（480/800/1200/1800），供前端 srcset 使用。
# 默认 dry-run；确认后加 --apply。需要 ImageMagick（magick 或 convert）。
#
#   scripts/optimize-images.sh                 # dry-run，扫描 ./uploads
#   scripts/optimize-images.sh --apply         # 写入变体
#   scripts/optimize-images.sh /path/uploads --apply
set -euo pipefail

ROOT="uploads"
APPLY=0
for arg in "$@"; do
  case "$arg" in
    --apply) APPLY=1 ;;
    -h|--help)
      echo "usage: $0 [uploads-dir] [--apply]"
      exit 0
      ;;
    *) ROOT="$arg" ;;
  esac
done

if [ ! -d "$ROOT" ]; then
  echo "目录不存在: $ROOT" >&2
  exit 1
fi

if command -v magick >/dev/null 2>&1; then
  IM=magick
elif command -v convert >/dev/null 2>&1; then
  IM=convert
else
  echo "需要 ImageMagick：请先安装 magick 或 convert" >&2
  exit 1
fi

WIDTHS=(480 800 1200 1800)

echo "扫描 ${ROOT}（apply=${APPLY}，工具=${IM}）"
count=0
while IFS= read -r -d '' file; do
  for width in "${WIDTHS[@]}"; do
    output="${file%.*}-${width}.webp"
    if [ -f "$output" ] && [ "$output" -nt "$file" ]; then
      continue
    fi
    echo "$IM \"$file\" -auto-orient -strip -resize \"${width}x>\" -quality 82 -define webp:method=6 \"$output\""
    if [ "$APPLY" = "1" ]; then
      "$IM" "$file" -auto-orient -strip -resize "${width}x>" -quality 82 -define webp:method=6 "$output"
      count=$((count + 1))
    fi
  done
done < <(find "$ROOT" -type f \( -iname '*.jpg' -o -iname '*.jpeg' -o -iname '*.png' \) -print0)

if [ "$APPLY" = "1" ]; then
  echo "完成，生成 $count 个变体。"
else
  echo "这是 dry-run；确认无误后加 --apply 执行。"
fi
