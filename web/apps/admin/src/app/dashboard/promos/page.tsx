"use client";

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDate } from "@rahalgo/i18n";
import {
  Tabs,
  Alert,
  PageHeader,
  Button,
  Input,
  Select,
  Badge,
  Modal,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  IconPromos,
  IconAdd,
  IconStatus,
  IconWallet,
  IconDate,
  IconDelete,
  IconEdit,
  Checkbox,
} from "@rahalgo/ui";
import { api, ApiError, mediaUrl } from "@/lib/api";
import ImageUpload from "@/components/ImageUpload";
import { useAuth } from "@/lib/auth";
import DiscountsTab from "@/components/DiscountsTab";

const m = getMessages(defaultLocale);

interface Promo {
  id: string;
  code: string;
  kind: "percent" | "fixed" | "free_delivery";
  value: number;
  min_order: number;
  first_order_only: boolean;
  once_per_user: boolean;
  max_uses: number | null;
  used_count: number;
  expires_at: string | null;
  active: boolean;
}

interface Banner {
  id: string;
  title: string;
  image_url: string | null;
  image_thumb_url: string | null;
  target: string;
  sort_order: number;
  active: boolean;
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

const KIND_LABEL: Record<Promo["kind"], string> = {
  percent: m.admin.promos.kindPercent,
  fixed: m.admin.promos.kindFixed,
  free_delivery: m.admin.promos.kindFreeDelivery,
};

export default function PromosPage() {
  const { user: me } = useAuth();
  const isAdmin = !!me?.roles.includes("admin");
  const [tab, setTab] = useState<"codes" | "discounts">("codes");

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <PageHeader icon={IconPromos} title={m.admin.promos.title} />
        {/* **وكانت حبّاتٍ ممتلئةً في صندوق** — والتبويبُ ليس فعلاً بل
            موضعٌ أنت فيه، **والصندوقُ الممتلئُ يُقرأ زرّاً.** */}
        <Tabs
          items={[
            { key: "codes" as const, label: m.admin.promos.tabCodes },
            { key: "discounts" as const, label: m.admin.promos.tabDiscounts },
          ]}
          value={tab}
          onChange={setTab}
        />
      </div>
      {/* **ثلاثةُ تبويباتٍ في صفحةٍ واحدة.**

          كودٌ يُكتب · ولافتةٌ تُرى · وخصمٌ يُطبَّق في الدفتر — **ثلاثةُ أشكالٍ
          لغرضٍ واحد**، وشاشتان لهما تجعلان من يبحث عن عرضٍ يفتح الاثنتين.
          (قرارُ المالك ٢٠٢٦-٠٨-٠٥.) */}
      {tab === "codes" ? (
        <CodesTab isAdmin={isAdmin} />
      ) : (
        <DiscountsTab />
      )}
    </div>
  );
}

function CodesTab({ isAdmin }: { isAdmin: boolean }) {
  const [promos, setPromos] = useState<Promo[]>([]);
  const [error, setError] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [view, setView] = useViewMode("promos");

  const load = useCallback(async () => {
    try {
      setPromos(await api<Promo[]>("/api/v1/admin/promos"));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function toggleActive(p: Promo) {
    try {
      await api(`/api/v1/admin/promos/${p.id}`, {
        method: "PATCH",
        body: JSON.stringify({ active: !p.active }),
      });
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  const columns: DataColumn<Promo>[] = [
    {
      id: "code",
      header: m.admin.promos.code,
      icon: <IconPromos />,
      primary: true,
      cell: (p) => (
        <span dir="ltr" className="font-mono font-bold tracking-wider">
          {p.code}
        </span>
      ),
    },
    {
      id: "kind",
      header: m.admin.promos.kind,
      cell: (p) => (
        <Badge variant="primary">
          {KIND_LABEL[p.kind]}
          {p.kind === "percent" && ` ${p.value}%`}
          {p.kind === "fixed" && ` ${fmtNum(p.value)}`}
        </Badge>
      ),
    },
    {
      id: "min",
      header: m.admin.promos.minOrder,
      icon: <IconWallet />,
      cell: (p) => `${fmtNum(p.min_order)} ${m.common.currency}`,
    },
    {
      id: "uses",
      header: m.admin.promos.usedCount,
      cell: (p) => `${p.used_count}${p.max_uses ? ` / ${p.max_uses}` : ""}`,
    },
    {
      id: "expiry",
      header: m.admin.promos.expiresAt,
      icon: <IconDate />,
      cell: (p) =>
        p.expires_at ? (
          fmtDate(p.expires_at)
        ) : (
          <span className="text-ink-muted">{m.admin.promos.noExpiry}</span>
        ),
    },
    {
      id: "status",
      header: m.admin.users.table.status,
      icon: <IconStatus />,
      cell: (p) => (
        <Badge variant={p.active ? "success" : "danger"}>
          {p.active ? m.admin.merchants.active : m.admin.merchants.inactive}
        </Badge>
      ),
    },
  ];

  return (
    <div>
      <div className="mb-4 flex items-center justify-between gap-3">
        {isAdmin ? (
          <Button onClick={() => setCreateOpen(true)} className="flex items-center gap-1.5">
            <IconAdd size={16} />
            {m.admin.promos.create}
          </Button>
        ) : (
          <span />
        )}
        <ViewToggle
          view={view}
          onChange={setView}
          tableLabel={m.common.viewTable}
          cardsLabel={m.common.viewCards}
        />
      </div>
      {error && (
        <Alert className="mb-4">{error}</Alert>
      )}
      <DataView
        items={promos}
        getKey={(p) => p.id}
        columns={columns}
        view={view}
        empty={m.admin.promos.empty}
        actions={
          isAdmin
            ? (p) => (
                <Button
                  variant={p.active ? "danger" : "secondary"}
                  onClick={() => toggleActive(p)}
                >
                  {p.active ? m.admin.merchants.deactivate : m.admin.merchants.activate}
                </Button>
              )
            : undefined
        }
      />
      {createOpen && (
        <PromoModal
          onClose={() => setCreateOpen(false)}
          onSaved={() => {
            setCreateOpen(false);
            void load();
          }}
        />
      )}
    </div>
  );
}

function PromoModal({ onClose, onSaved }: { onClose: () => void; onSaved: () => void }) {
  const [code, setCode] = useState("");
  const [kind, setKind] = useState<Promo["kind"]>("percent");
  const [value, setValue] = useState("");
  const [minOrder, setMinOrder] = useState("0");
  const [maxUses, setMaxUses] = useState("");
  const [expiresAt, setExpiresAt] = useState("");
  const [firstOnly, setFirstOnly] = useState(false);
  const [oncePerUser, setOncePerUser] = useState(true);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/admin/promos", {
        method: "POST",
        body: JSON.stringify({
          code,
          kind,
          value: Number(value) || 0,
          min_order: Number(minOrder) || 0,
          max_uses: maxUses === "" ? null : Number(maxUses),
          expires_at: expiresAt === "" ? null : new Date(expiresAt).toISOString(),
          first_order_only: firstOnly,
          once_per_user: oncePerUser,
        }),
      });
      onSaved();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={m.admin.promos.createTitle}>
      <form onSubmit={submit} className="space-y-4">
        <Input
          id="p-code"
          label={m.admin.promos.code}
          dir="ltr"
          required
          value={code}
          onChange={(e) => setCode(e.target.value.toUpperCase())}
          className="font-mono uppercase"
          placeholder="WELCOME50"
        />
        <div className="grid grid-cols-2 gap-3">
          <Select
            id="p-kind"
            label={m.admin.promos.kind}
            value={kind}
            onChange={(e) => setKind(e.target.value as Promo["kind"])}
          >
            <option value="percent">{m.admin.promos.kindPercent}</option>
            <option value="fixed">{m.admin.promos.kindFixed}</option>
            <option value="free_delivery">{m.admin.promos.kindFreeDelivery}</option>
          </Select>
          <Input
            id="p-value"
            label={m.admin.promos.value}
            type="number"
            min="0"
            disabled={kind === "free_delivery"}
            required={kind !== "free_delivery"}
            value={value}
            onChange={(e) => setValue(e.target.value)}
          />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <Input
            id="p-min"
            label={m.admin.promos.minOrder}
            type="number"
            min="0"
            value={minOrder}
            onChange={(e) => setMinOrder(e.target.value)}
          />
          <Input
            id="p-max"
            label={m.admin.promos.maxUses}
            type="number"
            min="1"
            value={maxUses}
            onChange={(e) => setMaxUses(e.target.value)}
          />
        </div>
        <Input
          id="p-exp"
          label={m.admin.promos.expiresAt}
          type="date"
          value={expiresAt}
          onChange={(e) => setExpiresAt(e.target.value)}
        />
        <div className="flex gap-4">
          <Checkbox
            id="promo-first-only"
            checked={firstOnly}
            onChange={(e) => setFirstOnly(e.target.checked)}
            label={m.admin.promos.firstOrderOnly}
          />
          <Checkbox
            id="promo-once-per-user"
            checked={oncePerUser}
            onChange={(e) => setOncePerUser(e.target.checked)}
            label={m.admin.promos.oncePerUser}
          />
        </div>
        {error && (
          <Alert>{error}</Alert>
        )}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button type="submit" disabled={busy}>
            {m.common.save}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
