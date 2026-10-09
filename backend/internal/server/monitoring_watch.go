package server

// ══════════════════════════════════════════════════════════════════════
// **راصدُ المنصّة — تنبيهاتٌ للمالك على واتسابه** (مراقبةُ المنصّة، الدفعةُ الأولى ٢٠٢٦-١٠-٠٩)
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **العطبُ يُرى حين يفتح أحدٌ اللوحة** — بوتٌ انفصل ليلاً، قرصٌ امتلأ،
// مسارٌ ينهار مئةَ مرّة: لا يعلم بها المالكُ حتّى يشتكي زبون.
//
// # وما صار
//
// **راصدٌ يقيس كلَّ دقيقة** سبعةَ أشياء، ولكلٍّ منها تنبيهٌ بكلامٍ بسيطٍ يصل
// المالكَ في محادثته مع نفسه على رقم البوت:
//
//	البوت      مفصولٌ أكثرَ من خمس دقائق
//	الطلبات    عالقةٌ بعد مهلتها (`orders.Alerts` — نفسُ شاشة المراقب)
//	القاعدة    واقفة (`dbDownFlag`)
//	Redis      لا يردّ
//	القرص      امتلأ فوق `monitoring.disk_alert_percent`
//	النسخ      لا نسخةَ جديدةً منذ يوم (إن ضُبط `BACKUPS_DIR`)
//	الأخطاء    ٥xx في عشر دقائق ≥ `monitoring.error_spike`
//
// # ولا تكرارَ مزعج
//
//   - **التنبيهُ مرّةً** — ولا يُعاد قبل ساعةٍ ما دامت المشكلةُ قائمة.
//   - **وحين تُحلّ يُقال «رجع طبيعي ✅»** — ثمّ إن عادت أُنذر من جديد.
//   - **وما لم يُقَس** (القاعدةُ واقفةٌ فلا تُعدّ الطلبات) **لا يُحكم عليه** —
//     فلا «رجع طبيعي» كاذب.
//
// # وإلى أين يُرسَل
//
// **إلى رقم البوت نفسِه لا إلى أحدٍ غيره** (`SendToSelf`). **وإن تعذّر**
// (البوتُ هو المفصول) **يُكتب إشعاراً في اللوحة للمالك والمدير** — كي لا
// يضيع تنبيهُ البوت لأنّ البوتَ نفسَه مفصول.
//
// **ولا اسمَ زبونٍ ولا رقمَه ولا عنوانَه ولا سرّ** — أعدادٌ وكلمات.
//
// # والتقريرُ الصباحيّ
//
// **كلَّ يومٍ في الساعة `monitoring.morning_hour` بتوقيت دمشق** — أرقامُ
// مبارح وأخطاءُ آخر يوم و«وضع المنصة» نفسُه (`platformReport`).

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/servacode/rahalgo/backend/internal/platform"
)

// AlertSender **من يكتب للمالك في محادثته مع نفسه** — يُرضيه البوت.
//
// **وواجهةٌ لا نوعٌ بعينه** — فالفحصُ يلتقط الرسائلَ بلا واتساب.
type AlertSender interface {
	SendToSelf(ctx context.Context, text string) error
}

// SetAlertSender **يحقن قناةَ التنبيه** — يُنادى مرّةً عند الإقلاع.
func (s *Server) SetAlertSender(a AlertSender) { s.alertOut = a }

const (
	// alertRepeat **لا يُعاد التنبيهُ نفسُه قبل ساعة** ما دامت المشكلةُ قائمة.
	alertRepeat = time.Hour
	// botGrace **البوتُ ينقطع ويعود وحدَه** — فلا يُنذَر قبل خمس دقائق.
	botGrace = 5 * time.Minute
	// spikeWindow **نافذةُ قفزة الأخطاء.**
	spikeWindow = 10 * time.Minute
	// backupStale **نسخةٌ أقدمُ من هذا قديمة** — يومٌ وساعتان سماحاً للتوقيت.
	backupStale = 26 * time.Hour
	// errorsKeep **كم يبقى عدُّ الأخطاء في الجدول.**
	errorsKeep = 30 * 24 * time.Hour
)

// monCheck **فحصٌ واحدٌ ونتيجتُه.**
type monCheck struct {
	Key string
	// Known **أقِيس؟** — والكاذبُ لا يُحكم عليه: لا تنبيهَ ولا «رجع طبيعي».
	Known bool
	Bad   bool
	// Text **نصُّ التنبيه** و`OKText` **نصُّ عودة السلامة.**
	Text   string
	OKText string
}

// alertBook **ما أُنذر به ومتى** — قلبُ منع التكرار.
type alertBook struct {
	active map[string]time.Time
}

// step **خطوةٌ واحدة**: لكلّ فحصٍ يُقرَّر أيُرسَل شيء. و`send` تُرجع أوصلت —
// **وما لم يصل يُعاد في الخطوة التالية** لا يُعلَّم مُرسَلاً.
func (b *alertBook) step(checks []monCheck, now time.Time, send func(string) bool) {
	if b.active == nil {
		b.active = map[string]time.Time{}
	}
	for _, c := range checks {
		if !c.Known {
			continue
		}
		last, active := b.active[c.Key]
		switch {
		case c.Bad && !active:
			if send(c.Text) {
				b.active[c.Key] = now
			}
		case c.Bad && now.Sub(last) >= alertRepeat:
			if send("🔁 لسّا ما انحلّت:\n" + c.Text) {
				b.active[c.Key] = now
			}
		case !c.Bad && active:
			if send(c.OKText) {
				delete(b.active, c.Key)
			}
		}
	}
}

// monitorState **ذاكرةُ الراصد** — وصفرُها صالح.
type monitorState struct {
	mu           sync.Mutex
	book         alertBook
	botDownSince time.Time
	lastMorning  string
	lastPurge    time.Time
}

// RunMonitoring **راصدُ المنصّة** — يُطلَق مرّةً عند الإقلاع.
func (s *Server) RunMonitoring(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		cctx, cancel := context.WithTimeout(ctx, 45*time.Second)
		s.monitorTick(cctx, time.Now())
		cancel()
	}
}

// monitorTick **نبضةٌ واحدة** — والوقتُ يُمرَّر فيُفحص بلا انتظار.
func (s *Server) monitorTick(ctx context.Context, now time.Time) {
	// **وعدُّ الأخطاء يُكتب دائماً** — ولو أُطفئت التنبيهات: اللوحةُ تقرؤه.
	if err := s.flushErrors(ctx); err != nil {
		s.logger.Warn("المراقبة: تعذّرت كتابةُ عدّاد الأخطاء — يُعاد بعد دقيقة", "error", err)
	}
	s.mon.mu.Lock()
	defer s.mon.mu.Unlock()
	if s.pg != nil && now.Sub(s.mon.lastPurge) >= 24*time.Hour && !dbDownFlag.Load() {
		if _, err := s.pg.Exec(ctx, `DELETE FROM server_errors WHERE hour < $1`,
			now.Add(-errorsKeep)); err == nil {
			s.mon.lastPurge = now
		}
	}
	if !s.monSetting(ctx, "monitoring.enabled") {
		return
	}
	checks := s.monitorChecks(ctx, now)
	s.mon.book.step(checks, now, func(text string) bool { return s.alertOwner(ctx, text) })

	if s.monSetting(ctx, "monitoring.morning_report") {
		hour := 9
		if s.settings != nil {
			hour = int(s.settings.GetInt(ctx, "monitoring.morning_hour"))
		}
		if day, due := morningDue(now, hour, s.mon.lastMorning); due {
			if s.sendSelf(ctx, s.morningReport(ctx, now)) {
				s.mon.lastMorning = day
			}
		}
	}
}

// monSetting **إعدادٌ منطقيّ** — وبلا مخزنٍ يُعدّ مشغَّلاً (افتراضُ الفهرس).
func (s *Server) monSetting(ctx context.Context, key string) bool {
	if s.settings == nil {
		return true
	}
	return s.settings.GetBool(ctx, key)
}

func (s *Server) monInt(ctx context.Context, key string, def int64) int64 {
	if s.settings == nil {
		return def
	}
	return s.settings.GetInt(ctx, key)
}

// morningDue **أحان التقريرُ الصباحيّ؟** — في ساعته بتوقيت دمشق، مرّةً في اليوم.
//
// **وفي الساعة نفسِها لا بعدها** — خادمٌ أُعيد تشغيلُه الظهرَ لا يرسل
// «صباح الخير» في الواحدة.
func morningDue(now time.Time, hour int, last string) (string, bool) {
	l := now.In(platform.Location())
	day := l.Format("2006-01-02")
	return day, l.Hour() == hour && last != day
}

// sendSelf **يكتب للمالك على واتسابه** — ويُرجع أوصل.
func (s *Server) sendSelf(ctx context.Context, text string) bool {
	if s.alertOut == nil {
		return false
	}
	sctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := s.alertOut.SendToSelf(sctx, text); err != nil {
		s.logger.Warn("المراقبة: تعذّر إرسالُ التنبيه على الواتساب", "error", err)
		return false
	}
	return true
}

// alertOwner **تنبيهٌ للمالك** — واتسابُه أوّلاً، وإلّا إشعارُ اللوحة.
func (s *Server) alertOwner(ctx context.Context, text string) bool {
	if s.sendSelf(ctx, text) {
		return true
	}
	title, body, _ := strings.Cut(text, "\n")
	return s.sendOutage(ctx, title, body)
}

// monitorChecks **الفحوصُ السبعة الآن.**
func (s *Server) monitorChecks(ctx context.Context, now time.Time) []monCheck {
	n := func(v int) string { return arNum(int64(v)) }
	var out []monCheck
	dbDown := dbDownFlag.Load()

	// ── البوت ──────────────────────────────────────────────────────
	bot := monCheck{Key: "bot",
		OKText: "✅ رجع طبيعي: بوت الواتساب رجع متّصل."}
	if s.merchant != nil {
		bot.Known = true
		if s.merchant.Ready() {
			s.mon.botDownSince = time.Time{}
		} else {
			if s.mon.botDownSince.IsZero() {
				s.mon.botDownSince = now
			}
			if d := now.Sub(s.mon.botDownSince); d >= botGrace {
				bot.Bad = true
				bot.Text = "⚠️ بوت الواتساب مفصول من " + n(int(d.Minutes())) + " دقيقة\n" +
					"المتاجر ما عم توصلها الطلبات ع الواتس. افتح اللوحة ← الإعدادات ← واتساب وشوف الربط."
			}
		}
	}
	out = append(out, bot)

	// ── القاعدة ────────────────────────────────────────────────────
	out = append(out, monCheck{Key: "db", Known: s.pg != nil, Bad: dbDown,
		Text:   "🚨 قاعدة البيانات واقفة\nالمنصة عم تردّ «مشغولة» على الكل لحتى ترجع.",
		OKText: "✅ رجع طبيعي: قاعدة البيانات رجعت شغّالة."})

	// ── Redis ──────────────────────────────────────────────────────
	rc := monCheck{Key: "redis",
		Text:   "⚠️ الذاكرة السريعة (Redis) ما عم تردّ\nممكن يتأثّر تتبّع السائقين والإشعارات اللحظيّة.",
		OKText: "✅ رجع طبيعي: الذاكرة السريعة (Redis) رجعت تردّ."}
	if s.rdb != nil {
		pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		rc.Known, rc.Bad = true, s.rdb.Ping(pctx).Err() != nil
		cancel()
	}
	out = append(out, rc)

	// ── الطلبات العالقة ─────────────────────────────────────────────
	st := monCheck{Key: "stuck",
		OKText: "✅ رجع طبيعي: ما في طلبات عالقة هلّق."}
	if s.orders != nil && !dbDown {
		if al, err := s.orders.Alerts(ctx); err == nil {
			st.Known, st.Bad = true, len(al) > 0
			st.Text = "⚠️ في " + n(len(al)) + " طلب عالق متأخّر عن مهلته\n" +
				"افتح اللوحة ← المراقبة وشوف مين عليه."
		}
	}
	out = append(out, st)

	// ── القرص ──────────────────────────────────────────────────────
	dk := monCheck{Key: "disk",
		OKText: "✅ رجع طبيعي: في مساحة كافية ع القرص."}
	if s.cfg != nil && s.cfg.UploadsDir != "" {
		if used, ok := diskUsedPercent(s.cfg.UploadsDir); ok {
			limit := s.monInt(ctx, "monitoring.disk_alert_percent", 85)
			dk.Known, dk.Bad = true, limit > 0 && used >= float64(limit)
			dk.Text = "⚠️ القرص قرّب يمتلي: " + n(int(used)) + "٪ مستعمل\n" +
				"إذا امتلى بتوقف الصور والطلبات. لازم ينضّف أو يكبر."
		}
	}
	out = append(out, dk)

	// ── النسخ الاحتياطيّة ───────────────────────────────────────────
	bk := monCheck{Key: "backup",
		OKText: "✅ رجع طبيعي: في نسخة احتياطيّة جديدة."}
	if s.cfg != nil && s.cfg.BackupsDir != "" {
		newest, ok := newestFile(s.cfg.BackupsDir)
		bk.Known = true
		switch {
		case !ok:
			bk.Bad = true
			bk.Text = "⚠️ ما في ولا نسخة احتياطيّة لقاعدة البيانات\nإذا صار شي للخادم منخسر البيانات."
		case now.Sub(newest) > backupStale:
			bk.Bad = true
			bk.Text = "⚠️ آخر نسخة احتياطيّة عمرها " + n(int(now.Sub(newest).Hours())) + " ساعة\n" +
				"النسخ اليوميّ وقف. لازم ينشاف."
		}
	}
	out = append(out, bk)

	// ── قفزةُ الأخطاء ───────────────────────────────────────────────
	sp := monCheck{Key: "errors",
		OKText: "✅ رجع طبيعي: أخطاء الخادم خفّت."}
	if limit := s.monInt(ctx, "monitoring.error_spike", 20); limit > 0 && !dbDown {
		c := s.errs.countSince(now.Add(-spikeWindow))
		sp.Known, sp.Bad = true, c >= int(limit)
		sp.Text = "⚠️ في " + n(c) + " خطأ بالخادم آخر عشر دقايق\n" +
			"في شي عم ينكسر. التفاصيل باللوحة ← المراقبة ← أخطاء الخادم."
	}
	out = append(out, sp)
	return out
}

// newestFile **أحدثُ ملفٍّ في المجلّد** (لا يدخل ما تحته) — وكاذبٌ إن لم يكن ملف.
func newestFile(dir string) (time.Time, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return time.Time{}, false
	}
	var newest time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fi, err := os.Stat(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		if fi.ModTime().After(newest) {
			newest = fi.ModTime()
		}
	}
	return newest, !newest.IsZero()
}

// yesterdayTotals **أرقامُ مبارح بتوقيت دمشق** — بشروط «اليوم» في الإحصاءات نفسِها.
func (s *Server) yesterdayTotals(ctx context.Context, now time.Time) (orders, delivered, cancelled int64, sales int64, err error) {
	today, yesterday, _ := overviewDays(now)
	err = s.pg.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE created_at >= $1 AND created_at < $2),
			count(*) FILTER (WHERE status = 'delivered' AND delivered_at >= $1 AND delivered_at < $2),
			count(*) FILTER (WHERE status IN ('cancelled','rejected','failed')
			                 AND closed_at >= $1 AND closed_at < $2),
			COALESCE(sum(total) FILTER (WHERE status = 'delivered'
			                 AND delivered_at >= $1 AND delivered_at < $2), 0)
		FROM orders
		WHERE (created_at >= $1 AND created_at < $2)
		   OR (closed_at >= $1 AND closed_at < $2)
		   OR (delivered_at >= $1 AND delivered_at < $2)`,
		yesterday, today).Scan(&orders, &delivered, &cancelled, &sales)
	return
}

// morningReport **نصُّ التقرير الصباحيّ** — بلا اسمٍ ولا رقمِ زبون.
func (s *Server) morningReport(ctx context.Context, now time.Time) string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f+"\n", a...) }
	w("🌅 *صباح الخير — تقرير رحّال غو*")
	w("")
	if s.pg != nil && !dbDownFlag.Load() {
		if o, d, c, sales, err := s.yesterdayTotals(ctx, now); err == nil {
			w("*مبارح:* %s طلب · تسلّم %s · انلغى %s · مبيعات %s",
				arNum(o), arNum(d), arNum(c), arNum(sales))
		} else {
			w("*مبارح:* ما قدرت أقرأ أرقام مبارح.")
		}
		if es, err := s.errorsSince(ctx, now.Add(-24*time.Hour), now); err == nil {
			if es.Total == 0 {
				w("*أخطاء الخادم آخر ٢٤ ساعة:* ولا خطأ ✅")
			} else {
				w("*أخطاء الخادم آخر ٢٤ ساعة:* %s", arNum(es.Total))
				for i, g := range es.Groups {
					if i == 3 {
						break
					}
					w("  • %s (%d) × %s", g.Route, g.Code, arNum(g.Count))
				}
			}
		}
	}
	w("")
	b.WriteString(s.platformReport(ctx, "all"))
	return strings.TrimRight(b.String(), "\n")
}
