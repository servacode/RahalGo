"use client";

import { useCallback, useEffect, useState } from "react";
import dynamic from "next/dynamic";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
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
import type { ZoneShape } from "./ZonesMap";

const ZonesMap = dynamic(() => import("./ZonesMap"), { ssr: false });

const m = getMessages(defaultLocale);

interface Zone extends ZoneShape {
  min_order: number;
  sort_order: number;
}

function translateKey(key: string): string {
  let node: unknown = m;
  for (const part of key.split(".")) {
    if (typeof node !== "object" || node === null) return m.errors.internal;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" ? node : m.errors.internal;
}
function errText(err: unknown): string {
  return err instanceof ApiError ? translateKey(err.body.message_key) : m.errors.internal;
}

interface Draft {
  id: string | null; // null = إنشاء جديد
  name: string;
  lat: number | null;
  lng: number | null;
  radiusM: number;
}

export default function ZonesPanel() {
  const { user: me } = useAuth();
  const isAdmin = !!me?.roles.includes("admin");

  const [zones, setZones] = useState<Zone[]>([]);
  const [error, setError] = useState("");
  const [draft, setDraft] = useState<Draft | null>(null);
  const [selectedID, setSelectedID] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  /** **المنطقةُ المرشَّحةُ للحذف** — تنتظر تأكيداً من نافذة المنصّة. */
  const [pendingDelete, setPendingDelete] = useState<Zone | null>(null);

  const load = useCallback(async () => {
    try {
      setZones(await api<Zone[]>("/api/v1/admin/zones"));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  function startCreate() {
    setSelectedID(null);
    setDraft({ id: null, name: "", lat: null, lng: null, radiusM: 2000 });
  }

  function startEdit(z: Zone) {
    setSelectedID(z.id);
    setDraft({
      id: z.id,
      name: z.name,
      lat: z.lat,
      lng: z.lng,
      radiusM: z.radius_m,
    });
  }

  async function save() {
    if (!draft || draft.lat == null) return;
    setBusy(true);
    setError("");
    const body = {
      name: draft.name,
      lat: draft.lat,
      lng: draft.lng,
      radius_m: draft.radiusM,
      // **والعمودُ يُكتب صفراً ولا يُقرأ** — بقي في القاعدة لتاريخٍ مضى،
      // **ولا يُحذف بترحيلٍ لأنّ حذفَ عمودٍ لا يُتراجع عنه.**
      delivery_fee: 0,
    };
    try {
      if (draft.id) {
        await api(`/api/v1/admin/zones/${draft.id}`, { method: "PATCH", body: JSON.stringify(body) });
      } else {
        await api("/api/v1/admin/zones", { method: "POST", body: JSON.stringify(body) });
      }
      setDraft(null);
      setSelectedID(null);
      await load();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  async function toggleActive(z: Zone) {
    try {
      await api(`/api/v1/admin/zones/${z.id}`, {
        method: "PATCH",
        body: JSON.stringify({ active: !z.active }),
      });
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  /* **وحذفُ منطقةٍ يُسأل عنه بنافذة المنصّة** — و`confirm()` الأصليّةُ نافذةُ
     نظامٍ بخطّه ولغته وأزرارِها التي لا تقول فعلَها. (شكوى المالك
     ٢٠٢٦-٠٨-١٠.) */
  async function deleteZone(z: Zone) {
    setPendingDelete(null);
    try {
      await api(`/api/v1/admin/zones/${z.id}`, { method: "DELETE" });
      setSelectedID(null);
      setDraft(null);
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  return (
    <div className="flex h-[calc(100vh-6rem)] flex-col lg:h-[calc(100vh-3rem)]">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <PageHeader icon={IconZones} title={m.admin.zones.title} />
        {isAdmin && !draft && (
          <Button onClick={startCreate} className="flex items-center gap-1.5">
            <IconAdd size={16} />
            {m.admin.zones.newZone}
          </Button>
        )}
      </div>

      {draft && (
        <p className="mb-3 rounded-control bg-accent-tint px-3 py-2 text-sm text-accent-dark">
          {m.admin.zones.centerHint}
        </p>
      )}
      {error && (
        <Alert className="mb-3">{error}</Alert>
      )}

      <div className="flex min-h-0 flex-1 flex-col gap-4 md:flex-row">
        <div className="min-h-64 flex-1 overflow-hidden rounded-card border border-line">
          <ZonesMap
            zones={zones}
            editing={!!draft}
            draft={draft && draft.lat != null ? { lat: draft.lat, lng: draft.lng!, radiusM: draft.radiusM } : null}
            selectedID={selectedID}
            onMapClick={(lat, lng) => setDraft((d) => (d ? { ...d, lat, lng } : d))}
            onZoneClick={(id) => {
              const z = zones.find((x) => x.id === id);
              if (z && isAdmin && !draft) startEdit(z);
              else setSelectedID(id);
            }}
          />
        </div>

        <aside className="w-full shrink-0 space-y-3 overflow-y-auto md:w-80">
          {/* نموذج الإنشاء/التعديل */}
          {draft && (
            <div className="space-y-3 surface !border-accent p-4">
              <div className="flex items-center justify-between">
                <h2 className="font-bold">
                  {draft.id ? m.admin.zones.editZone : m.admin.zones.newZone}
                </h2>
                <button onClick={() => setDraft(null)} className="text-ink-muted hover:text-ink">
                  <IconClose size={18} />
                </button>
              </div>
              <Input
                id="z-name"
                label={m.admin.zones.zoneName}
                required
                value={draft.name}
                onChange={(e) => setDraft({ ...draft, name: e.target.value })}
                placeholder={m.admin.zones.zoneNamePlaceholder}
              />
              <div>
                <div className="mb-1 flex items-center justify-between text-sm">
                  <span className="font-medium">{m.admin.zones.radius}</span>
                  <Badge variant="primary">
                    {(draft.radiusM / 1000).toFixed(1)} {m.admin.zones.km}
                  </Badge>
                </div>
                <input
                  type="range"
                  min={500}
                  max={15000}
                  step={100}
                  value={draft.radiusM}
                  onChange={(e) => setDraft({ ...draft, radiusM: Number(e.target.value) })}
                  className="w-full accent-primary"
                />
              </div>
              {/* **ولا حقلَ أجرةٍ هنا** — المنطقةُ تغطيةٌ لا تسعير.

                  كانت لكلّ دائرةٍ أجرتُها. **وصارت الأجرةُ رقماً مقطوعاً
                  واحداً في الإعدادات** (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «رقمٌ
                  مقطوعٌ فقط، لا نسبة ولا مسافة ولا شيء») — فبقي الحقلُ
                  يُكتب ولا يُقرأ.

                  **وحقلٌ يكتب قيمةً لا يقرؤها أحدٌ زرٌّ كاذب**: يظنّ صاحبُه
                  أنّه ضبط شيئاً. وهي القاعدةُ التي حُذف بها حقلُ «الحدّ
                  الأدنى» من هنا قبله.

                  **والدائرةُ تقول «إلى أين نُوصّل» والرقمُ يقول «بكم»** —
                  ومن خلطهما فتح المدينةَ كلَّها بمجرّد أن وحّد الأجرة. */}
              {draft.lat == null && (
                <Alert tone="warning">
                  {m.admin.zones.centerUnset}
                </Alert>
              )}
              <Button
                onClick={save}
                disabled={busy || draft.lat == null || !draft.name}
                className="w-full"
              >
                {m.common.save}
              </Button>
            </div>
          )}

          {/* قائمة المناطق */}
          {zones.length === 0 && !draft && (
            <p className="surface p-6 text-center text-sm text-ink-muted">
              {m.admin.zones.empty}
            </p>
          )}
          {zones.map((z) => (
            <button
              key={z.id}
              onClick={() => (isAdmin ? startEdit(z) : setSelectedID(z.id))}
              className={`w-full rounded-card border p-3 text-start transition-colors ${
                z.id === selectedID
                  ? "border-accent bg-accent-tint"
                  : "border-line bg-surface hover:border-primary-edge"
              }`}
            >
              <div className="flex items-center justify-between">
                <span className="font-bold">{z.name}</span>
                <Badge variant={z.active ? "success" : "danger"}>
                  {z.active ? m.admin.zones.active : m.admin.zones.inactive}
                </Badge>
              </div>
              <p className="mt-1 flex items-center gap-1.5 text-sm text-ink-muted">
                {(z.radius_m / 1000).toFixed(1)} {m.admin.zones.km}
              </p>
              {z.id === selectedID && isAdmin && (
                <div className="mt-2 flex gap-2 border-t border-line-soft pt-2">
                  <Button
                    variant={z.active ? "danger" : "secondary"}
                    onClick={(e) => {
                      e.stopPropagation();
                      void toggleActive(z);
                    }}
                  >
                    {z.active ? m.admin.merchants.deactivate : m.admin.merchants.activate}
                  </Button>
                  <Button
                    variant="ghost"
                    onClick={(e) => {
                      e.stopPropagation();
                      setPendingDelete(z);
                    }}
                  >
                    <IconDelete size={15} className="text-danger" />
                  </Button>
                </div>
              )}
            </button>
          ))}
        </aside>
      </div>

      <Confirm
        open={!!pendingDelete}
        title={m.admin.zones.deleteConfirm}
        body={pendingDelete?.name}
        confirmLabel={m.common.delete}
        onConfirm={() => pendingDelete && void deleteZone(pendingDelete)}
        onCancel={() => setPendingDelete(null)}
      />
    </div>
  );
}
