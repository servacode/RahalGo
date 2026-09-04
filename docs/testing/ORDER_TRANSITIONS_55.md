# رحّال غو — انتقالاتُ الطلب ٥٥/٥٥

> **مُولَّدةٌ آليّاً** من `internal/orders/statuses.go` — الخارطتان معاً
> مع قاعدةِ الدمج في `transitionsFor`.
>
> **CODE TRUTH BASELINE**: `26f93c5d`

---

## القواعدُ المشتركةُ لكلّ انتقال — **مُثبَتةٌ مرّةً وتسري على الـ٥٥**

| البند | المُثبَت |
|---|---|
| **البوّابة** | `Service.TransitionWithReason` — **ولا انتقالَ إلّا عبرها** إلّا `DSW-2` |
| **التخويل** | `canTransition(kind, from, to, roles)` · **والأدمنُ مخوَّلٌ بكلّ ما في الخارطة ولا يخترع** ✅ |
| **القفل** | `SELECT … FROM orders WHERE id = $1 **FOR UPDATE**` — **يُسلسل المتزامنَين** ✅ |
| **التكرار** | **النداءُ الثاني يقرأ الحالةَ الجديدة** فيسقط في `canTransition` — **بلا أثرٍ جانبيّ** ✅ |
| **التاريخ** | `INSERT INTO order_events (order_id, from_status, to_status, actor_id, note)` ✅ |
| **العلّةُ الإلزاميّة** | `requiresReason` = `rejected · cancelled · failed · refunded` ✅ |
| **المعاملة** | **واحدةٌ تضمّ الحالةَ والتاريخَ والمال** ✅ |
| **البثّ** | `touch("order","ops")` |
| **التدقيق** | **للموظّفين فقط** — «المتجر يقبل مئة طلب في اليوم، وتسجيلُها يُغرق السجلّ» |
| **الفشل** | **تُرجَع المعاملةُ كاملةً — ولا كتابةَ جزئيّة** ✅ |

## قاعدةُ الدمج — **لماذا ٥٥ لا ٣٨**

```go
transitionsFor(kind, from):
  إن كان مخصَّصاً:
    ١· تخصيصٌ صريحٌ لهذه الحال؟        → يُؤخذ وحدَه
    ٢· الحالُ من حالات المتجر؟          → لا انتقالَ (accepted · preparing)
    ٣· وإلّا                            → خارطةُ العامّ منقّاةً من حالات المتجر
```

**فالثمانيةُ في `customTransitions` ليست الخارطةَ** — **بل تخصيصاتٌ
تسبق العامّ.**

---

### الطلبُ العاديّ — ٣٠

| # | من | إلى | الأدوار | علّة؟ | المال |
|---|---|---|---|---|---|
| N01 | `Pending` | `Accepted` | merchant · ops | — | — |
| N02 | `Pending` | `Rejected` | merchant · ops | **سبب** | ردٌّ للمحفظة |
| N03 | `Pending` | `Cancelled` | customer ·  ops | **سبب** | ردٌّ للمحفظة |
| N04 | `Accepted` | `Preparing` | merchant · ops | — | — |
| N05 | `Accepted` | `Dispatching` | ops | — | — |
| N06 | `Accepted` | `Cancelled` | customer ·  merchant ·  ops | **سبب** | ردٌّ للمحفظة |
| N07 | `Preparing` | `Dispatching` | ops | — | — |
| N08 | `Preparing` | `Cancelled` | merchant · ops | **سبب** | ردٌّ للمحفظة |
| N09 | `Dispatching` | `Assigned` | driver · ops | — | — |
| N10 | `Dispatching` | `Cancelled` | ops | **سبب** | ردٌّ للمحفظة |
| N11 | `Assigned` | `AtPickup` | driver · ops | — | — |
| N12 | `Assigned` | `Dispatching` | driver · ops | — | — |
| N13 | `Assigned` | `Cancelled` | ops | **سبب** | ردٌّ للمحفظة |
| N14 | `AtPickup` | `PickedUp` | driver · ops | — | — |
| N15 | `AtPickup` | `Failed` | driver · ops | **سبب** | بيدِ المكتب |
| N16 | `AtPickup` | `Dispatching` | driver · ops | — | — |
| N17 | `AtPickup` | `Cancelled` | ops | **سبب** | ردٌّ للمحفظة |
| N18 | `PickedUp` | `OnTheWay` | driver · ops | — | — |
| N19 | `PickedUp` | `Dispatching` | ops | — | — |
| N20 | `PickedUp` | `Failed` | ops | **سبب** | بيدِ المكتب |
| N21 | `PickedUp` | `Cancelled` | ops | **سبب** | ردٌّ للمحفظة |
| N22 | `OnTheWay` | `AtDropoff` | driver · ops | — | — |
| N23 | `OnTheWay` | `Dispatching` | ops | — | — |
| N24 | `OnTheWay` | `Failed` | ops | **سبب** | بيدِ المكتب |
| N25 | `OnTheWay` | `Cancelled` | ops | **سبب** | ردٌّ للمحفظة |
| N26 | `AtDropoff` | `Delivered` | driver · ops | — | **تسويةٌ كاملة** |
| N27 | `AtDropoff` | `Failed` | driver · ops | **سبب** | بيدِ المكتب |
| N28 | `AtDropoff` | `Dispatching` | ops | — | — |
| N29 | `AtDropoff` | `Cancelled` | ops | **سبب** | ردٌّ للمحفظة |
| N30 | `Delivered` | `Refunded` | **admin وحدَه** | **سبب** | عكسُ تسوية |

### الطلبُ المخصَّص — ٢٥

| # | من | إلى | الأدوار | علّة؟ | المصدر |
|---|---|---|---|---|---|
| C01 | `Pending` | `Dispatching` | ops | — | تخصيصٌ صريح |
| C02 | `Pending` | `Rejected` | ops | **سبب** | تخصيصٌ صريح |
| C03 | `Pending` | `Cancelled` | customer ·  ops | **سبب** | تخصيصٌ صريح |
| C04 | `Dispatching` | `Assigned` | driver · ops | — | تخصيصٌ صريح |
| C05 | `Dispatching` | `Cancelled` | customer ·  ops | **سبب** | تخصيصٌ صريح |
| C06 | `Assigned` | `PickedUp` | driver | — | تخصيصٌ صريح |
| C07 | `Assigned` | `Dispatching` | driver ·  ops | — | تخصيصٌ صريح |
| C08 | `Assigned` | `Cancelled` | customer ·  ops | **سبب** | تخصيصٌ صريح |
| C09 | `AtPickup` | `PickedUp` | driver · ops | — | موروثٌ منقّى |
| C10 | `AtPickup` | `Failed` | driver · ops | **سبب** | موروثٌ منقّى |
| C11 | `AtPickup` | `Dispatching` | driver · ops | — | موروثٌ منقّى |
| C12 | `AtPickup` | `Cancelled` | ops | **سبب** | موروثٌ منقّى |
| C13 | `PickedUp` | `OnTheWay` | driver · ops | — | موروثٌ منقّى |
| C14 | `PickedUp` | `Dispatching` | ops | — | موروثٌ منقّى |
| C15 | `PickedUp` | `Failed` | ops | **سبب** | موروثٌ منقّى |
| C16 | `PickedUp` | `Cancelled` | ops | **سبب** | موروثٌ منقّى |
| C17 | `OnTheWay` | `AtDropoff` | driver · ops | — | موروثٌ منقّى |
| C18 | `OnTheWay` | `Dispatching` | ops | — | موروثٌ منقّى |
| C19 | `OnTheWay` | `Failed` | ops | **سبب** | موروثٌ منقّى |
| C20 | `OnTheWay` | `Cancelled` | ops | **سبب** | موروثٌ منقّى |
| C21 | `AtDropoff` | `Delivered` | driver · ops | — | موروثٌ منقّى |
| C22 | `AtDropoff` | `Failed` | driver · ops | **سبب** | موروثٌ منقّى |
| C23 | `AtDropoff` | `Dispatching` | ops | — | موروثٌ منقّى |
| C24 | `AtDropoff` | `Cancelled` | ops | **سبب** | موروثٌ منقّى |
| C25 | `Delivered` | `Refunded` | **admin وحدَه** | **سبب** | موروثٌ منقّى |
