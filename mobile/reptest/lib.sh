# **أدواتُ القيادة المشتركة** — تُستدعى من كلّ دفعة.
#
# **ولا نقرةَ عمياء**: كلُّ ضغطةٍ من حدودِ عنصرٍ يُعثر عليه بنصّه.

S=$(cat /tmp/serial.txt 2>/dev/null || adb devices | grep "device$" | head -1 | cut -f1)
a(){ timeout 60 adb -s "$S" "$@" 2>/dev/null; }
d(){ a exec-out uiautomator dump /dev/tty | tr '<' '\n<' > /tmp/u.xml; }
T(){ grep -oP 'text="\K[^"]+' /tmp/u.xml | grep -v '^$'; }
has(){ T | grep -qF "$1"; }

# c **مركزُ عنصرٍ بنصّه.**
c(){ local B X1 Y1 X2 Y2
  B=$(grep -F "text=\"$1\"" /tmp/u.xml | grep -oP 'bounds="\[[0-9,]+\]\[[0-9,]+\]' | head -1)
  [ -z "$B" ] && return 1
  X1=$(echo "$B"|grep -oP '\[\K[0-9]+'|sed -n 1p); Y1=$(echo "$B"|grep -oP ',\K[0-9]+'|sed -n 1p)
  X2=$(echo "$B"|grep -oP '\[\K[0-9]+'|sed -n 2p); Y2=$(echo "$B"|grep -oP ',\K[0-9]+'|sed -n 2p)
  echo "$(( (X1+X2)/2 )) $(( (Y1+Y2)/2 ))"; }

t(){ local P; P=$(c "$1") || return 1; a shell input tap $P; sleep "${2:-3}"; d; }
tf(){ local P; P=$(c "$1") || return 1; a shell input tap $P; }   # بلا انتظارٍ ولا قراءة
ty(){ local P; P=$(c "$1") || return 1; a shell input tap $P
      local i; for i in $(seq 1 "${3:-0}"); do a shell input keyevent 67 >/dev/null; done
      a shell input text "$2"; a shell input keyevent 4; sleep 1; d; }
up(){ local i; for i in $(seq 1 "${1:-1}"); do a shell input swipe 540 1800 540 700 300; sleep 1; done; d; }
dn(){ local i; for i in $(seq 1 "${1:-1}"); do a shell input swipe 540 700 540 1800 300; sleep 1; done; d; }
bk(){ a shell input keyevent 4; sleep "${1:-2}"; d; }
shot(){ a exec-out screencap -p > "C:/Users/adzor/AppData/Local/Temp/claude/d--RahalGo/d280da2b-5a33-42cd-a85f-f46ee558557b/scratchpad/$1.png"; }

OKN=0; NON=0
ok(){ OKN=$((OKN+1)); printf '  ✔ %-7s %s\n' "$1" "$2"; }
no(){ NON=$((NON+1)); printf '  ✗ %-7s %s — %s\n' "$1" "$2" "$3"; }
sk(){ printf '  ؟ %-7s %s — %s\n' "$1" "$2" "$3"; }

# menu_ **يمشي إلى قائمة أصناف العميل الأوّل.**
menu_(){
  a shell am force-stop com.rahalgo.rep
  a shell am start -n com.rahalgo.rep/.MainActivity >/dev/null; sleep 6; d
  t "عملائي" 5 >/dev/null
  local n; n=$(T | grep -m1 -P '^رحال')
  t "$n" 5 >/dev/null
  t "الأصناف" 5 >/dev/null
}
