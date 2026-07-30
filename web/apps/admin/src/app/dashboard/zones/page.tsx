"use client";

import { useCallback, useEffect, useState } from "react";
import dynamic from "next/dynamic";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Button,
  Input,
  Badge,
  Modal,
  IconAdd,
  IconDelete,
  IconZones,
  IconWallet,
  IconCheck,
  IconClose,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import type { ZoneShape } from "./ZonesMap";

// Leaflet لا يعمل إلا في المتصفح
const ZonesMap = dynamic(() => import("./ZonesMap"), { ssr: false });

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");

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

export default function ZonesPage() {
  const { user: me } = useAuth();
  const isAdmin = !!me?.roles.includes("admin");

  const [zones, setZones] = useState<Zone[]>([]);
  const [error, setError] = useState("");
  const [drawing, setDrawing] = useState(false);
  const [draft, setDraft] = useState<[number, number][]>([]);
  const [saveOpen, setSaveOpen] = useState(false);
  const [selectedID, setSelectedID] = useState<string | null>(null);

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

  const selected = zones.find((z) => z.id === selectedID) ?? null;

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

  async function deleteZone(z: Zone) {
    if (!confirm(m.admin.zones.deleteConfirm)) return;
    try {
      await api(`/api/v1/admin/zones/${z.id}`, { method: "DELETE" });
      setSelectedID(null);
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  return (
    <div className="flex h-[calc(100vh-3rem)] flex-col">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <h1 className="flex items-center gap-2 text-2xl font-bold">
          <IconZones className="text-primary" />
          {m.admin.zones.title}
        </h1>
        {isAdmin && !drawing && (
          <Button
            onClick={() => {
              setDrawing(true);
              setDraft([]);
              setSelectedID(null);
            }}
            className="flex items-center gap-1.5"
          >
            <IconAdd size={16} />
            {m.admin.zones.newZone}
          </Button>
        )}
        {drawing && (
          <div className="flex items-center gap-2">
            <Badge variant="warning">
              {m.admin.zones.pointsCount.replace("{count}", String(draft.length))}
            </Badge>
            <Button
              disabled={draft.length < 3}
              onClick={() => setSaveOpen(true)}
              className="flex items-center gap-1.5"
            >
              <IconCheck size={15} />
              {m.admin.zones.finishDrawing}
            </Button>
            <Button
              variant="secondary"
              onClick={() => {
                setDrawing(false);
                setDraft([]);
              }}
              className="flex items-center gap-1.5"
            >
              <IconClose size={15} />
              {m.admin.zones.cancelDrawing}
            </Button>
          </div>
        )}
      </div>

      {drawing && (
        <p className="mb-3 rounded-control bg-accent/10 px-3 py-2 text-sm text-accent-dark">
          {m.admin.zones.drawing}
        </p>
      )}
      {error && (
        <p className="mb-3 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      <div className="flex min-h-0 flex-1 gap-4">
        <div className="flex-1 overflow-hidden rounded-card border border-line">
          <ZonesMap
            zones={zones}
            drawing={drawing}
            draft={draft}
            selectedID={selectedID}
            onMapClick={(lng, lat) => setDraft((d) => [...d, [lng, lat]])}
            onZoneClick={(id) => !drawing && setSelectedID(id)}
          />
        </div>

        <aside className="w-72 shrink-0 space-y-2 overflow-y-auto">
          {zones.length === 0 && (
            <p className="rounded-card border border-line bg-surface p-6 text-center text-sm text-ink-muted">
              {m.admin.zones.empty}
            </p>
          )}
          {zones.map((z) => (
            <button
              key={z.id}
              onClick={() => setSelectedID(z.id === selectedID ? null : z.id)}
              className={`w-full rounded-card border p-3 text-start transition-colors ${
                z.id === selectedID ? "border-accent bg-accent/5" : "border-line bg-surface hover:border-primary/40"
              }`}
            >
              <div className="flex items-center justify-between">
                <span className="font-bold">{z.name}</span>
                <Badge variant={z.active ? "success" : "danger"}>
                  {z.active ? m.admin.zones.active : m.admin.zones.inactive}
                </Badge>
              </div>
              <p className="mt-1 flex items-center gap-1.5 text-sm text-ink-muted">
                <IconWallet size={14} />
                {m.admin.zones.deliveryFee}: {fmt.format(z.delivery_fee)} {m.common.currency}
              </p>
              {z.id === selectedID && isAdmin && (
                <div className="mt-2 flex gap-2 border-t border-line pt-2">
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
                      void deleteZone(z);
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

      {saveOpen && (
        <SaveZoneModal
          draft={draft}
          onClose={() => setSaveOpen(false)}
          onSaved={() => {
            setSaveOpen(false);
            setDrawing(false);
            setDraft([]);
            void load();
          }}
        />
      )}
    </div>
  );
}

function SaveZoneModal({
  draft,
  onClose,
  onSaved,
}: {
  draft: [number, number][];
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState("");
  const [fee, setFee] = useState("");
  const [minOrder, setMinOrder] = useState("0");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/admin/zones", {
        method: "POST",
        body: JSON.stringify({
          name,
          polygon: draft,
          delivery_fee: Number(fee) || 0,
          min_order: Number(minOrder) || 0,
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
    <Modal open onClose={onClose} title={m.admin.zones.newZone}>
      <form onSubmit={submit} className="space-y-4">
        <Input
          id="z-name"
          label={m.admin.zones.zoneName}
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <div className="grid grid-cols-2 gap-3">
          <Input
            id="z-fee"
            label={`${m.admin.zones.deliveryFee} (${m.common.currency})`}
            type="number"
            min="0"
            required
            value={fee}
            onChange={(e) => setFee(e.target.value)}
          />
          <Input
            id="z-min"
            label={`${m.admin.zones.minOrder} (${m.common.currency})`}
            type="number"
            min="0"
            value={minOrder}
            onChange={(e) => setMinOrder(e.target.value)}
          />
        </div>
        {error && (
          <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
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
