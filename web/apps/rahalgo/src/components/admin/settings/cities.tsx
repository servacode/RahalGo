"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **المدنُ — أين تعمل المنصّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٢٠: «مو معقول شخصٌ بالشام يطلب من الرقّة… ولا
 *  زبونٌ بأوّل الشام من مطعمٍ بآخر الشام».)
 *
 * # والمدينةُ غيرُ المنطقة
 *
 *	المدينةُ  ←  أيَّ سوقٍ يرى الزبون     (دمشقُ لا حلب)
 *	المنطقةُ  ←  إلى أين نُوصّل داخلَها   (المزّةُ · جرمانا)
 *
 * **والمنطقةُ بنتُ المدينة** — ولذلك تجاورها هنا: **من فتح دمشقَ يضيف
 * مناطقَها في اللوحة نفسِها**، ولا يبحث عن بابٍ ثانٍ.
 *
 * # ولا تُحذف مدينةٌ فيها متاجر
 *
 * **الحذفُ يُفرغ `city_id` فتصير متاجرُها بلا مدينةٍ فتُخفى عن الجميع
 * بلا أن يعلم أحد.** والمحرّكُ يردّها، **والزرُّ هنا لا يُعرض أصلاً**:
 * عددُ متاجرها ظاهرٌ على البطاقة، **وزرٌّ يُضغط فيُردَّ دائماً يُقرأ
 * عطباً.**
 *
 * # والخريطةُ هي خريطةُ المناطق نفسُها
 *
 * **مركزٌ ونصفُ قطر** — والنموذجُ واحد. **وخريطتان بأسلوبين لالتقاط
 * نقطةٍ واحدةٍ تجعلان اللوحةَ تبدو لوحتين.**
 */

import { useCallback, useEffect, useState } from "react";
import dynamic from "next/dynamic";
import { getMessages, defaultLocale, fmtNum, errorText} from "@rahalgo/i18n";
import {
  Alert,
  PageHeader,
  Button,
  Input,
  Badge,
  IconAdd,
  IconDelete,
  IconZones,
  IconClose,
  Confirm,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const CitiesMap = dynamic(() => import("./ZonesMap"), { ssr: false });

const m = getMessages(defaultLocale);
const C = m.admin.cities;

interface City {
  id: string;
  name: string;
  lat: number;
  lng: number;
  radius_m: number;
  /** **مدى التوصيل داخلها** — `null` يعني «خذ افتراضَ المنصّة». */
  max_delivery_m: number | null;
  active: boolean;
  sort_order: number;
  /** **كم متجراً فيها** — من يطفئها يجب أن يعرف كم يُخفي. */
  merchants: number;
}

function translateKey(key: string): string {
  let node: unknown = m;
  for (const part of key.split(".")) {
    if (typeof node !== "object" || node === null) return m.errors.internal;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" ? node : m.errors.internal;
}

interface Draft {
  id: string | null;
  name: string;
  lat: number | null;
  lng: number | null;
  radiusM: number;
  /** **فارغٌ يعني «افتراضُ المنصّة»** — لا صفراً. */
  reachM: string;
  sortOrder: string;
}

export default function CitiesPanel() {
  const { user: me } = useAuth();
  const isAdmin = !!me?.roles.includes("admin");

  const [cities, setCities] = useState<City[]>([]);
  const [error, setError] = useState("");
  const [draft, setDraft] = useState<Draft | null>(null);
  const [selectedID, setSelectedID] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [pendingDelete, setPendingDelete] = useState<City | null>(null);

  const load = useCallback(async () => {
    try {
      const res = await api<{ cities: City[] }>("/api/v1/admin/cities");
      setCities(res.cities ?? []);
      setError("");
    } catch (err) {
      setError(errorText(err));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  function startCreate() {
    setSelectedID(null);
    // **ونصفُ قطرٍ خمسةٌ وعشرون كيلومترا** — يغطّي مدينةً وضواحيَها،
    // **ورقمٌ يبدأ صفراً يجعل أوّلَ حفظٍ يفشل.**
    setDraft({
      id: null,
      name: "",
      lat: null,
      lng: null,
      radiusM: 25000,
      reachM: "",
      sortOrder: "0",
    });
  }

  function startEdit(c: City) {
    setSelectedID(c.id);
    setDraft({
      id: c.id,
      name: c.name,
      lat: c.lat,
      lng: c.lng,
      radiusM: c.radius_m,
      reachM: c.max_delivery_m == null ? "" : String(c.max_delivery_m),
      sortOrder: String(c.sort_order),
    });
  }

  async function save() {
    if (!draft || draft.lat == null) return;
    setBusy(true);
    setError("");
    const reach = draft.reachM.trim();
    const body = {
      name: draft.name.trim(),
      lat: draft.lat,
      lng: draft.lng,
      radius_m: draft.radiusM,
      // **والفراغُ يُرسَل `null` لا صفراً** — العمودُ يرفض الصفر،
      // **و`null` هي «خذ افتراضَ المنصّة».**
      max_delivery_m: reach === "" ? null : Number(reach),
      sort_order: Number(draft.sortOrder) || 0,
      active: true,
    };
    try {
      if (draft.id) {
        await api(`/api/v1/admin/cities/${draft.id}`, {
          method: "PUT",
          body: JSON.stringify(body),
        });
      } else {
        await api("/api/v1/admin/cities", { method: "POST", body: JSON.stringify(body) });
      }
      setDraft(null);
      setSelectedID(null);
      await load();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  /**
   * **والإطفاءُ يُرسل المدينةَ كلَّها** — النقطةُ `PUT` لا `PATCH`.
   *
   * **وحقلٌ يُنسى في `PUT` يُمحى** — فتُرسَل المدينةُ كما هي والرايةُ
   * وحدَها مقلوبة.
   */
  async function toggleActive(c: City) {
    setError("");
    try {
      await api(`/api/v1/admin/cities/${c.id}`, {
        method: "PUT",
        body: JSON.stringify({
          name: c.name,
          lat: c.lat,
          lng: c.lng,
          radius_m: c.radius_m,
          max_delivery_m: c.max_delivery_m,
          sort_order: c.sort_order,
          active: !c.active,
        }),
      });
      await load();
    } catch (err) {
      setError(errorText(err));
    }
  }

  async function remove(c: City) {
    setPendingDelete(null);
    try {
      await api(`/api/v1/admin/cities/${c.id}`, { method: "DELETE" });
      setSelectedID(null);
      setDraft(null);
      await load();
    } catch (err) {
      setError(errorText(err));
    }
  }

  return (
    <div className="flex h-[calc(100vh-6rem)] flex-col lg:h-[calc(100vh-3rem)]">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <PageHeader icon={IconZones} title={C.title} subtitle={C.hint} />
        {isAdmin && !draft && (
          <Button onClick={startCreate} className="flex items-center gap-1.5">
            <IconAdd size={16} />
            {C.newCity}
          </Button>
        )}
      </div>

      {draft && (
        <p className="mb-3 rounded-control bg-accent-tint px-3 py-2 text-sm text-accent-dark">
          {C.centerHint}
        </p>
      )}
      {error && <Alert className="mb-3">{error}</Alert>}

      <div className="flex min-h-0 flex-1 flex-col gap-4 md:flex-row">
        <div className="min-h-64 flex-1 overflow-hidden rounded-card border border-line">
          <CitiesMap
            zones={cities.map((c) => ({
              id: c.id,
              name: c.name,
              lat: c.lat,
              lng: c.lng,
              radius_m: c.radius_m,
              active: c.active,
            }))}
            editing={!!draft}
            draft={
              draft && draft.lat != null
                ? { lat: draft.lat, lng: draft.lng!, radiusM: draft.radiusM }
                : null
            }
            selectedID={selectedID}
            onMapClick={(lat, lng) => setDraft((d) => (d ? { ...d, lat, lng } : d))}
            onZoneClick={(id) => {
              const c = cities.find((x) => x.id === id);
              if (c && isAdmin && !draft) startEdit(c);
              else setSelectedID(id);
            }}
          />
        </div>

        <aside className="w-full shrink-0 space-y-3 overflow-y-auto md:w-80">
          {draft && (
            <div className="space-y-3 surface !border-accent p-4">
              <div className="flex items-center justify-between">
                <h2 className="font-bold">{draft.id ? C.editCity : C.newCity}</h2>
                <button onClick={() => setDraft(null)} className="text-ink-muted hover:text-ink">
                  <IconClose size={18} />
                </button>
              </div>

              <Input
                id="c-name"
                label={C.cityName}
                required
                value={draft.name}
                onChange={(e) => setDraft({ ...draft, name: e.target.value })}
                placeholder={C.cityNamePlaceholder}
              />

              {/* **نصفُ قطرِ المدينة — من وقع داخله فهو من أهلها.**

                  **وهو ليس مدى التوصيل**: يقول «هذه دمشق»، **والمدى يقول
                  وإلى أين نُوصّل فيها.** ومن خلطهما فتح الغوطةَ لمطعمٍ في
                  المزّة. */}
              <div>
                <div className="mb-1 flex items-center justify-between text-sm">
                  <span className="font-medium">{C.radius}</span>
                  <Badge variant="primary">
                    {(draft.radiusM / 1000).toFixed(0)} {C.km}
                  </Badge>
                </div>
                <input
                  type="range"
                  min={5000}
                  max={80000}
                  step={1000}
                  value={draft.radiusM}
                  onChange={(e) => setDraft({ ...draft, radiusM: Number(e.target.value) })}
                  className="w-full accent-primary"
                />
                <p className="mt-1 text-xs text-ink-muted">{C.radiusHint}</p>
              </div>

              {/* **ومدى التوصيل داخلها — فارغٌ يعني افتراضَ المنصّة.**

                  **والمدينةُ الصغيرةُ تُترك فارغةً فتغطّيها كلَّها** —
                  والرقّةُ لا تحتاج شيئاً. */}
              <Input
                id="c-reach"
                type="number"
                label={C.reach}
                value={draft.reachM}
                onChange={(e) => setDraft({ ...draft, reachM: e.target.value })}
                placeholder={C.reachPlaceholder}
              />
              <p className="-mt-2 text-xs text-ink-muted">{C.reachHint}</p>

              <Input
                id="c-sort"
                type="number"
                label={m.admin.sections.sort}
                value={draft.sortOrder}
                onChange={(e) => setDraft({ ...draft, sortOrder: e.target.value })}
              />

              {draft.lat == null && <Alert tone="warning">{C.centerUnset}</Alert>}

              <Button
                onClick={() => void save()}
                disabled={busy || draft.lat == null || !draft.name.trim()}
                className="w-full"
              >
                {m.common.save}
              </Button>
            </div>
          )}

          {cities.length === 0 && !draft && (
            <p className="surface p-6 text-center text-sm text-ink-muted">{C.empty}</p>
          )}

          {cities.map((c) => (
            <button
              key={c.id}
              onClick={() => (isAdmin ? startEdit(c) : setSelectedID(c.id))}
              className={`w-full rounded-card border p-3 text-start transition-colors ${
                c.id === selectedID
                  ? "border-accent bg-accent-tint"
                  : "border-line bg-surface hover:border-primary-edge"
              }`}
            >
              <div className="flex items-center justify-between">
                <span className="font-bold">{c.name}</span>
                <Badge variant={c.active ? "success" : "danger"}>
                  {c.active ? m.admin.zones.active : m.admin.zones.inactive}
                </Badge>
              </div>
              <p className="mt-1 text-sm text-ink-muted">
                {(c.radius_m / 1000).toFixed(0)} {C.km}
                {" · "}
                {C.merchants.replace("{n}", fmtNum(c.merchants))}
              </p>
              {/* **والمدى يُقال نصّاً ولا يُترك للحدس** — «افتراضُ المنصّة»
                  جوابٌ، **وفراغٌ سؤال.** */}
              <p className="mt-0.5 text-xs text-ink-muted">
                {c.max_delivery_m == null
                  ? C.reachDefault
                  : C.reachIs.replace("{n}", fmtNum(c.max_delivery_m))}
              </p>

              {c.id === selectedID && isAdmin && (
                <div className="mt-2 flex gap-2 border-t border-line-soft pt-2">
                  <Button
                    variant={c.active ? "danger" : "secondary"}
                    onClick={(e) => {
                      e.stopPropagation();
                      void toggleActive(c);
                    }}
                  >
                    {c.active ? m.admin.merchants.deactivate : m.admin.merchants.activate}
                  </Button>
                  {/* **ولا يُعرض زرُّ الحذف لمدينةٍ فيها متاجر** — **وزرٌّ
                      يُضغط فيُردَّ دائماً يُقرأ عطباً**، والصوابُ أن
                      يُطفئها. */}
                  {c.merchants === 0 && (
                    <Button
                      variant="ghost"
                      onClick={(e) => {
                        e.stopPropagation();
                        setPendingDelete(c);
                      }}
                    >
                      <IconDelete size={15} className="text-danger" />
                    </Button>
                  )}
                </div>
              )}
            </button>
          ))}
        </aside>
      </div>

      <Confirm
        open={!!pendingDelete}
        title={C.deleteConfirm}
        body={pendingDelete?.name}
        confirmLabel={m.common.delete}
        onConfirm={() => pendingDelete && void remove(pendingDelete)}
        onCancel={() => setPendingDelete(null)}
      />
    </div>
  );
}
