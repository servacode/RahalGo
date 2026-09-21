# -*- coding: utf-8 -*-
"""
CUSTOMER-ACCEPTANCE case-row ↔ group-row ↔ roll-up total audit.

**تدقيقُ العدّ** — يقرأ كلَّ صفِّ حالةٍ في مصفوفة القبول، ويعدّ الحالاتِ لكلّ
مجموعة، ويقارنها بصفِّ الدرَجِ (roll-up) وبصفِّ المجموع. **المطلوب: صفرُ تعارض.**

(قرارُ المالك ٢٠٢٦-٠٩-٢١: «المصالحةُ الآليّةُ جزءٌ من انضباط الجودة — أبقِها
مفعَّلةً وشغّلها قبل كلّ نقطةِ تفتيش».)

    python scripts/rollup_audit.py     # exit 0 = 0 mismatches, exit 1 otherwise
"""
import re, io, os, sys
sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8')

HERE = os.path.dirname(os.path.abspath(__file__))
MD = os.path.join(HERE, "..", "docs", "testing", "CUSTOMER-ACCEPTANCE-MASTER.md")

STATUSES = ["NOT_TESTED", "NOT_APPLICABLE", "PASS", "FAIL", "BLOCKED"]
ALIAS = {"N/A": "NOT_APPLICABLE", "NA": "NOT_APPLICABLE"}   # the doc uses both spellings
RECOG = set(STATUSES) | set(ALIAS)
lines = io.open(MD, encoding='utf-8').read().split('\n')

# ---- 1. count case rows per group by status (first matrix occurrence of each id) ----
from collections import defaultdict, OrderedDict
grp = defaultdict(lambda: defaultdict(int))
seen = set()
row_re = re.compile(r'^\|\s*(CUST-[A-Z0-9]+-\d+)\s*\|')
for ln in lines:
    m = row_re.match(ln)
    if not m:
        continue
    cid = m.group(1)
    if cid in seen:
        continue
    st = None
    for b in re.findall(r'`([A-Z_/]+)`', ln):
        if b in RECOG:
            st = ALIAS.get(b, b); break
    if st is None:
        continue
    seen.add(cid)
    grp[cid.rsplit('-', 1)[0]][st] += 1

# ---- 2. parse roll-up group rows (§ may carry a letter suffix, e.g. 26A) ----
rollup = OrderedDict()
ru_re = re.compile(r'^\|\s*\d+[A-Z]?\s*\|\s*(CUST-[A-Z0-9]+)\s*\|\s*(\d+)\s*\|\s*(\d+)\s*\|\s*(\d+)\s*\|'
                   r'\s*(\d+)\s*\|\s*(\d+)\s*\|\s*(\d+)\s*\|\s*(\d+)\s*\|\s*(\d+)\s*\|')
for ln in lines:
    m = ru_re.match(ln)
    if m:
        rollup[m.group(1)] = dict(rows=int(m.group(2)),
                                  NOT_TESTED=int(m.group(5)), NOT_APPLICABLE=int(m.group(6)),
                                  PASS=int(m.group(7)), FAIL=int(m.group(8)), BLOCKED=int(m.group(9)))

# ---- 3. compare group by group ----
mismatches = 0
print("%-14s %-30s %-30s" % ("GROUP", "actual(NT/NA/PASS/FAIL/BLK)", "rollup(NT/NA/PASS/FAIL/BLK)"))
for g in sorted(set(list(grp) + list(rollup))):
    av = [grp.get(g, {}).get(s, 0) for s in STATUSES]; a_tot = sum(av)
    r = rollup.get(g)
    astr = "%d/%d/%d/%d/%d(=%d)" % (*av, a_tot)
    if r is None:
        print("%-14s %-30s  <no rollup row>" % (g, astr)); mismatches += 1; continue
    rv = [r[s] for s in STATUSES]
    ok = (av == rv) and (a_tot == r['rows'])
    if not ok: mismatches += 1
    print("%-14s %-30s %-30s%s" % (g, astr, "%d/%d/%d/%d/%d(=%d)" % (*rv, sum(rv)),
                                   "" if ok else "  <<< MISMATCH"))

# ---- 4. totals ----
tot = {s: sum(grp[g].get(s, 0) for g in grp) for s in STATUSES}
grand = sum(tot.values())
print("\nACTUAL TOTALS: PASS=%d FAIL=%d BLOCKED=%d N/A=%d NOT_TESTED=%d  TOTAL=%d" %
      (tot['PASS'], tot['FAIL'], tot['BLOCKED'], tot['NOT_APPLICABLE'], tot['NOT_TESTED'], grand))
tre = re.compile(r'\*\*Total\*\*\s*\|\s*\*\*(\d+)\*\*\s*\|\s*\*\*(\d+)\*\*\s*\|\s*\*\*(\d+)\*\*\s*\|'
                 r'\s*\*\*(\d+)\*\*\s*\|\s*\*\*(\d+)\*\*\s*\|\s*\*\*(\d+)\*\*\s*\|\s*\*\*(\d+)\*\*\s*\|\s*\*\*(\d+)\*\*')
for ln in lines:
    m = tre.search(ln)
    if m:
        d_rows, _, _, d_nt, d_na, d_pass, d_fail, d_blk = map(int, m.groups())
        print("DOC TOTAL ROW: rows=%d NT=%d NA=%d PASS=%d FAIL=%d BLK=%d" %
              (d_rows, d_nt, d_na, d_pass, d_fail, d_blk))
        for name, dv, av in [("rows", d_rows, grand), ("NT", d_nt, tot['NOT_TESTED']),
                             ("NA", d_na, tot['NOT_APPLICABLE']), ("PASS", d_pass, tot['PASS']),
                             ("FAIL", d_fail, tot['FAIL']), ("BLK", d_blk, tot['BLOCKED'])]:
            if dv != av:
                print("  TOTAL MISMATCH %s: doc=%d actual=%d" % (name, dv, av)); mismatches += 1
        break

print("\nMISMATCHES:", mismatches)
sys.exit(0 if mismatches == 0 else 1)
