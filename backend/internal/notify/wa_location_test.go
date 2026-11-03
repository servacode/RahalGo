package notify

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type fakeLIDs map[string]string // lid user ⇒ pn user

func (f fakeLIDs) GetPNForLID(_ context.Context, lid types.JID) (types.JID, error) {
	if pn, ok := f[lid.User]; ok {
		return types.NewJID(pn, types.DefaultUserServer), nil
	}
	return types.JID{}, errors.New("no mapping")
}

func locMsg(lat, lng float64, acc uint32) *waE2E.Message {
	return &waE2E.Message{LocationMessage: &waE2E.LocationMessage{
		DegreesLatitude: proto.Float64(lat), DegreesLongitude: proto.Float64(lng),
		AccuracyInMeters: proto.Uint32(acc)}}
}

func TestLocationFromMessage(t *testing.T) {
	if l, ok := LocationFromMessage(locMsg(35.9, 39.0, 12)); !ok || l.Lat != 35.9 || l.Lng != 39.0 || l.AccuracyM != 12 || l.Live {
		t.Fatalf("static: %+v %v", l, ok)
	}
	live := &waE2E.Message{LiveLocationMessage: &waE2E.LiveLocationMessage{
		DegreesLatitude: proto.Float64(1.5), DegreesLongitude: proto.Float64(2.5)}}
	if l, ok := LocationFromMessage(live); !ok || !l.Live || l.Lat != 1.5 || l.Lng != 2.5 {
		t.Fatalf("live: %+v %v", l, ok)
	}
	if _, ok := LocationFromMessage(&waE2E.Message{Conversation: proto.String("hi")}); ok {
		t.Fatal("نصٌّ عُدّ موقعاً")
	}
	if _, ok := LocationFromMessage(nil); ok {
		t.Fatal("nil عُدّ موقعاً")
	}
}

func TestResolveSenderPhone(t *testing.T) {
	ctx := context.Background()
	pn := types.NewJID("963944555602", types.DefaultUserServer)
	lid := types.NewJID("9988776655", types.HiddenUserServer)
	cases := []struct {
		name string
		info types.MessageInfo
		lids lidResolver
		want string
	}{
		{"رقمٌ مباشر", types.MessageInfo{MessageSource: types.MessageSource{Sender: pn}}, nil, "+963944555602"},
		{"معرّفٌ خفيّ وعنوانٌ بديل", types.MessageInfo{MessageSource: types.MessageSource{Sender: lid, SenderAlt: pn}}, nil, "+963944555602"},
		{"معرّفٌ خفيّ وجدولُ التطابق", types.MessageInfo{MessageSource: types.MessageSource{Sender: lid}}, fakeLIDs{"9988776655": "963944555602"}, "+963944555602"},
		{"معرّفٌ خفيّ بلا تطابق", types.MessageInfo{MessageSource: types.MessageSource{Sender: lid}}, fakeLIDs{}, ""},
		{"معرّفٌ خفيّ بلا مخزن", types.MessageInfo{MessageSource: types.MessageSource{Sender: lid}}, nil, ""},
	}
	for _, c := range cases {
		if got := ResolveSenderPhone(ctx, c.info, c.lids); got != c.want {
			t.Errorf("%s: %q — والمنتظَر %q", c.name, got, c.want)
		}
	}
}

func TestLocationReplyRoutesResolvedPhone(t *testing.T) {
	s := &WhatsAppSender{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	var gotFrom string
	var gotLoc InboundLocation
	calls := 0
	s.SetInboundLocation(func(_ context.Context, from string, l InboundLocation) string {
		calls++
		gotFrom, gotLoc = from, l
		return "تمّ"
	})
	lid := types.NewJID("55", types.HiddenUserServer)
	msg := &events.Message{Info: types.MessageInfo{MessageSource: types.MessageSource{Sender: lid}}, Message: locMsg(35.9, 39.0, 7)}
	reply, isLoc := s.locationReply(context.Background(), msg, fakeLIDs{"55": "963944555602"})
	if !isLoc || reply != "تمّ" || gotFrom != "+963944555602" || gotLoc.Lat != 35.9 || calls != 1 {
		t.Fatalf("reply=%q isLoc=%v from=%q loc=%+v calls=%d", reply, isLoc, gotFrom, gotLoc, calls)
	}
	// رقمٌ لا يُعرف ⇒ يُعدّ موقعاً (لا يمرّ لمسار النصّ) لكن لا نداء ولا ردّ.
	reply, isLoc = s.locationReply(context.Background(), msg, fakeLIDs{})
	if !isLoc || reply != "" || calls != 1 {
		t.Fatalf("بلا رقم: reply=%q isLoc=%v calls=%d", reply, isLoc, calls)
	}
	// رسالةُ نصٍّ ⇒ ليست موقعاً.
	txt := &events.Message{Message: &waE2E.Message{Conversation: proto.String("x")}}
	if _, isLoc := s.locationReply(context.Background(), txt, nil); isLoc {
		t.Fatal("نصٌّ عُدّ موقعاً")
	}
}
