#!/usr/bin/env bash
# ══════════════════════════════════════════════════════════════════════
# **بوّابةُ الهدف للصدفة — أختُ `target.py` بالقاعدة نفسِها**
# ══════════════════════════════════════════════════════════════════════
#
# (قرارُ المالك ٢٠٢٦-٠٩-٢٩: «لا سكربتَ يقصد الإنتاجَ افتراضاً أو صامتاً».)
#
# # ما كان
#
# **ثلاثةُ محرّكاتِ اختبارٍ كانت تكتب سطراً واحداً**:
#
#     A="${API:-https://api.rahalgo.com/api/v1}"
#
# **فمن شغّلها بلا متغيّرٍ ضرب الإنتاجَ** — لا التجهيز. **والافتراضُ هو
# ما يقع فعلاً**، لأنّ أحداً لا يُصدّر `API` وهو يجرّب على عجل.
#
# # والقاعدةُ الآن
#
#     لا متغيّرَ                  ⇒ التجهيز
#     RAHALGO_TARGET=production  ⇒ **يُرفض** إلّا بعلَمَين صريحين
#
# # كيف يُستعمل
#
#     . "$(dirname "${BASH_SOURCE[0]}")/../../scripts/qa/target.sh"
#     A="$(rg_base_url /api/v1)"
#
# **و`API` القديمُ يبقى مقبولاً** لمن يمرّره صريحاً — فمن كتب
# `API=http://localhost:8080/api/v1` يعمل كما كان، **ولا يُرفض إلّا إن
# كان عنوانَ الإنتاج.**

RG_STAGING="https://staging-api.rahalgo.com"
RG_PRODUCTION="https://api.rahalgo.com"
RG_CONFIRM_WORD="I-KNOW-THIS-IS-PRODUCTION"

# rg_target **يردّ `staging` أو `production`** — والافتراضُ التجهيز.
rg_target() {
  local raw
  raw="$(printf '%s' "${RAHALGO_TARGET:-staging}" | tr '[:upper:]' '[:lower:]')"
  case "$raw" in
    ""|staging|stg) printf 'staging' ;;
    production|prod) printf 'production' ;;
    *) printf 'RAHALGO_TARGET unknown: %s (staging|production)\n' "$raw" >&2
       exit 2 ;;
  esac
}

# rg_refuse_production **يطبع الرفضَ ويقتل العمليّة.**
rg_refuse_production() {
  {
    printf '\n'
    printf '  !!! REFUSED: production needs an explicit permission !!!\n'
    printf '  *** REFUSED: no run against production without permission ***\n\n'
    printf '  target asked : production (%s)\n' "$RG_PRODUCTION"
    printf '  missing      : RAHALGO_ALLOW_PRODUCTION=1\n'
    printf '                 RAHALGO_CONFIRM=%s\n\n' "$RG_CONFIRM_WORD"
    printf '  production holds REAL customer data: a test order there is a\n'
    printf '  real order, and a money row there is a real ledger row.\n'
    printf '  Pass nothing at all and the script goes to staging.\n\n'
  } >&2
  exit 2
}

# rg_base_url **عنوانُ الهدف** — `rg_base_url /api/v1` أو بلا لاحقة.
rg_base_url() {
  local suffix="${1:-}" t
  t="$(rg_target)" || exit 2

  # **ومن مرّر `API` صريحاً فله ما مرّر** — إلّا أن يكون الإنتاجَ.
  if [ -n "${API:-}" ]; then
    case "$API" in
      "$RG_PRODUCTION"*)
        [ "${RAHALGO_ALLOW_PRODUCTION:-}" = "1" ] \
          && [ "${RAHALGO_CONFIRM:-}" = "$RG_CONFIRM_WORD" ] \
          || rg_refuse_production
        printf 'WARNING target=production (%s) - explicit permission given\n' \
          "$RG_PRODUCTION" >&2 ;;
    esac
    printf '%s' "$API"
    return 0
  fi

  if [ "$t" = staging ]; then
    printf '%s%s' "$RG_STAGING" "$suffix"
    return 0
  fi

  [ "${RAHALGO_ALLOW_PRODUCTION:-}" = "1" ] \
    && [ "${RAHALGO_CONFIRM:-}" = "$RG_CONFIRM_WORD" ] \
    || rg_refuse_production
  printf 'WARNING target=production (%s) - explicit permission given\n' \
    "$RG_PRODUCTION" >&2
  printf '%s%s' "$RG_PRODUCTION" "$suffix"
}

# rg_banner **سطرٌ يُطبع في أوّل كلّ محرّك** — فلا يُقرأ تقريرٌ بلا هدفه.
rg_banner() { printf 'target=%s  url=%s' "$(rg_target)" "$(rg_base_url)"; }
