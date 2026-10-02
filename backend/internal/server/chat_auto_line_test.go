package server

// **السطرُ الآليُّ لا يُعدّ على السائق في حدّ المعدّل** (٢٠٢٦-١٠-٠٢) — كانت تحيّتُه
// المكتوبةُ باسمه لحظةَ القبول تجعل أوّلَ ما يكتبه يُردّ «أسرعت».

import (
	"context"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/comms"
)

func TestChat_AutoLineDoesNotCountAgainstTheDriver(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	f.srv.comms = comms.New(f.pool)
	id := f.problemOrderAt(t, "assigned", d)
	// **التحيّةُ الآليّة** — كما يكتبها `openPlainChat` لحظةَ الإسناد.
	if _, err := f.pool.Exec(context.Background(), `
		INSERT INTO order_messages (order_id, sender_id, sender_role, body, driver_id, auto)
		VALUES ($1, $2, 'driver', 'تحيّة', $2, true)`, id, d); err != nil {
		t.Fatal(err)
	}
	send := func() int {
		return f.call(f.srv.handleSendOrderMessage, http.MethodPost, "/orders/"+id+"/messages", id, d,
			[]string{"driver"}, `{"body":"أنا عند المتجر"}`).Code
	}
	if code := send(); code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("أوّلُ رسالةٍ بعد التحيّة ردّت %d — **عُدّ عليه سطرٌ لم يكتبه**", code)
	}
	// **والحدُّ باقٍ على كلامه هو.**
	if code := send(); code < 400 {
		t.Errorf("رسالتان في ثانيتين قُبلتا (%d) — **ذهب حدُّ المعدّل كلُّه**", code)
	}
}
