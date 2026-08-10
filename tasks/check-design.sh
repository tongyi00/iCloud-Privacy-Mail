#!/usr/bin/env bash
# 前端重构守护脚本：校验 DOM 契约完整性 + docs/design-system/DESIGN.md 的静态验收标准
# 用法：bash tasks/check-design.sh
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
T="$ROOT/internal/app/templates"
FAIL=0

pass() { printf '  OK   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; FAIL=1; }

# 断言某个 grep 计数等于期望值
expect_count() {
  local label="$1" actual="$2" want="$3"
  if [ "$actual" -eq "$want" ]; then pass "$label ($actual)"; else fail "$label: 实际 $actual，期望 $want"; fi
}

echo "== DOM 契约：JS 引用的 id 必须在页面中存在 =="
# index.html 与 manage.html 中 $('xxx') 引用的 id，逐个确认页面里有 id="xxx"
for f in index.html manage.html; do
  missing=0
  while IFS= read -r id; do
    [ -z "$id" ] && continue
    if ! grep -q "id=\"$id\"" "$T/$f"; then
      fail "$f 缺少 JS 依赖的 id: $id"
      missing=$((missing + 1))
    fi
  done < <(grep -oE "\\\$\('[^']+'\)" "$T/$f" 2>/dev/null | sed -E "s/^\\\$\('//; s/'\)$//" | sort -u)
  [ "$missing" -eq 0 ] && pass "$f 的所有 \$('id') 引用均已命中"
done

echo "== DOM 契约：querySelectorAll 选择器必须仍有匹配 =="
# 这些选择器由 JS 硬编码，类名或 data 属性改名会静默失效
check_sel() {
  local file="$1" pattern="$2" label="$3"
  if grep -q "$pattern" "$T/$file"; then pass "$label"; else fail "$label 在 $file 中已无匹配"; fi
}
check_sel index.html 'class="[^"]*mailbox-message-html'  '.mailbox-message-html'
check_sel index.html 'class="[^"]*manage-refresh-clock'  '.manage-refresh-clock'
check_sel index.html 'class="nav-item[^"]*"[^>]*data-view' '.nav-item[data-view]'
check_sel index.html 'class="[^"]*view-section[^"]*"[^>]*data-view' '.view-section[data-view]'
check_sel index.html 'data-density-option'                '[data-density-option]'
check_sel index.html 'data-log-category'                  '[data-log-category]'
check_sel manage.html 'data-view'                         'manage: [data-view]'
check_sel manage.html 'class="[^"]*account-card'          'manage: .account-card'

echo "== DESIGN.md 验收：装饰清除 =="
# 只统计真实声明，排除 CSS 注释行（注释里提到被移除的属性名不算违规）
decls() { grep -hoE "^[^/*]*\b$1\b" "$T"/*.html | grep -vE "^\s*(/\*|\*)" | wc -l | tr -d ' '; }
grad=$(decls "gradient")
expect_count "gradient 声明数" "$grad" 0
bdf=$(grep -hoE "^\s+-?(webkit-)?backdrop-filter:" "$T"/*.html | wc -l | tr -d ' ')
# 顶栏同时需要 backdrop-filter 与 -webkit- 前缀版本，故允许 2 条声明
if [ "$bdf" -le 2 ]; then pass "backdrop-filter ($bdf ≤ 2，仅允许粘性顶栏及其前缀)"; else fail "backdrop-filter: $bdf 处声明，最多允许 2"; fi
hl=$(grep -hoE "inset 0 1px 0 rgba\(255" "$T"/*.html | wc -l | tr -d ' ')
expect_count "假高光 inset highlight" "$hl" 0

echo "== DESIGN.md 验收：字重只用 400/500/600 =="
heavy=$(grep -hoE "font-weight: *(7|8|9)[0-9]{2}" "$T"/*.html | wc -l | tr -d ' ')
expect_count "font-weight > 600" "$heavy" 0

echo "== DESIGN.md 验收：动效预算 =="
mv=$(grep -hoE "transform: *translate[XY]\(-?[0-9]" "$T"/*.html | wc -l | tr -d ' ')
expect_count "transform 位移" "$mv" 0

echo "== DESIGN.md 验收：圆角取值 ⊆ {4,6,8,999}px =="
bad_radius=$(grep -hoE "border-radius: *[0-9]+px" "$T"/*.html \
  | grep -oE "[0-9]+" | sort -u | grep -vE "^(4|6|8|999)$" | tr '\n' ' ')
if [ -z "$bad_radius" ]; then pass "圆角取值合规"; else fail "非法圆角取值: $bad_radius"; fi

echo "== DESIGN.md 验收：焦点环覆盖 =="
fv=$(grep -ho "focus-visible" "$T"/*.html | wc -l | tr -d ' ')
if [ "$fv" -ge 3 ]; then pass "focus-visible 已定义 ($fv 处)"; else fail "focus-visible 仅 $fv 处，需覆盖按钮/链接/导航/可点卡片"; fi

echo "== DESIGN.md 验收：主题收敛为 light/dark =="
themes=$(grep -hoE 'data-theme="[a-z-]+"' "$T"/*.html | sort -u | grep -vE '"(light|dark)"' | tr '\n' ' ')
if [ -z "$themes" ]; then pass "主题已收敛"; else fail "仍存在多余主题: $themes"; fi

echo ""
if [ "$FAIL" -eq 0 ]; then echo "全部检查通过"; else echo "存在未通过项（重构未完成阶段会预期失败）"; fi
exit "$FAIL"
