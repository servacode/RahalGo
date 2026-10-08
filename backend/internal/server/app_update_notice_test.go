package server

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/release"
)

// **كلُّ تطبيقٍ في مركز التنزيل له دورٌ يُعلَن له تحديثُه** — تطبيقٌ خامسٌ يُضاف
// بلا دورٍ هنا يُرفع ولا يعلم به أحد.
func TestAppUpdateRoleCoversEveryApp(t *testing.T) {
	for _, a := range release.Apps {
		if appUpdateRole[a.Key] == "" {
			t.Errorf("التطبيق %q بلا دورٍ لإعلان تحديثه", a.Key)
		}
	}
}
