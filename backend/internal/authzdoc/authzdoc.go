// حزمةُ authzdoc — **عقدُ التخويل: من يملك ماذا، وبأيّ إثبات.**
//
// # وما لا يُعاد إخراجُه
//
// **`api/contract.json` يحمل قدرةَ كلِّ مسارٍ أصلاً** (`apidoc/routes.go:327`
// يسأل `authz.LookupAdmin` نفسَها) — **فلا يُعاد.** وما يُخرَج هنا ما لا
// يُخرجه أحد:
//
//	معجمُ القدرات بأوصافها      · لا ملفَّ له، واجهةٌ حيّةٌ فقط
//	جدولُ الأفعال الحسّاسة       · لا ملفَّ له
//	سياسةُ الحقول               · لا ملفَّ له
//	الاستثناءاتُ **بأسبابها**    · العقدُ يطويها في «بحسب المفتاح»
//	تصنيفُ الأدوار وسلطةُ المنح  · لا ملفَّ له
//	طبقاتُ خريطة العمليات       · لا ملفَّ له
//
// # ولماذا لا تُخرَج مصفوفةُ الدور ⇒ القدرة
//
// **لأنّها ليست في الشيفرة.** `role_capabilities` جدولٌ في القاعدة
// (`0137_role_capabilities.sql:28`)، **ويُبدّله كلُّ من يملك `roles.manage`
// من اللوحة.** فما تبذره الهجراتُ حالةٌ أولى لا عقد.
//
// **ومُولِّدٌ يقرأ قاعدةً يكذب مرّتين**: يكذب على من لا قاعدةَ له، **ويُثبّت
// في وثيقةٍ ما يتغيّر بنقرة.** فيُقال ذلك نصّاً في الوثيقة **ولا يُخرَج
// جدولاً يُظنّ ثابتاً.**
package authzdoc

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/opsmap"
)

const (
	JSONPath = "../../../docs/testing/system/AUTHZ_CONTRACT.json"
	DocPath  = "../../../docs/SECURITY-CAPABILITY-CONTRACT.md"
)

var blockRe = regexp.MustCompile(`(?s)<!-- gen:([a-z-]+) -->.*?<!-- /gen:([a-z-]+) -->`)

// Contract عقدُ التخويل مقروءاً بالآلة.
type Contract struct {
	Note string `json:"_note"`

	Capabilities []CapabilityRow `json:"capabilities"`
	FieldPolicy  []FieldRow      `json:"field_policy"`
	RoutePolicy  RoutePolicyInfo `json:"route_policy"`
	Exemptions   []ExemptionRow  `json:"exemptions"`
	Sensitive    []SensitiveRow  `json:"sensitive_actions"`
	RoleClasses  []RoleClassRow  `json:"role_classes"`
	OpsMapPerms  []OpsMapRow     `json:"ops_map_perms"`

	StepUp StepUpInfo `json:"step_up"`
}

// CapabilityRow قدرةٌ واحدةٌ — وأتحرس مساراً أم حقلاً؟
type CapabilityRow struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	// GuardsRoutes عددُ صفوف السياسة التي تطلبها.
	GuardsRoutes int `json:"guards_routes"`
	// GuardsFields حقولٌ في الردود تُحجب بغيابها.
	GuardsFields []string `json:"guards_fields,omitempty"`
	// FieldScopedOnly **قدرةٌ لا تحرس باباً بل حقلاً** — وليست سهواً.
	FieldScopedOnly bool `json:"field_scoped_only,omitempty"`
}

type FieldRow struct {
	Field string `json:"field"`
	Need  string `json:"need"`
}

type RoutePolicyInfo struct {
	Rules int `json:"rules"`
	// Note **ولا تُخرَج الصفوفُ هنا** — انظر `api/contract.json`.
	Note string `json:"note"`
	// CapabilitiesUsed القدراتُ التي يطلبها صفٌّ واحدٌ على الأقلّ.
	CapabilitiesUsed []string `json:"capabilities_used"`
}

type ExemptionRow struct {
	Pattern string `json:"pattern"`
	// Reason **السببُ نصّاً** — وهو ما يضيع في «بحسب المفتاح».
	Reason string `json:"reason"`
}

type SensitiveRow struct {
	Method      string   `json:"method"`
	Pattern     string   `json:"pattern"`
	Action      string   `json:"action"`
	TargetType  string   `json:"target_type"`
	TargetSeg   int      `json:"target_segment"`
	Params      []string `json:"fingerprint_params"`
	Conditional string   `json:"conditional,omitempty"`
	// Need قدرةُ المسار — يُعاد التحقّقُ منها عند إصدار الإذن.
	Need string `json:"need,omitempty"`
}

type RoleClassRow struct {
	Class            string   `json:"class"`
	Roles            []string `json:"roles"`
	Authority        string   `json:"grant_authority"`
	CreatableAtSetup bool     `json:"creatable_at_signup"`
}

type OpsMapRow struct {
	Capability string   `json:"capability"`
	Grants     []string `json:"grants_layers"`
}

// StepUpInfo وصفُ آليّةِ الإثبات — **مُصرَّحٌ بمصدره.**
type StepUpInfo struct {
	ScopeFormula string `json:"scope_formula"`
	TTL          string `json:"ttl"`
	SingleUse    bool   `json:"single_use"`
	Factor       string `json:"factor"`
	Header       string `json:"header"`
	Source       string `json:"source"`
	// PinIsNotStepUp **تصحيحٌ يُكتب صريحاً** — انظر الوثيقة.
	PinIsNotStepUp string `json:"pin_is_not_step_up"`
}

// Build يبني العقدَ من `authz` و`opsmap` — **لا من قائمةٍ هنا.**
func Build() Contract {
	c := Contract{
		Note: "مولَّدٌ من internal/authz و internal/opsmap — لا يُحرَّر بيد. " +
			"go run ./cmd/authzdoc. " +
			"ومصفوفةُ الدور⇒القدرة ليست هنا: جدولٌ في القاعدة يُبدَّل من اللوحة.",
	}

	// ── الحقولُ أوّلاً، لأنّ القدراتِ تُسأل عنها ──────────────────────
	fieldsOf := map[string][]string{}
	for field, need := range authz.FieldPolicy {
		fieldsOf[string(need)] = append(fieldsOf[string(need)], field)
	}
	for _, v := range fieldsOf {
		sort.Strings(v)
	}
	fields := make([]FieldRow, 0, len(authz.FieldPolicy))
	for field, need := range authz.FieldPolicy {
		fields = append(fields, FieldRow{Field: field, Need: string(need)})
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].Field < fields[j].Field })
	c.FieldPolicy = fields

	// ── صفوفُ السياسة تُعَدّ لكلّ قدرة ─────────────────────────────
	routeCount := map[string]int{}
	for _, r := range authz.Rules() {
		routeCount[string(r.Need)]++
	}

	for _, cap := range authz.All() {
		code := string(cap)
		row := CapabilityRow{
			Code:         code,
			Description:  authz.Describe(cap),
			GuardsRoutes: routeCount[code],
			GuardsFields: fieldsOf[code],
		}
		// **قدرةٌ بلا مسارٍ وبحقول** ليست يتيمةً بل حقليّة
		// (`catalog.go:182`): **حقلٌ في ردٍّ لا بابٌ في موجّه.**
		row.FieldScopedOnly = row.GuardsRoutes == 0 && len(row.GuardsFields) > 0
		c.Capabilities = append(c.Capabilities, row)
	}

	used := make([]string, 0, len(routeCount))
	for k := range routeCount {
		used = append(used, k)
	}
	sort.Strings(used)
	c.RoutePolicy = RoutePolicyInfo{
		Rules: authz.PolicyCount(),
		Note: "الصفوفُ نفسُها في api/contract.json بحقل capability لكلّ نقطة — " +
			"فلا تُعاد هنا. المصدر: internal/authz/policy.go:36",
		CapabilitiesUsed: used,
	}

	exs := make([]ExemptionRow, 0, len(authz.Exempt))
	for pat, reason := range authz.Exempt {
		exs = append(exs, ExemptionRow{Pattern: pat, Reason: reason})
	}
	sort.Slice(exs, func(i, j int) bool { return exs[i].Pattern < exs[j].Pattern })
	c.Exemptions = exs

	for _, s := range authz.SensitiveActions() {
		need := ""
		if n, ok := authz.LookupAdmin(s.Method, s.Pattern); ok {
			need = string(n)
		}
		params := append([]string{}, s.Params...)
		if params == nil {
			params = []string{}
		}
		c.Sensitive = append(c.Sensitive, SensitiveRow{
			Method: s.Method, Pattern: s.Pattern, Action: s.Action,
			TargetType: s.TargetType, TargetSeg: s.TargetSeg,
			Params: params, Conditional: s.Conditional, Need: need,
		})
	}
	sort.Slice(c.Sensitive, func(i, j int) bool {
		return c.Sensitive[i].Action < c.Sensitive[j].Action
	})

	for _, cls := range []authz.RoleClass{
		authz.ClassProtected, authz.ClassElevated, authz.ClassStaff,
		authz.ClassAccountType, authz.ClassLegacy,
	} {
		roles := authz.RolesInClass(cls)
		sort.Strings(roles)
		if len(roles) == 0 {
			continue
		}
		c.RoleClasses = append(c.RoleClasses, RoleClassRow{
			Class:            string(cls),
			Roles:            roles,
			Authority:        string(authz.AuthorityToGrant(roles[0])),
			CreatableAtSetup: authz.CreatableAtSignup(roles[0]),
		})
	}

	// ── طبقاتُ خريطة العمليات ────────────────────────────────────────
	//
	// **و`capPerms` غيرُ مُصدَّرة** — فتُستنتج بالسؤال: لكلّ قدرةٍ ولكلّ
	// طبقةٍ، **أتسمح؟** (`opsmap.Allows`). **فلا قائمةٌ ثانيةٌ تشيخ.**
	for _, cap := range authz.All() {
		layers := []string{}
		for _, p := range opsmap.All() {
			if opsmap.Allows([]string{string(cap)}, p) {
				layers = append(layers, string(p))
			}
		}
		if len(layers) > 0 {
			c.OpsMapPerms = append(c.OpsMapPerms, OpsMapRow{
				Capability: string(cap), Grants: layers,
			})
		}
	}

	c.StepUp = StepUpInfo{
		ScopeFormula: `sha256(method + "\n" + path + "\n" + material)`,
		TTL:          "5m",
		SingleUse:    true,
		Factor:       "password",
		Header:       "X-Step-Up",
		Source:       "internal/identity/stepup.go:44",
		PinIsNotStepUp: "الرمزُ السرّيُّ عاملٌ ثانٍ عند الدخول يُحكم بالدور " +
			"(admin أو owner_super_admin، identity/admin_pin.go:93) — " +
			"وليس شرطاً على نقطةٍ. والإثباتُ (step-up) كلمةُ مرورٍ " +
			"محدودةٌ بمسارٍ وجسم. **ولا نقطةَ تطلب الرمزَ السرّيّ.**",
	}

	return c
}

// JSON نصُّ العقد.
func JSON() (string, error) {
	b, err := json.MarshalIndent(Build(), "", " ")
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}

func capabilitiesTable() string {
	c := Build()
	var b strings.Builder
	b.WriteString("| القدرة | تحرس مسارات | تحرس حقولاً | الوصف |\n|---|---|---|---|\n")
	for _, r := range c.Capabilities {
		fields := "—"
		if len(r.GuardsFields) > 0 {
			fields = fmt.Sprintf("**%d**", len(r.GuardsFields))
		}
		routes := fmt.Sprintf("%d", r.GuardsRoutes)
		if r.GuardsRoutes == 0 {
			routes = "**0**"
		}
		b.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s |\n",
			r.Code, routes, fields, r.Description))
	}
	return b.String()
}

func sensitiveTable() string {
	c := Build()
	var b strings.Builder
	b.WriteString("| الفعل | الطريقة | المسار | بصمةُ الجسم | بشرط | القدرة |\n")
	b.WriteString("|---|---|---|---|---|---|\n")
	for _, s := range c.Sensitive {
		params := "—"
		if len(s.Params) > 0 {
			params = "`" + strings.Join(s.Params, "`, `") + "`"
		}
		cond := "دائماً"
		if s.Conditional != "" {
			cond = "`" + s.Conditional + "`"
		}
		need := s.Need
		if need == "" {
			need = "*بحسب المفتاح*"
		} else {
			need = "`" + need + "`"
		}
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | %s | %s | %s |\n",
			s.Action, s.Method, s.Pattern, params, cond, need))
	}
	return b.String()
}

func fieldPolicyTable() string {
	c := Build()
	var b strings.Builder
	b.WriteString("| الحقل | يحتاج |\n|---|---|\n")
	for _, f := range c.FieldPolicy {
		b.WriteString(fmt.Sprintf("| `%s` | `%s` |\n", f.Field, f.Need))
	}
	return b.String()
}

func exemptionsTable() string {
	c := Build()
	var b strings.Builder
	b.WriteString("| النمط | السبب |\n|---|---|\n")
	for _, e := range c.Exemptions {
		b.WriteString(fmt.Sprintf("| `%s` | %s |\n", e.Pattern, e.Reason))
	}
	return b.String()
}

func roleClassTable() string {
	c := Build()
	var b strings.Builder
	b.WriteString("| الصنف | الأدوار | سلطةُ المنح | يُنشأ بالتسجيل؟ |\n|---|---|---|---|\n")
	for _, r := range c.RoleClasses {
		yes := "لا"
		if r.CreatableAtSetup {
			yes = "**نعم**"
		}
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | %s |\n",
			r.Class, strings.Join(r.Roles, "`, `"), r.Authority, yes))
	}
	return b.String()
}

func opsMapTable() string {
	c := Build()
	var b strings.Builder
	b.WriteString("| القدرة | الطبقاتُ التي تفتحها |\n|---|---|\n")
	for _, r := range c.OpsMapPerms {
		b.WriteString(fmt.Sprintf("| `%s` | `%s` |\n",
			r.Capability, strings.Join(r.Grants, "`, `")))
	}
	return b.String()
}

func countsLine() string {
	c := Build()
	orphanFields, noRoute := 0, 0
	for _, r := range c.Capabilities {
		if r.FieldScopedOnly {
			orphanFields++
		}
		if r.GuardsRoutes == 0 && len(r.GuardsFields) == 0 {
			noRoute++
		}
	}
	return fmt.Sprintf(
		"**%d قدرةً · %d صفَّ سياسةٍ للمسارات · %d استثناءً · %d فعلاً حسّاساً · "+
			"%d حقلاً محروساً · %d قدرةً حقليّةً لا تحرس باباً · %d قدرةً لا تحرس شيئاً.**\n",
		len(c.Capabilities), c.RoutePolicy.Rules, len(c.Exemptions),
		len(c.Sensitive), len(c.FieldPolicy), orphanFields, noRoute)
}

// Sections ما يُولَّد في الوثيقة.
func Sections() map[string]string {
	return map[string]string{
		"counts":        countsLine(),
		"capabilities":  capabilitiesTable(),
		"field-policy":  fieldPolicyTable(),
		"exemptions":    exemptionsTable(),
		"sensitive":     sensitiveTable(),
		"role-classes":  roleClassTable(),
		"ops-map-perms": opsMapTable(),
	}
}

// Render يُدخل الأقسام المولَّدة.
func Render(doc string) (string, error) {
	secs := Sections()
	seen := map[string]bool{}
	var bad error
	out := blockRe.ReplaceAllStringFunc(doc, func(mm string) string {
		g := blockRe.FindStringSubmatch(mm)
		open, closeName := g[1], g[2]
		if open != closeName {
			bad = fmt.Errorf("فاصلٌ مفتوحٌ %q ومُغلَقٌ %q", open, closeName)
			return mm
		}
		body, ok := secs[open]
		if !ok {
			bad = fmt.Errorf("فاصلٌ في الوثيقة بلا مُولِّد: %q", open)
			return mm
		}
		seen[open] = true
		return fmt.Sprintf("<!-- gen:%s -->\n%s<!-- /gen:%s -->", open, body, open)
	})
	if bad != nil {
		return "", bad
	}
	missing := []string{}
	for k := range secs {
		if !seen[k] {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return "", fmt.Errorf("مُولِّدٌ بلا فاصلٍ في الوثيقة: %s",
			strings.Join(missing, ", "))
	}
	return out, nil
}

// Write يكتب الوثيقةَ والعقد.
func Write(docPath, jsonPath string) error {
	raw, err := os.ReadFile(docPath)
	if err != nil {
		return err
	}
	out, err := Render(string(raw))
	if err != nil {
		return err
	}
	if out != string(raw) {
		if err := os.WriteFile(docPath, []byte(out), 0o644); err != nil {
			return err
		}
	}
	js, err := JSON()
	if err != nil {
		return err
	}
	if cur, _ := os.ReadFile(jsonPath); string(cur) == js {
		return nil
	}
	return os.WriteFile(jsonPath, []byte(js), 0o644)
}
