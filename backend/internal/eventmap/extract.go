// Package eventmap عقودُ الأحداث — **البثُّ والإشعارُ والدفع.**
//
// **ولا قائمةَ خصوصيّةٍ ثانية** (`P-7` البند ١): الخصوصيّةُ تُقرأ من
// `internal/qa.OrderPrivacy` — عقدُ `P-1` — **ومن نسخ حقلاً أوجد مصدرَي
// حقيقةٍ يفترقان.**
//
// # وما يُستخرَج وما يُعلَن
//
// **مواضعُ الإطلاق تُستخرَج من الشيفرة** — فلا يُثبَّت عددٌ بيد
// (`P-7` البند ٣). **والعقدُ يُعلَن** — «من يجب أن يستقبل» قصدٌ لا تقوله
// الشيفرة.
package eventmap

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Site موضعُ إطلاقٍ كما وُجد في الشيفرة.
type Site struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Call     string `json:"call"`
	Kind     string `json:"kind,omitempty"`
	Entity   string `json:"entity,omitempty"`
	HasApps  bool   `json:"has_app_targeting"`
	HasEntID bool   `json:"has_entity_id"`
}

// Publisher موضعُ بثٍّ لحظيّ.
type Publisher struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Room string `json:"room"`
}

var (
	reNotify = regexp.MustCompile(`\.(Notify|NotifyMany|NotifyOps|NotifyRole|NotifyWallet)\(`)
	rePub    = regexp.MustCompile(`\.Publish\(\s*"([a-z]+)(:?)`)
	reKind   = regexp.MustCompile(`Kind:\s*(?:notifications\.)?Kind(\w+)`)
	reEntity = regexp.MustCompile(`Entity:\s*"([a-z_]+)"`)
)

// Root جذرُ الخلفيّة.
type Root string

// Sites يستخرج مواضعَ إطلاق الإشعارات — **ولا يُثبَّت عددٌ بيد.**
func (r Root) Sites() ([]Site, error) {
	var out []Site
	err := walkGo(string(r), func(path string, src string) {
		lines := strings.Split(src, "\n")
		for i, ln := range lines {
			m := reNotify.FindStringSubmatch(ln)
			if m == nil {
				continue
			}
			// **والكتلةُ تُقرأ حتّى تُغلق** — فحقولُ `Input` قد تمتدّ سطوراً.
			block := blockFrom(lines, i)
			s := Site{File: rel(string(r), path), Line: i + 1, Call: m[1]}
			if k := reKind.FindStringSubmatch(block); k != nil {
				s.Kind = k[1]
			}
			if e := reEntity.FindStringSubmatch(block); e != nil {
				s.Entity = e[1]
			}
			s.HasApps = strings.Contains(block, "Apps:")
			s.HasEntID = strings.Contains(block, "EntityID:")
			out = append(out, s)
		}
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out, nil
}

// Publishers يستخرج مواضعَ البثّ وغرفَها.
func (r Root) Publishers() ([]Publisher, error) {
	var out []Publisher
	err := walkGo(string(r), func(path string, src string) {
		for i, ln := range strings.Split(src, "\n") {
			m := rePub.FindStringSubmatch(ln)
			if m == nil {
				continue
			}
			room := m[1]
			if m[2] == ":" {
				room += ":*"
			}
			out = append(out, Publisher{File: rel(string(r), path), Line: i + 1, Room: room})
		}
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out, nil
}

// Rooms الغرفُ المستعمَلةُ فعلاً.
func Rooms(ps []Publisher) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range ps {
		if !seen[p.Room] {
			seen[p.Room] = true
			out = append(out, p.Room)
		}
	}
	sort.Strings(out)
	return out
}

// UntargetedUserSites إشعاراتُ مستخدمٍ بلا توجيهِ تطبيق — **`XOB-4`.**
//
// **و`NotifyOps` و`NotifyRole` تُستثنيان**: الأولى للعمليّات في الويب،
// **والثانيةُ تُوجَّه بالدور لا بالتطبيق.**
func UntargetedUserSites(ss []Site) []Site {
	var out []Site
	for _, s := range ss {
		if s.Call == "NotifyOps" || s.Call == "NotifyRole" {
			continue
		}
		if !s.HasApps {
			out = append(out, s)
		}
	}
	return out
}

// blockFrom يقرأ من السطر حتّى تُغلق الأقواسُ — **حدُّه ثلاثون سطراً.**
func blockFrom(lines []string, i int) string {
	depth := 0
	var b strings.Builder
	for j := i; j < len(lines) && j < i+30; j++ {
		b.WriteString(lines[j])
		b.WriteByte('\n')
		for _, c := range lines[j] {
			switch c {
			case '(', '{':
				depth++
			case ')', '}':
				depth--
			}
		}
		if j > i && depth <= 0 {
			break
		}
	}
	return b.String()
}

func walkGo(root string, fn func(path, src string)) error {
	return filepath.Walk(filepath.Join(root, "internal"), func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		// **وحزمُ الاختبار لا تُحصى** — `qa` تنادي ما تنادي.
		if strings.Contains(p, string(filepath.Separator)+"qa"+string(filepath.Separator)) {
			return nil
		}
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		fn(p, string(b))
		return nil
	})
}

func rel(root, p string) string {
	r, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return filepath.ToSlash(r)
}
