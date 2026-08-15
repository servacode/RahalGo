# ══════════════════════════════════════════════════════════════════════
#  الويب — بناءٌ على ثلاث مراحل
# ══════════════════════════════════════════════════════════════════════
#
# **ولم تكن له صورةٌ قبل اليوم**: Render كانت تبنيه بأمرٍ لا بحاوية
# (`buildCommand`)، **وسيرفرُنا لا يعرف أمراً — يعرف صورة.**
#
# # ولماذا من جذر مساحة العمل
#
# **الحزمُ المشتركةُ (`packages/*`) تُبنى معه** — و`workspace:*` في
# `package.json` لا يُحلّ إن نُسخ التطبيقُ وحدَه. **فيُنسخ الجذرُ كلُّه.**
#
# # و`NEXT_PUBLIC_*` تُستبدَل وقتَ البناء لا وقتَ التشغيل
#
# **فتُمرَّر كوسائطِ بناءٍ لا كمتغيّراتِ بيئة** — **ومن مرّرها بيئةً فقط
# وجد المتصفّحَ ينادي `localhost:8080`** وهو على الهاتف.

FROM node:22-alpine AS deps
RUN corepack enable
WORKDIR /app
COPY pnpm-workspace.yaml pnpm-lock.yaml package.json turbo.json ./
COPY apps/rahalgo/package.json apps/rahalgo/package.json
COPY packages ./packages
RUN pnpm install --frozen-lockfile

# ══════════════════════════════════════════════════════════════════════
FROM node:22-alpine AS build
RUN corepack enable
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/apps/rahalgo/node_modules ./apps/rahalgo/node_modules
COPY . .

ARG NEXT_PUBLIC_API_URL
ARG NEXT_PUBLIC_SITE_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_SITE_URL=$NEXT_PUBLIC_SITE_URL
ENV NEXT_TELEMETRY_DISABLED=1
RUN pnpm --filter @rahalgo/web build

# ══════════════════════════════════════════════════════════════════════
FROM node:22-alpine AS run
RUN corepack enable
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
COPY --from=build /app /app
EXPOSE 3000
CMD ["pnpm", "--filter", "@rahalgo/web", "start"]
