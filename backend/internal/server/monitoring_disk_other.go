//go:build !linux

package server

// diskUsedPercent **لا قياسَ خارج لينكس** — الخادمُ لينكس، وجهازُ المطوّر
// لا يُنذَر عن قرصه. **فيُترك الفحصُ ولا يُخترع رقم.**
func diskUsedPercent(string) (float64, bool) { return 0, false }
