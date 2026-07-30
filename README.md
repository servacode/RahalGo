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

# 2) الخادم (طرفية ثانية)
cd backend
ADMIN_PHONE=09XXXXXXXX go run ./cmd/api    # ضع رقمك — سيصبح أدمن تلقائياً

# 3) لوحة الأدمن (طرفية ثالثة)
cd web
pnpm install
pnpm dev
```

ثم افتح **http://localhost:3001**

**الدخول أول مرة**: اختر تبويب **"برمز التحقق"** وأدخل رقمك — في وضع التطوير يُطبع الرمز في **طرفية الخادم** (سطر `DEV OTP`). بعد الدخول اضبط كلمة مرور من لوحة... (أو استمر بالـOTP).

> على ويندوز: استخدم `set ADMIN_PHONE=09XXXXXXXX && go run ./cmd/api` أو شغّلها من Git Bash.
> لتفعيل واتساب الحقيقي: `OTP_PROVIDER=whatsapp` ثم امسح QR من صفحة "بوت واتساب" في اللوحة.

## التقنيات المعتمدة

Go • PostgreSQL + PostGIS • Redis • Next.js + TypeScript + Tailwind • Kotlin + Jetpack Compose • OpenStreetMap + OSRM • whatsmeow • Docker + Caddy • Hetzner
