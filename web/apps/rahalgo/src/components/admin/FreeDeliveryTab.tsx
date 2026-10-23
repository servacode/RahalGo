"use client";

/**
 * **«التوصيل المجاني»** — تبويبٌ في صفحة العروض (قرارُ المالك ٢٠٢٦-١٠-٢٣).
 *
 * مفتاحٌ وبدايةٌ ونهاية، وما يشمله العرض، وسقفٌ اختياريّ. **مرّةٌ لكلّ زبونٍ بالعرض**،
 * و«عرض جديد» يُرجع للجميع حقَّ توصيلة. والعدّادُ: كم انعطى وكم دفعت المنصّة.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtMoney, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Card,
  Checkbox,
  Input,
  StatGrid,
  StatCard,
  Switch,
  IconWallet,
  IconPromos,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const F = m.admin.promos.freeDelivery;

const KINDS = [
  { key: "standard", label: F.kindStandard },
  { key: "purchase", label: F.kindPurchase },
  { key: "amanah", label: F.kindAmanah },
  { key: "merchant", label: F.kindMerchant },
] as const;

interface Campaign {
  id: string;
  active: boolean;
  starts_at: string;
  ends_at: string;
  kinds: string[];
  max_fee: number | null;
  running: boolean;
  claims: number;
  waived: number;
}

/** **تاريخُ الحقل بتوقيت الجهاز** — `datetime-local` لا يقرأ المنطقة. */
function toLocal(iso: string): string {
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function blank() {
  const now = new Date();
  const week = new Date(now.getTime() + 7 * 24 * 3600 * 1000);
  return {
    active: false,
    starts: toLocal(now.toISOString()),
    ends: toLocal(week.toISOString()),
    kinds: ["standard"] as string[],
    maxFee: "",
  };
}

export default function FreeDeliveryTab() {
  const [campaign, setCampaign] = useState<Campaign | null>(null);
  const [form, setForm] = useState(blank);
  const [loaded, setLoaded] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);

  const fill = useCallback((c: Campaign | null) => {
    setCampaign(c);
    if (c) {
      setForm({
        active: c.active,
        starts: toLocal(c.starts_at),
        ends: toLocal(c.ends_at),
        kinds: c.kinds,
        maxFee: c.max_fee ? String(c.max_fee) : "",
      });
    }
  }, []);

  const load = useCallback(() => {
    api<{ campaign: Campaign | null }>("/api/v1/admin/free-delivery")
      .then((r) => fill(r.campaign))
      .catch((e) => setError(errorText(e)))
      .finally(() => setLoaded(true));
  }, [fill]);
  useEffect(load, [load]);

  async function save(asNew: boolean) {
    setBusy(true);
    setError("");
    setSaved(false);
    const body = {
      active: form.active,
      starts_at: new Date(form.starts).toISOString(),
      ends_at: new Date(form.ends).toISOString(),
      kinds: form.kinds,
      max_fee: form.maxFee.trim() ? Number(form.maxFee) : null,
    };
    try {
      const r =
        campaign && !asNew
          ? await api<{ campaign: Campaign }>(`/api/v1/admin/free-delivery/${campaign.id}`, {
              method: "PATCH",
              body: JSON.stringify(body),
            })
          : await api<{ campaign: Campaign }>("/api/v1/admin/free-delivery", { method: "POST", body: JSON.stringify(body) });
      fill(r.campaign);
      setSaved(true);
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  function toggleKind(k: string, on: boolean) {
    setForm((f) => ({ ...f, kinds: on ? [...f.kinds, k] : f.kinds.filter((x) => x !== k) }));
  }

  if (!loaded) return null;

  return (
    <div className="space-y-4">
      <p className="text-sm text-ink-muted">{F.hint}</p>
      {error && <Alert tone="error">{error}</Alert>}
      {saved && <Alert tone="success">{F.saved}</Alert>}

      {campaign && (
        <StatGrid>
          <StatCard icon={IconPromos} label={F.claims} value={fmtNum(campaign.claims)} />
          <StatCard icon={IconWallet} label={F.waived} value={fmtMoney(campaign.waived)} />
        </StatGrid>
      )}
      {!campaign && <Alert tone="info">{F.none}</Alert>}

      <Card>
        <div className="space-y-4">
          <Switch
            checked={form.active}
            onChange={(v) => setForm((f) => ({ ...f, active: v }))}
            label={F.active}
            hint={F.activeHint}
          />
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <Input
              type="datetime-local"
              label={F.startsAt}
              value={form.starts}
              onChange={(e) => setForm((f) => ({ ...f, starts: e.target.value }))}
            />
            <Input
              type="datetime-local"
              label={F.endsAt}
              value={form.ends}
              onChange={(e) => setForm((f) => ({ ...f, ends: e.target.value }))}
            />
          </div>
          <div>
            <p className="mb-2 text-sm font-bold">{F.kinds}</p>
            <div className="flex flex-col gap-2">
              {KINDS.map((k) => (
                <Checkbox
                  key={k.key}
                  id={`fd-${k.key}`}
                  label={k.label}
                  checked={form.kinds.includes(k.key)}
                  onChange={(e) => toggleKind(k.key, e.target.checked)}
                />
              ))}
            </div>
          </div>
          <div>
            <Input
              type="number"
              inputMode="numeric"
              min={1}
              label={F.maxFee}
              value={form.maxFee}
              onChange={(e) => setForm((f) => ({ ...f, maxFee: e.target.value }))}
            />
            <p className="mt-1 text-xs text-ink-muted">{F.maxFeeHint}</p>
          </div>
          {campaign && (
            <Badge variant={campaign.running ? "success" : "neutral"}>
              {campaign.running ? F.running : F.notRunning}
            </Badge>
          )}
          <div className="flex flex-wrap gap-2">
            <Button onClick={() => save(false)} disabled={busy}>
              {F.save}
            </Button>
            {campaign && (
              <Button variant="secondary" onClick={() => save(true)} disabled={busy}>
                {F.createNew}
              </Button>
            )}
          </div>
          {campaign && <p className="text-xs text-ink-muted">{F.createNewHint}</p>}
        </div>
      </Card>
    </div>
  );
}
