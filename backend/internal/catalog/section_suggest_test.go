package catalog

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const ameenNamesFile = "D:/tmp/ameen/ameen_names.txt"

// أسماءُ الأقسام التي يمكن أن يُرجعها الاقتراح (تُجمع من الثوابت).
func allSuggestSections() []string {
	return []string{
		secGrocery, secClean, secCare, secBeauty, secHome, secElectric, secKids,
		secStation, secDairy, secSpices, secNuts, secSnacks, secCold, secHot,
		secWater, secFrozen, secSweets, secBread, secMeat, secPoultry, secFish,
		secFruit, secVeg, secPharmacy, secMobile, secGifts, secClothes, secShoes,
		secHookah, secAccessories, secPets,
	}
}

func TestSuggestSectionNamesExistInMigration(t *testing.T) {
	// أسطر ('الاسم', 'icon', n) في هجرتي الأقسام
	re := regexp.MustCompile(`(?m)^\s*\('([^']+)',\s*'[a-z]+',\s*\d+\)`)
	have := map[string]bool{}
	for _, f := range []string{"0156_launch_sections.sql", "0521_pets_accessories_sections.sql"} {
		raw, err := os.ReadFile(filepath.Join("..", "migrate", "migrations", f))
		if err != nil {
			t.Fatalf("cannot read migration: %v", err)
		}
		for _, m := range re.FindAllStringSubmatch(string(raw), -1) {
			have[m[1]] = true
		}
	}
	if len(have) < 40 {
		t.Fatalf("parsed only %d sections from migrations", len(have))
	}
	for _, s := range allSuggestSections() {
		if !have[s] {
			t.Errorf("section %q is not in 0156/0521 migrations", s)
		}
	}
	// وكل قيمةٍ تخرج فعلاً من القواعد يجب أن تكون ضمن القائمة
	rs := rulesOnce()
	for _, l := range []ruleList{rs.strong, rs.head, rs.weak} {
		for _, rl := range l {
			if rl.section != secAbstain && !have[rl.section] {
				t.Errorf("rule section %q is not in the migration", rl.section)
			}
		}
	}
}

func TestSuggestSectionNormalization(t *testing.T) {
	cases := map[string]string{
		"شَامبو  أطفال":           "شامبو اطفال",
		"017057-شاي ١٠ ظرف":       "شاي ظرف",
		"إبريق شاي":               "ابريق شاي",
		"جبنة ٢٥٠":                "جبنه",
		"ENERGIZER بطاريات 4":     "energizer بطاريات",
		"مـــيـــاه  الــمــدينة": "مياه مدينه",
	}
	for in, want := range cases {
		got := strings.Join(tokenize(in), " ")
		if got != want {
			t.Errorf("tokenize(%q) = %q, want %q", in, got, want)
		}
	}
}

// الاسمُ نفسه بتشكيلٍ وأرقامٍ هندية ورقمِ مادةٍ بادئ يعطي نفس القسم.
func TestSuggestSectionNoiseInvariant(t *testing.T) {
	base, ok := SuggestSection("شامبو بانتين 390 مل")
	if !ok || base != secCare {
		t.Fatalf("base = %q,%v", base, ok)
	}
	for _, v := range []string{
		"017057-شامبو بانتين 390 مل",
		"شَامْبُو بَانْتِين ٣٩٠مل",
		"شامبو   بانتين*6",
		"ШАМПУ شامبو بانتين",
	} {
		if got, ok := SuggestSection(v); !ok || got != base {
			t.Errorf("SuggestSection(%q) = %q,%v; want %q", v, got, ok, base)
		}
	}
}

// أسماءٌ حقيقيّةٌ من ملفّات الأمين، مدقَّقةٌ يدويّاً.
func TestSuggestSectionRealNames(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		// مواد غذائيّة
		{"رز بسمتي الاحلام 1 كغ", secGrocery},
		{"سكر 10كغ", secGrocery},
		{"زيت زيتون زنار 500مل", secGrocery},
		{"طحين النورس 750غ", secGrocery},
		{"معكرونة سباكيتي الدرة 350 غ", secGrocery},
		{"مربى مشمش الدرة 5 كغ", secGrocery},
		{"دبس رمان الاحلام 350غ", secGrocery},
		{"تونا ليو تونغول مدخن 185 غ", secGrocery},
		{"ورق عنب الراقي 900غ", secGrocery},
		{"لحم تونا تمارى عالي البروتين 75غ", secGrocery},
		{"معجون طماطم العجمي 1 كغ", secGrocery},
		{"كورن فليكس نسلة غولد 650غ ذرة", secGrocery},
		// ألبان
		{"جبنة حلوم نيو بارك 540غ", secDairy},
		{"لبنة البوادي 350غ", secDairy},
		{"حليب ريفال كرتون 1 لتر كامل الدسم", secDairy},
		{"كريمة طبخ نيو بارك علبة 850مل", secDairy},
		{"شرائح جبنة سندويش المراعي 200غ 10 شرائح", secDairy},
		// بهارات
		{"بهارات كاري جوليانا لاند 50غ", secSpices},
		{"زعتر احمر سيدي هشام مطربان 500 غ", secSpices},
		{"ماء ورد الغوطة 250مل", secSpices},
		{"8230-بهارات كاري جوليانا لاند 50غ", secSpices},
		// موالح
		{"فستق حلبي جود لايف 250غ علبة", secNuts},
		{"تمر البدر جامبو 825غ", secNuts},
		{"بزر قرع عريض جود 250غ", secNuts},
		// سناكات وحلويات
		{"شيبس ليز 105غ كلاسك", secSnacks},
		{"بسكويت اوريو 41غ", secSnacks},
		{"كيك ريماس 40غ شوكولاته", secSnacks},
		{"شوكولاته جالاكسي 30غ كراميل", secSweets},
		{"علك فيفيدينت ستورمينغ 33غ مانجو", secSweets},
		{"حلاوة جوليانا 400غ فستق", secSweets},
		{"جلاتين هاريبو 80غ غولد بيرز", secSweets},
		// مشروبات
		{"عصير هنا 250 مل مانجا", secCold},
		{"كولا سفن اب 330 مل تنك سوري", secCold},
		{"مشروب طاقة مونيستر مانجو 500مل", secCold},
		{"قهوة باردة نسكافيه 240مل لاتيه", secCold},
		{"شاي بارد اريزونا 500مل ليمون", secCold},
		{"شاي كرك محمد السمان 500غ منجا", secHot},
		{"قهوة 3×1 سادة محمود 24×24 ظرف 18 غ", secHot},
		{"نسكافيه 3في1 18*25غ (450غ) اورجنال", secHot},
		{"مياه بردى 330مل", secWater},
		// منظّفات
		{"مسحوق غسيل ليتس 2 كغ", secClean},
		{"سائل جلي بريل ليمون 600 مل + 50 مل مجانا", secClean},
		{"اريل بلاتينيوم جل 1.8 ليتر فلورل", secClean},
		{"كلور كانوكس مدار 800 مل", secClean},
		{"معطر جو غليد رذاذ 300مل", secClean},
		{"محارم كرزة 500غ", secClean},
		{"مزيل بقع فانيش 450مل اوكسي اكشن", secClean},
		// عناية شخصية
		{"شامبو بانتين 390 مل سعودي", secCare},
		{"شامبو نون للاطفال ضد القمل 250مل", secCare},
		{"معجون اسنان كولجيت مضخة 100مل", secCare},
		{"مزيل تعرق دوف جاف 74غ", secCare},
		{"فوط نسائية صفا 25 ق", secCare},
		{"صابون سائل برافو 450مل", secCare},
		{"كريم يدين فلورنس 100مل", secCare},
		{"سيروم شعر هيد اند شولدرز 145مل", secCare},
		{"فرشاة اسنان بانات ثلاثية وسط 313", secCare},
		// تجميل وعطور
		{"عطر لطافة مهرة 100مل", secBeauty},
		{"حمرة سائلة بينك 15-587", secBeauty},
		{"ايشادو ايفونا 15 لون ES19", secBeauty},
		{"كريم اساس 5509", secBeauty},
		{"مجموعة فراشي مكياج 529-4", secBeauty},
		// منزل وكهرباء
		{"ابريق شاي 22-13", secHome},
		{"فناجين قهوة بورسلين N-2099", secHome},
		{"مقلاية يد بني فاتح 20سم /68-2/", secHome},
		{"مطحنة فلفل 7-72", secHome},
		{"مروحة شحن / N-1539", secElectric},
		{"بطاريات اينرجايزر 2016 BP1", secElectric},
		// أطفال وقرطاسية وموبايل
		{"حفاضات ريكسي نمرة 2", secKids},
		{"ببرونة اطفال بريدو 125مل", secKids},
		{"دفتر عربي وسط ن50", secStation},
		{"قلم رصاص زيرو 505", secStation},
		{"حقيبة مدرسية صيني 2 نافرة", secStation},
		{"شاحن ايفون اوليكس 25W كامل", secMobile},
		{"سماعة بلوتوث ايفون ابيض", secMobile},
		// لحوم ودواجن ومجمّدات وخبز
		{"مرتديلا هنا 380غ دجاج", secPoultry},
		{"هوت دوغ هنا 10اصابع 400غ بقر", secMeat},
		{"بطاطا مفرزة 1كغ الدرة", secFrozen},
		{"بوظة بونتي 50غ", secFrozen},
		{"كبة لبنية درة 500غ", secFrozen},
		{"خبز تنور العصر شعير 8ارغفة", secBread},
		// فواكه وخضار وأركيلة وألبسة
		{"تفاح احمر", secFruit},
		{"بندورة", secVeg},
		{"فحم يحيى ربع", secHookah},
		{"معسل روز", secHookah},
		{"حقيبة ديور قماش رسمات حجم وسط", secAccessories},
		{"ساعة يد الفجر اورجنال", secAccessories},
		{"خاتم اسود ناعم", secAccessories},
		{"اساور ستانلس", secAccessories},
		{"حباسة 11-532", secAccessories},
		{"نظارة رجالي بولارايز 22/23", secAccessories},
		{"بيورينا فريسكيس جاف للقطط البالغة سلمون 1.5 كيلوغرام", secPets},
		{"فيتمين قط فور لايف للقطط المعقمة بلحم الدجاج 1.8 كيلوغرام", secPets},
		{"فور ايفير شامبو للكلاب والقطط 250 مل", secPets},
		{"كنزة صيفي قطط وكلاب قياس M", secPets},
		{"حذاء نسائي جبلي مياو مياو جلد وقماش", secShoes},
		// تعارضات محسومة
		{"بسكويت اطفال اولكر 1كغ", secSnacks},
		{"شامبو فاين اطفال 500مل", secCare},
		{"بهارات كاري الاحلام 100 غ", secSpices},
		{"زهورات بابونج العطار 20 ظرف", secHot},
		{"شامبو سجاد فانيش 450مل", secClean},
	}
	if len(cases) < 60 {
		t.Fatalf("table too small: %d", len(cases))
	}
	for _, c := range cases {
		got, ok := SuggestSection(c.name)
		if !ok || got != c.want {
			t.Errorf("SuggestSection(%q) = %q,%v; want %q", c.name, got, ok, c.want)
		}
	}
}

// أسماء غامضة تنتظر إنساناً.
func TestSuggestSectionAbstains(t *testing.T) {
	for _, n := range []string{
		"ماستر طويل",
		"مجموعة كملي",
		"3ب1 ارمغان 250غ",
	} {
		if s, ok := SuggestSection(n); ok {
			t.Errorf("SuggestSection(%q) = %q; want ok=false", n, s)
		}
	}
}

func TestSuggestSectionEmpty(t *testing.T) {
	for _, in := range []string{"", "   ", "123", "17057-"} {
		if s, ok := SuggestSection(in); ok || s != "" {
			t.Errorf("SuggestSection(%q) = %q, %v; want empty,false", in, s, ok)
		}
	}
}

// لا قاعدةَ تُنتج اسماً خارج الفهرس، ولا اقتراح مع ok=false.
func TestSuggestSectionNoFalseOk(t *testing.T) {
	s, ok := SuggestSection("اسم غير معروف تماما xyz")
	if ok || s != "" {
		t.Errorf("got %q,%v", s, ok)
	}
}

func readAmeenNames(t *testing.T) []string {
	t.Helper()
	f, err := os.Open(ameenNamesFile)
	if err != nil {
		t.Skipf("names file absent: %v", err)
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if l := strings.TrimSpace(sc.Text()); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// تقرير تغطية (سجلٌّ فقط). SUGGEST_DUMP_DIR اختياري لكتابة matched/unmatched.
func TestSuggestSectionCoverageReport(t *testing.T) {
	names := readAmeenNames(t)
	per := map[string]int{}
	var matched, unmatched []string
	byName := map[string][]string{}
	for _, n := range names {
		s, ok := SuggestSection(n)
		if ok {
			per[s]++
			matched = append(matched, s+"\t"+n)
			byName[s] = append(byName[s], n)
		} else {
			unmatched = append(unmatched, n)
		}
	}
	pct := 100 * float64(len(matched)) / float64(len(names))
	t.Logf("names=%d matched=%d (%.1f%%) unmatched=%d", len(names), len(matched), pct, len(unmatched))
	keys := make([]string, 0, len(per))
	for k := range per {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return per[keys[i]] > per[keys[j]] })
	for _, k := range keys {
		t.Logf("  %-22s %d", k, per[k])
	}
	rng := rand.New(rand.NewSource(7))
	t.Log("--- 40 random unmatched ---")
	for i := 0; i < 40 && len(unmatched) > 0; i++ {
		t.Log("  " + unmatched[rng.Intn(len(unmatched))])
	}
	if dir := os.Getenv("SUGGEST_DUMP_DIR"); dir != "" {
		_ = os.WriteFile(filepath.Join(dir, "matched.txt"), []byte(strings.Join(matched, "\n")), 0o644)
		_ = os.WriteFile(filepath.Join(dir, "unmatched.txt"), []byte(strings.Join(unmatched, "\n")), 0o644)
		seed := int64(1)
		if v := os.Getenv("SUGGEST_SEED"); v != "" {
			fmt.Sscan(v, &seed)
		}
		sr := rand.New(rand.NewSource(seed))
		var spot []string
		for i := 0; i < 150 && len(matched) > 0; i++ {
			spot = append(spot, matched[sr.Intn(len(matched))])
		}
		_ = os.WriteFile(filepath.Join(dir, "spot150.txt"), []byte(strings.Join(spot, "\n")), 0o644)
		var sb strings.Builder
		first := map[string]int{}
		for _, n := range unmatched {
			tk := tokenize(n)
			if len(tk) > 0 {
				first[tk[0]]++
			}
		}
		fk := make([]string, 0, len(first))
		for k := range first {
			fk = append(fk, k)
		}
		sort.Slice(fk, func(i, j int) bool { return first[fk[i]] > first[fk[j]] })
		for _, k := range fk {
			fmt.Fprintf(&sb, "%s:%d\n", k, first[k])
		}
		_ = os.WriteFile(filepath.Join(dir, "unmatched_first.txt"), []byte(sb.String()), 0o644)
	}
}
