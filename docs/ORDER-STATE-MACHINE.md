# عقدُ آلةِ الحالات — الطلبُ من إنشائه إلى نهايته

> **نصفُ هذه الوثيقة مولَّدٌ من الشيفرة** — كلُّ ما بين `<!-- gen:… -->`
> يُكتب بـ`go run ./cmd/machinedoc` **ولا يُحرَّر بيد**، ويحرسه
> `TestMachineDocIsCurrent` فيُسقط البناءَ إن شاخ.
>
> **والسردُ بيدي** — وهو ما لا تحمله بنيةُ بيانات.
>
> **ومعها ملفٌّ يُقرأ بالآلة**: `docs/testing/system/ORDER_STATE_MACHINE.json`
> — **وهو الذي تسأله مصفوفةُ القبول**، لا هذه الوثيقة.

<!-- gen:counts -->
**14 حالةً · 5 نهائيّةً · 27 حدّاً قياسيّاً · 22 حدّاً مخصَّصاً · 1 حدّاً لا يملكه أحدٌ في وضع المنصّة · 0 سببَ تعذّر.**
<!-- /gen:counts -->

---

## لماذا وُجدت هذه الوثيقة

**كان في المستودع جدولان للانتقالات لا يعرف أحدُهما الآخر.**

`docs/TRUTH.md` يحمل جدولَين مولَّدَين — **لكنّهما للنوع القياسيّ وحدَه**
(`truth.go:83` يثبّت `KindStandard`)، **وبالتسمياتِ العربيّة وحدَها**،
**وبالأدوارِ الفعليّةِ بعد التنقية لا بالأدوارِ المُصرَّحة.** فمن أراد أن
يسأل «ما الذي صُرِّح في الخريطة؟» لم يجد جواباً.

و`docs/testing/ORDER_TRANSITIONS_55.md` يقول في سطره الثالث إنّه
**«مُولَّدةٌ آليّاً»** — **ولا مولِّدَ له في المستودع.** يحرسه عدُّ صفوفٍ
فقط (`testtruth/extract.go:116`): **فلو بُدِّل دورٌ في كلّ صفٍّ من صفوفه
الخمسةِ والخمسين لبقي الحارسُ أخضرَ.**

**وذلك بعينه ما تحذّر منه `truth.go:13`: «وثيقةٌ تكذب أخطرُ من غياب
الوثيقة».**

**فصار للانتقالات عقدٌ واحدٌ مولَّدٌ ومحروس.**

---

## الحالاتُ الأربعَ عشرة

<!-- gen:statuses -->
| # | الرمز | اللفظ | نهائيّة؟ | تُعيد المال؟ |
|---|---|---|---|---|
| 0 | `pending` | بانتظار القبول | لا | لا |
| 1 | `accepted` | مقبول | لا | لا |
| 2 | `preparing` | قيد التحضير | لا | لا |
| 3 | `dispatching` | في الطابور | لا | لا |
| 4 | `assigned` | أُسند لسائق | لا | لا |
| 5 | `at_pickup` | السائق عند المتجر | لا | لا |
| 6 | `picked_up` | استلم السائق | لا | لا |
| 7 | `on_the_way` | في الطريق | لا | لا |
| 8 | `at_dropoff` | عند الزبون | لا | لا |
| 9 | `delivered` | سُلّم | **نعم** | لا |
| 10 | `rejected` | مرفوض | **نعم** | **نعم** |
| 11 | `cancelled` | ملغى | **نعم** | **نعم** |
| 12 | `failed` | تعذّر التسليم | **نعم** | **نعم** |
| 13 | `refunded` | مُسترَدّ | **نعم** | **نعم** |
<!-- /gen:statuses -->

**و«نهائيّة» لا تعني «بلا مخرج».** `delivered` نهائيّةٌ بحكم `terminal()`
(`statuses.go:212`) **ولها مخرجٌ واحدٌ** إلى `refunded` (`statuses.go:182`).
**فمعنى النهائيّةِ هنا «يُغلَق الطلبُ وتقع المحاسبة»** لا «لا خَلَفَ لها».

**وحالتان تحملان نهائيّةَ الطلب في مكانَين**: `terminal()` دالّةٌ،
و`TerminalStatuses()` قائمةٌ — **وكلٌّ منهما يكتب الأسماءَ الخمسةَ بيده.**
فإن أُضيفت حالةٌ نهائيّةٌ سادسةٌ إلى إحداهما **لم تشتكِ الأخرى.** (والتعليقُ
عند `statuses.go:204` يحذّر من «قائمةٍ ثانيةٍ تشيخ صامتةً» — **وهو نفسُه
القائمةُ الثانية.**)

---

## الحدودُ — النوعُ القياسيّ

<!-- gen:edges-standard -->
| من | إلى | الأدوارُ المُصرَّحة | في وضع المنصّة | في وضع المتاجر |
|---|---|---|---|---|
| `pending` | `accepted` | `merchant`, `ops` | العمليات · المالك | المتجر · العمليات · المالك |
| `pending` | `rejected` | `merchant`, `ops` | العمليات · المالك | المتجر · العمليات · المالك |
| `pending` | `cancelled` | `customer`, `ops` | الزبون · العمليات · المالك | الزبون · العمليات · المالك |
| `accepted` | `preparing` | `merchant`, `ops` | **لا أحد** | المتجر · العمليات · المالك |
| `accepted` | `dispatching` | `ops` | العمليات · المالك | العمليات · المالك |
| `accepted` | `cancelled` | `customer`, `merchant`, `ops` | الزبون · العمليات · المالك | الزبون · المتجر · العمليات · المالك |
| `preparing` | `dispatching` | `ops` | العمليات · المالك | العمليات · المالك |
| `preparing` | `cancelled` | `merchant`, `ops` | العمليات · المالك | المتجر · العمليات · المالك |
| `dispatching` | `assigned` | `driver`, `ops` | العمليات · المالك · السائق | العمليات · المالك · السائق |
| `dispatching` | `cancelled` | `ops` | المالك | العمليات · المالك |
| `assigned` | `dispatching` | `driver`, `ops` | العمليات · المالك · السائق | العمليات · المالك · السائق |
| `assigned` | `at_pickup` | `driver`, `ops` | السائق | العمليات · المالك · السائق |
| `assigned` | `cancelled` | `ops` | المالك | العمليات · المالك |
| `at_pickup` | `dispatching` | `driver`, `ops` | العمليات · المالك · السائق | العمليات · المالك · السائق |
| `at_pickup` | `picked_up` | `driver`, `ops` | السائق | العمليات · المالك · السائق |
| `at_pickup` | `cancelled` | `ops` | المالك | العمليات · المالك |
| `at_pickup` | `failed` | `driver`, `ops` | العمليات · المالك · السائق | العمليات · المالك · السائق |
| `picked_up` | `dispatching` | `ops` | العمليات · المالك | العمليات · المالك |
| `picked_up` | `on_the_way` | `driver`, `ops` | السائق | العمليات · المالك · السائق |
| `picked_up` | `failed` | `ops` | العمليات · المالك | العمليات · المالك |
| `on_the_way` | `dispatching` | `ops` | العمليات · المالك | العمليات · المالك |
| `on_the_way` | `at_dropoff` | `driver`, `ops` | السائق | العمليات · المالك · السائق |
| `on_the_way` | `failed` | `ops` | العمليات · المالك | العمليات · المالك |
| `at_dropoff` | `dispatching` | `ops` | العمليات · المالك | العمليات · المالك |
| `at_dropoff` | `delivered` | `driver`, `ops` | السائق | العمليات · المالك · السائق |
| `at_dropoff` | `failed` | `ops` | العمليات · المالك | العمليات · المالك |
| `delivered` | `refunded` | — | المالك | المالك |
<!-- /gen:edges-standard -->

**والعمودُ الثالثُ هو ما صُرِّح، والرابعُ والخامسُ ما يقع.** والفرقُ بينهما
هو `rolesUnderMode` — **وهو ليس رفضاً بل تنقيةً** (`modes.go:56`): يُحذف
الدورُ من قائمةِ الفاعل، ثمّ يُسأل `canTransition` عمّا بقي. **فمن حمل
دورَين مرّ بأيِّهما بقي.**

**و«—» في عمود الأدوار المُصرَّحة تعني قائمةً فارغة** — **وهي لا تعني
«الجميع» بل «المالكَ وحدَه»**، لأنّ `canTransition` يُمرّر المالكَ على كلِّ
حدٍّ في الخريطة بلا نظرٍ إلى أدواره (`statuses.go:330`). **وهذا موضعٌ يُقرأ
مقلوباً** فيُظنّ الحدُّ مفتوحاً للكلّ.

## الحدودُ — النوعُ المخصَّص

<!-- gen:edges-custom -->
| من | إلى | الأدوارُ المُصرَّحة | في وضع المنصّة | في وضع المتاجر |
|---|---|---|---|---|
| `pending` | `dispatching` | `ops` | العمليات · المالك | العمليات · المالك |
| `pending` | `rejected` | `ops` | العمليات · المالك | العمليات · المالك |
| `pending` | `cancelled` | `customer`, `ops` | الزبون · العمليات · المالك | الزبون · العمليات · المالك |
| `dispatching` | `assigned` | `driver`, `ops` | العمليات · المالك · السائق | العمليات · المالك · السائق |
| `dispatching` | `cancelled` | `customer`, `ops` | الزبون · المالك | الزبون · العمليات · المالك |
| `assigned` | `dispatching` | `driver`, `ops` | العمليات · المالك · السائق | العمليات · المالك · السائق |
| `assigned` | `picked_up` | `driver` | السائق | المالك · السائق |
| `assigned` | `cancelled` | `ops` | المالك | العمليات · المالك |
| `at_pickup ↩` | `dispatching` | `driver`, `ops` | العمليات · المالك · السائق | العمليات · المالك · السائق |
| `at_pickup ↩` | `picked_up` | `driver`, `ops` | السائق | العمليات · المالك · السائق |
| `at_pickup ↩` | `cancelled` | `ops` | المالك | العمليات · المالك |
| `at_pickup ↩` | `failed` | `driver`, `ops` | العمليات · المالك · السائق | العمليات · المالك · السائق |
| `picked_up ↩` | `dispatching` | `ops` | العمليات · المالك | العمليات · المالك |
| `picked_up ↩` | `on_the_way` | `driver`, `ops` | السائق | العمليات · المالك · السائق |
| `picked_up ↩` | `failed` | `ops` | العمليات · المالك | العمليات · المالك |
| `on_the_way ↩` | `dispatching` | `ops` | العمليات · المالك | العمليات · المالك |
| `on_the_way ↩` | `at_dropoff` | `driver`, `ops` | السائق | العمليات · المالك · السائق |
| `on_the_way ↩` | `failed` | `ops` | العمليات · المالك | العمليات · المالك |
| `at_dropoff ↩` | `dispatching` | `ops` | العمليات · المالك | العمليات · المالك |
| `at_dropoff ↩` | `delivered` | `driver`, `ops` | السائق | العمليات · المالك · السائق |
| `at_dropoff ↩` | `failed` | `ops` | العمليات · المالك | العمليات · المالك |
| `delivered ↩` | `refunded` | — | المالك | المالك |
<!-- /gen:edges-custom -->

**و`↩` تعني حدّاً موروثاً** من الخريطة القياسيّة لا مُصرَّحاً للمخصَّص
(`statuses.go:318`). **و`accepted` و`preparing` لا مخرجَ لهما في المخصَّص
أصلاً** — `transitionsFor` يردّ `nil` لهما (`statuses.go:315`)، **فالطلبُ
المخصَّصُ لا يمرّ بمتجرٍ.**

---

## ما لا يملكه أحدٌ في وضع المنصّة

<!-- gen:nobody-platform -->
| النوع | من | إلى | الأدوارُ المُصرَّحة | ومن أسقطها |
|---|---|---|---|---|
| `standard` | `accepted` | `preparing` | `merchant`, `ops` | `P2` (وقبلَها `P1` للمتجر) |
<!-- /gen:nobody-platform -->

**وهذا أهمُّ جدولٍ في الوثيقة.**

**الشيفرةُ ترسم الحدَّ، والوضعُ يُسقط كلَّ من يملكه** — فيبقى بابٌ مرسومٌ لا
يفتحه أحد. **ومن قرأ الخريطةَ وحدَها ظنَّه مفتوحاً.**

**وأخطرُها `picked_up → failed` و`on_the_way → failed`.** التعليقُ عند
`statuses.go:151` يقول إنّ هذه المخارجَ **«للعمليات وحدَها»** — **و`P3`
تُسقط `ops` والمالكَ معاً** لأنّ `failed` في `driverOnly` (`modes.go:262`).
**فسائقٌ حمل الطلبَ ثمّ تعذّر عليه التسليمُ في الطريق لا يملك أحدٌ أن
يسجّل تعذّرَه** — **إلّا أن يبلغ `at_dropoff` أوّلاً**، فهناك يملكها
السائق.

**وليس هذا رأياً**: الجدولُ المولَّدُ في `TRUTH.md:138` و`TRUTH.md:142`
يقول `**لا أحد**` منذ أن وُلِّد. **والتعليقُ في الشيفرة يقول غيرَ ذلك.**

> **يُسجَّل بنداً في المصفوفة** (`GAP-SM-01`) — **ولا يُصلَح قبل أن تُثبت
> الدورةُ المتكاملةُ حاجتَه** (حكمُ المالك السادس).

---

## قواعدُ تنقية الأدوار بالوضع

<!-- gen:mode-rules -->
| المعرّف | ما هو | المصدر |
|---|---|---|
| `P1` | وضعُ المنصّة يُسقط دورَ المتجر دائماً | `internal/orders/modes.go:131` |
| `P2` | الدخولُ إلى preparing يُسقط ops وadmin | `internal/orders/modes.go:144` |
| `P3` | وجهةٌ في driverOnly تُسقط ops وadmin — **إلّا at_dropoff → failed**: المكتبُ يُنهي عند الباب (مساءَ ٢٠٢٦-١٠-٠٢) | `internal/orders/modes.go:171` |
| `P4` | **مُبتَلَعةٌ في P3** — شرطُها جزءٌ من شرطِها فلا تُسقط شيئاً جديداً | `internal/orders/modes.go:208` |
| `P5` | الإلغاءُ بعد التسليم للسائق يُسقط ops لغير المالك | `internal/orders/modes.go:224` |
| `M0` | وضعُ المتاجر لا يُنقّي شيئاً — يردّ الأدوارَ كما هي | `internal/orders/modes.go:123` |
| `A0` | المالكُ يملك كلَّ حدٍّ في الخريطة ولا يخترع حدّاً | `internal/orders/statuses.go:330` |
| `D0` | **driverHolds يُمرَّر ولا يُقرأ** — فالجدولُ دالّةُ (kind, from, to, roles, mode) وحدَها | `internal/orders/modes.go:61` |
<!-- /gen:mode-rules -->

**و`P4` مُبتَلَعةٌ في `P3`** — شرطُها `goodsWithDriver[from] && driverOnly[to]`
جزءٌ من شرطِ `P3` وهو `driverOnly[to]` وحدَه. **فلا تُسقط شيئاً لم تُسقطه
`P3` قبلها**، والتعليقُ الطويلُ عند `modes.go:196` يعرضها قاعدةً عاملةً.

**و`driverHolds` تُمرَّر ولا تُقرأ.** وُثِّقت عند `modes.go:59` أنّها تغيّر
الجواب، **وجسمُ الدالّة لا يذكرها.** و`truth.go:80` يقول إنّ الحالَين
يُسألان ويُذكر القيدُ إن اختلفا — **ويُمرّر `true` ثابتاً**، فلا يختلفان
أبداً. **فالجدولُ دالّةُ `(kind, from, to, roles, mode)` وحدَها.**

---

## الحرّاسُ بترتيب تنفيذهم

<!-- gen:guards -->
| المعرّف | ما هو | المصدر |
|---|---|---|
| `G0` | قفلُ الصفِّ FOR UPDATE قبل أيّ قراءة | `internal/orders/transitions.go:70` |
| `G1` | الوضعُ يُقرأ من داخل المعاملة لا قبلها | `internal/orders/transitions.go:106` |
| `G2` | canTransition — وإلّا invalid_transition (409) | `internal/orders/transitions.go:109` |
| `G3` | at_pickup وpicked_up يشترطان سائقاً مُسنَداً — driver_required (409) | `internal/orders/transitions.go:115` |
| `G4` | المخصَّصُ إلى picked_up يشترط custom_agreed_at | `internal/orders/transitions.go:128` |
| `G5` | ويشترط قفلَ السعر quote_confirmed_version == quote_version | `internal/orders/transitions.go:141` |
| `G6` | at_pickup → failed **يُحوَّل** إلى merchantBlocked ولا يُنفَّذ | `internal/orders/transitions.go:168` |
| `G8` | at_dropoff → failed **بلا ذنبٍ مكتوب** يُردّ door_needs_ops (409) — بابُه ResolveDoor وحدَه | `internal/orders/transitions.go:180` |
| `G7` | نافذةُ إلغاء الزبون — وتُقاس على الأدوار الخامّ فيتجاوزها زبونٌ يحمل ops | `internal/orders/transitions.go:301` |
| `H1` | الأدمن يحتاج القدرة orders.intervene | `internal/server/server.go:1176` |
| `H2` | السائقُ إلى delivered يحتاج إثباتاً إن فُعّل drivers.require_delivery_photo | `internal/server/delivery_proof.go:189` |
| `H3` | قبولُ العرض يشترط ورديّةً مفتوحةً وسقفَ نقدٍ وسقفَ طلبات | `internal/server/driver_handlers.go:614` |
<!-- /gen:guards -->

**و`G6` ليس حارساً بل تحويلة**: `at_pickup → failed` **لا يُنفَّذ أبداً** —
يُحوَّل إلى `merchantBlocked` (`transitions.go:168`) فيرجع الطلبُ إلى
`accepted`، **ويُفرَّغ السائقُ ويُعوَّض، ويُثبَّت الذنبُ على المتجر.**

**و`G7` يُقاس على الأدوار الخامّ لا المنقّاة** (`transitions.go:301`) —
**فزبونٌ يحمل `ops` أو المالكَ يتجاوز نافذةَ الإلغاء.** وذلك مقصودٌ ظاهراً،
**ولم أجد نصّاً يقوله.**

---

## ما يقع بلا فاعلٍ بشريّ

<!-- gen:auto-rules -->
| المعرّف | ما هو | المصدر |
|---|---|---|
| `AUTO-PREPARING` | accepted → preparing — **وضعُ المتاجر وحدَه**، بأدوار الفاعل الأصليّ | `internal/orders/transitions.go:552` |
| `AUTO-DISPATCH` | preparing → dispatching — وضعُ المتاجر وorders.auto_dispatch، بأدوار ops قسراً | `internal/orders/transitions.go:559` |
| `AUTO-DISPATCH-MD` | accepted → dispatching لـ«لدي توصيلة» بعد قبول المكتب — orders.auto_dispatch، بأدوار ops قسراً، وفي الوضعين | `internal/orders/transitions.go:591` |
| `AUTO-DISPATCH-EXPORTED` | AutoDispatch بلا شرطِ وضعٍ ولا علَم — **وهو طريقُ وضع المنصّة** | `internal/orders/transitions.go:568` |
| `AUTO-ACCEPT` | pending → accepted بعد orders.auto_accept_min (افتراضُه صفرٌ ⇒ مطفأ) | `internal/orders/watchdog.go:257` |
| `AUTO-TRANSFER` | pending → accepted ثمّ إنزال — orders.auto_transfer (افتراضُه لا) | `internal/server/auto_transfer.go:98` |
| `AUTO-ASSIGN` | dispatching → assigned بالدور | `internal/orders/rotation.go:541` |
<!-- /gen:auto-rules -->

**وثلاثتُها تقع بعد `Commit` لا قبله** (`transitions.go:342`) — **فانتقالٌ
تلقائيٌّ يفشل لا يُرجِع انتقالاً وقع.** وذلك صوابٌ: **الطلبُ الذي قُبل قد
قُبل**، ولو تعذّر إنزالُه إلى الطابور.

**وفي وضع المنصّة — وهو وضعُ التجهيز الحقيقيّ** — **`autoPreparing` و
`autoDispatch` لا تعملان أصلاً**: كلتاهما تشترط وضعَ المتاجر
(`transitions.go:552` و`:559`). **والذي يعمل هو `AutoDispatch` المُصدَّرة**
— بلا شرطِ وضعٍ ولا علَم (`transitions.go:568`) — **ويناديها مسارُ
«أبلِغ المتجرَ»** (`merchant_dispatch.go:297`).

**فالخطوةُ التشغيليّةُ الحقيقيّةُ في وضع المنصّة ثلاثيّة**:

```
قبولٌ  →  «أبلِغ المتجر» (واتساب)  →  إنزالٌ إلى الطابور
accepted → sent_to_merchant_at → dispatching
```

**و«أبلِغ المتجرَ» لا يُبلِغ أحداً.** يبني نصّاً ورابطَ `wa.me` ويسجّل
`sent_to_merchant_at`، **والإرسالُ نقرةُ إنسانٍ على واتساب**
(`merchant_dispatch.go:232`). **فالحقلُ يعني «سُلِّم للعملياتِ لتُرسل» لا
«وصل المتجرَ».**

**و`preparing` لا تُبلَغ في وضع المنصّة أبداً**: `accepted → preparing`
لا يملكها أحد (`P1` تُسقط المتجرَ و`P2` تُسقط البقيّة).

---

## حدودٌ تقع ولا تظهر في خريطة

<!-- gen:implicit-edges -->
| المعرّف | ما هو | المصدر |
|---|---|---|
| `IMPLICIT-AT-PICKUP-ACCEPTED` | at_pickup → accepted — **لا تظهر في خريطةٍ ولا في TRUTH.md**: merchantBlocked يردّ الطلبَ مقبولاً ويُفرّغ السائقَ ويُثبّت الذنبَ على المتجر | `internal/orders/merchant_blocked.go:46` |
<!-- /gen:implicit-edges -->

**ومن قرأ الخريطتَين ظنَّ أنّه رآها كلَّها.**

---

## أسبابُ التعذّر

<!-- gen:fail-reasons -->
| السبب | الذنبُ على | يُعرض عند |
|---|---|---|
<!-- /gen:fail-reasons -->

**والذنبُ يحكم المال**: التعويضُ يُدفع للسائق إذا كان الذنبُ على الزبون أو
المتجر وحدَهما (`transitions.go:632`)، بنسبةِ
`drivers.failed_compensation_percent`. **و`FaultPlatform` مُعرَّفٌ ولا
يحمله سببٌ واحد** (`failreasons.go:43`).

**ولا سببَ إلغاءٍ مُعدَّدٌ في المحرّك** — `cancel_reason` نصٌّ حرٌّ
(`0011_orders.sql:31`) يُملأ من ملاحظةِ الفاعل. **والمطلوبُ أن تكون غيرَ
فارغةٍ فقط** لهذه الحالات:

`rejected` · `cancelled` · `failed` · `refunded`

**وقائمةُ أسبابِ الرفضِ التي يراها المتجرُ واجهةٌ لا محرّك** — يشرحها
تعليقٌ عند `merchant_handlers.go:389` **ولا وجودَ لها في Go.** فمن انتظر
عقداً بأسبابِ إلغاءٍ معدَّدةٍ **لن يجده.**

---

## كيف يُعاد التوليد

```
cd backend && go run ./cmd/machinedoc
go test ./internal/machinedoc/ -run TestMachineDocIsCurrent -count=1
```
