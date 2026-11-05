package drivermonitor

import "time"

// عتبات الحالة الحيّة (طلب المالك ٢٠٢٦-١١-٠٤: «يمشي ولا متوقف»).
const (
	// LiveMoveWindow نافذة قياس الإزاحة: يمشي إن ابتعد أكثر من LiveMoveM خلالها.
	LiveMoveWindow = 2 * time.Minute
	// LiveMoveM إزاحة ٥٠ م خلال النافذة تعني الحركة، وما دونها رجفة GPS.
	LiveMoveM = 50.0
	// LiveMoveSpeedMps سرعة مبلَّغة فوق ١٫٥ م/ث (~٥ كم/س) تعني الحركة.
	LiveMoveSpeedMps = 1.5
	// LiveOfflineAfter لا نقطة منذ ٥ دقائق = غير متصل (النبضة الواقفة كل دقيقة).
	LiveOfflineAfter = 5 * time.Minute
	// LiveLookback مدى النقاط المقروءة؛ يحدّ أيضا «متوقف منذ» بساعة.
	LiveLookback = 60 * time.Minute
)

// الحالات الحيّة.
const (
	LiveMoving  = "moving"
	LiveStopped = "stopped"
	LiveOffline = "offline"
)

// Live الحالة الحيّة لسائق.
type Live struct {
	State    string     `json:"state"`
	Since    *time.Time `json:"since"`
	LastSeen *time.Time `json:"last_seen"`
	OnShift  bool       `json:"on_shift"`
	HasOrder bool       `json:"has_order"`
}

// movingAt هل كانت النقطة i متحرّكة؟ سرعتها فوق العتبة، أو ابتعدت عن نقاط نافذتها.
func movingAt(pts []Point, i int) bool {
	p := pts[i]
	if p.SpeedMps != nil && *p.SpeedMps > LiveMoveSpeedMps {
		return true
	}
	for j := i - 1; j >= 0 && p.At.Sub(pts[j].At) <= LiveMoveWindow; j-- {
		if Dist(pts[j], p) > LiveMoveM {
			return true
		}
	}
	return false
}

// DeriveLive **دالة صافية**: تشتق الحالة من نقاط مرتّبة بالزمن (نقاط آخر ساعة).
//
// «غير متصل» و«آخر ظهور» من النقاط الخام (أي إشارة تعني التطبيق حيّ)، و«يمشي/متوقف» من
// النقاط **بعد `Clean`** (٢٠٢٦-١١-٠٥): نقطتان دقتهما ٣٠٠ م وبينهما ٣٠٠ م ما بيعنوا إنه
// ماشي — **وجوال ما بيبعت غير نقاط ضعيفة بينقرا «متوقف» مو «غير متصل».**
func DeriveLive(raw []Point, now time.Time) Live {
	if len(raw) == 0 {
		return Live{State: LiveOffline}
	}
	seen := raw[len(raw)-1].At
	lv := Live{LastSeen: &seen}
	if now.Sub(seen) > LiveOfflineAfter {
		lv.State = LiveOffline
		lv.Since = &seen
		return lv
	}
	pts := Clean(raw, DefaultConfig())
	if len(pts) == 0 {
		// ما في ولا نقطة منقاسة بالساعة: ما منعرف إنه تحرّك، فمتوقف من أول ما شفناه.
		since := raw[0].At
		lv.State = LiveStopped
		lv.Since = &since
		return lv
	}
	n := len(pts) - 1
	last := pts[n]
	if movingAt(pts, n) && now.Sub(last.At) <= LiveMoveWindow {
		lv.State = LiveMoving
		i := n
		for i > 0 && movingAt(pts, i-1) {
			i--
		}
		since := pts[i].At
		lv.Since = &since
		return lv
	}
	lv.State = LiveStopped
	i := n
	for i > 0 && Dist(pts[i-1], last) <= LiveMoveM {
		i--
	}
	since := pts[i].At
	lv.Since = &since
	return lv
}
