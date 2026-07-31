# RahalGo — رحّال

منصة توصيل طلبات متعددة الفئات في محافظة الرقة — سوريا.

**الحزمة الكاملة**: خادم Go • واجهات ويب Next.js (زبون، أدمن/عمليات، متاجر، مندوبون) • تطبيقا أندرويد أصليان Kotlin (زبون، سائق) • بوت واتساب للتحقق والإشعارات.

## الوثائق — تُقرأ بهذا الترتيب

| الملف | المحتوى |
|---|---|
| [docs/PLAN.md](docs/PLAN.md) | الخطة المعتمدة: الرؤية، بحث السوق، المعمارية، القرارات المثبتة |
| [docs/GROUND-RULES.md](docs/GROUND-RULES.md) | القواعد الهندسية الملزمة (المركزية، RTL، المعايير، الأمان) |
| [docs/ROADMAP.md](docs/ROADMAP.md) | مراحل التنفيذ من الصفر إلى الإطلاق ومعايير قبول كل مرحلة |
| [docs/PROGRESS.md](docs/PROGRESS.md) | سجل توثيق المراحل المنجزة |
| [docs/BRAND.md](docs/BRAND.md) | الهوية البصرية: الألوان، الخطوط، قواعد الشكل |

## هيكل المستودع

```
RahalGo/
├── backend/     # خادم Go: REST API + WebSocket + بوت واتساب + مهام خلفية
├── web/         # Monorepo واجهات الويب (Next.js + TypeScript)
│   ├── apps/        # customer / admin / merchant / sales
│   └── packages/    # ui (توكنز+مكونات) / i18n / api-client / config
├── android/     # تطبيقا Kotlin + Jetpack Compose (customer / driver)
└── docs/        # الوثائق
```

## التشغيل المحلي (للتطوير والتجربة)

**المتطلبات**: [Go 1.24+](https://go.dev/dl/) • [Node 22+](https://nodejs.org) مع `corepack enable` (لتفعيل pnpm) • [Docker](https://docs.docker.com/get-docker/)

```bash
# 1) قواعد البيانات (طرفية واحدة — مرة واحدة)
docker compose up -d

# 2) زراعة بيانات التطوير (مرة واحدة — حسابات جاهزة بكلمات مرور + متاجر بقوائمها)
cd backend
go run ./cmd/seed

# 3) الخادم (نفس الطرفية أو ثانية)
go run ./cmd/api

# 4) واجهات الويب (طرفية أخرى)
cd web
pnpm install
pnpm dev     # يشغّل الواجهات كلها معاً
```

| الواجهة | الرابط | الدخول (من بيانات الزراعة) |
|---|---|---|
| لوحة الإدارة | http://localhost:3001 | `0999000001` / `RahalGo@2026` |
| بوابة المتجر | http://localhost:3002 | `0966777888` — OTP من طرفية الخادم |
| موقع الزبون | http://localhost:3003 | تصفح حر — وللطلب: `0933000111` OTP |
| لوحة المندوب | http://localhost:3004 | `0977888999` — OTP من طرفية الخادم |
| تطبيق السائق | http://localhost:3005 | `0955111222` — OTP من طرفية الخادم |

**كل حسابات الزراعة** تُطبع عند تشغيل `go run ./cmd/seed` (أدمن، عمليات، مالية، مندوب، سائق، أصحاب متاجر، زبون برصيد محفظة 200000).

> في وضع التطوير رمز OTP يُطبع في **طرفية الخادم** (سطر `DEV OTP`).
> لتفعيل واتساب الحقيقي: `OTP_PROVIDER=whatsapp` ثم امسح QR من صفحة "بوت واتساب" في اللوحة.

## التقنيات المعتمدة

Go • PostgreSQL + PostGIS • Redis • Next.js + TypeScript + Tailwind • Kotlin + Jetpack Compose • OpenStreetMap + OSRM • whatsmeow • Docker + Caddy • Hetzner
