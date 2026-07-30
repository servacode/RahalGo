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

## التقنيات المعتمدة

Go • PostgreSQL + PostGIS • Redis • Next.js + TypeScript + Tailwind • Kotlin + Jetpack Compose • OpenStreetMap + OSRM • whatsmeow • Docker + Caddy • Hetzner
