package impact

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// ChangedFrom يجمع الملفّاتِ المتغيّرة (البند ٢).
//
// **وثلاثةُ مداخل**: مدىً في git · وشجرةُ العمل · وقائمةٌ صريحة.
// **ولا يُخمَّن المصدر** — يُسجَّل في التقرير.
func ChangedFrom(root, base, head string, explicit []string) (source string, files []string, err error) {
	if len(explicit) > 0 {
		return "EXPLICIT_FILES", norm(explicit), nil
	}
	if base != "" {
		rng := base
		if head != "" {
			rng = base + ".." + head
		}
		out, e := git(root, "diff", "--name-only", rng)
		if e != nil {
			return "", nil, fmt.Errorf("مدى git غيرُ مقروء: %w", e)
		}
		return "GIT_RANGE", norm(strings.Split(out, "\n")), nil
	}
	// **شجرةُ العمل** — المعدَّلُ والمُدرَجُ وغيرُ المتتبَّع.
	var all []string
	for _, args := range [][]string{
		{"diff", "--name-only"},
		{"diff", "--name-only", "--cached"},
		{"ls-files", "--others", "--exclude-standard"},
	} {
		out, e := git(root, args...)
		if e != nil {
			continue
		}
		all = append(all, strings.Split(out, "\n")...)
	}
	return "WORKING_TREE", norm(all), nil
}

func git(root string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	b, err := cmd.Output()
	return string(b), err
}

func norm(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range in {
		p = strings.TrimSpace(strings.ReplaceAll(p, "\\", "/"))
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// Human التقريرُ البشريُّ المختصر (البند ٤٠).
func (r *Result) Human() string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f+"\n", a...) }

	w("CHANGE SOURCE = %s   BASE = %s   HEAD = %s", r.Source, dash(r.Base), dash(r.Head))
	w("")
	w("CHANGED (%d)", len(r.Files))
	for i, f := range r.Files {
		if i >= 12 {
			w("  … و%d غيرُها", len(r.Files)-12)
			break
		}
		w("  %-14s %s", f.Category, f.Path)
	}
	w("")
	w("IMPACTED")
	w("  FLOWS    %v", short(r.Flows, 10))
	w("  APPS     %v", r.Apps)
	w("  DEFECTS  %v", short(r.Defects, 10))
	w("  RISKS    %v", short(r.Risks, 10))
	w("  GAPS     %v", short(r.Gaps, 10))
	w("  SETTINGS %d مفتاحاً", len(r.Settings))
	w("")
	w("WHY")
	seen := map[string]bool{}
	n := 0
	for _, x := range r.Reasons {
		k := x.Rule + x.Path
		if seen[k] {
			continue
		}
		seen[k] = true
		if n >= 10 {
			w("  … وأسبابٌ أخرى في CHANGE_IMPACT.json")
			break
		}
		n++
		w("  [%s] %s — %s", x.Depth, x.Rule, x.Path)
	}
	w("")
	req, rec := 0, 0
	for _, t := range r.Tests {
		if t.Required {
			req++
		} else {
			rec++
		}
	}
	w("RUN THESE TESTS   (إلزاميٌّ %d · موصىً به %d)", req, rec)
	w("  %s", r.Command())
	w("")
	w("MODES            %v", r.Modes)
	w("CONFIDENCE       %s", r.Confidence)
	w("RISK CLASS       %s", r.Risk)
	if r.Fallback != "" {
		w("FALLBACK         %s", r.Fallback)
	}
	w("")
	w("DEVICE REQUIRED?  %s", yesList(r.DeviceRequired))
	w("STAGING REQUIRED? %s", yesList(r.StagingRequired))
	w("UNKNOWN IMPACT?   %s", yesList(r.Unknowns))
	safe := "YES"
	if r.Fallback != "" {
		safe = "NO — " + r.Fallback
	}
	w("SAFE TO USE IMPACTED MODE? %s", safe)
	for _, x := range r.Warnings {
		w("⚠️  %s", x)
	}
	return b.String()
}

func dash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func short(in []string, n int) []string {
	if len(in) <= n {
		return in
	}
	return append(append([]string{}, in[:n]...), fmt.Sprintf("…+%d", len(in)-n))
}

func yesList(in []string) string {
	if len(in) == 0 {
		return "NO"
	}
	return "YES — " + strings.Join(in, " · ")
}
