"use client";

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  PageHeader, Button, Modal, IconSettings, IconEdit,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);

interface Setting {
  key: string;
  value: unknown;
  updated_at: string;
}

function errText(err: unknown): string {
  return err instanceof ApiError ? m.errors.internal : m.errors.internal;
}

export default function SettingsPage() {
  const { user: me } = useAuth();
  const isAdmin = !!me?.roles.includes("admin");
  const [settings, setSettings] = useState<Setting[]>([]);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState<Setting | null>(null);

  const load = useCallback(async () => {
    try {
      setSettings(await api<Setting[]>("/api/v1/admin/settings"));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div>
      <PageHeader icon={IconSettings} title={m.admin.settingsPage.title} />
      <p className="mb-6 text-sm text-ink-muted">{m.admin.settingsPage.hint}</p>

      {error && (
        <p className="mb-4 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      <div className="space-y-3">
        {settings.map((s) => (
          <div
            key={s.key}
            className="flex items-start justify-between gap-4 rounded-card border border-line bg-surface p-4"
          >
            <div className="min-w-0">
              <p dir="ltr" className="text-end font-mono text-sm font-bold text-primary-dark">
                {s.key}
              </p>
              <p className="mt-1 whitespace-pre-wrap break-words text-sm text-ink-muted">
                {typeof s.value === "string" ? s.value : JSON.stringify(s.value)}
              </p>
            </div>
            {isAdmin && (
              <Button
                variant="secondary"
                onClick={() => setEditing(s)}
                className="flex shrink-0 items-center gap-1.5"
              >
                <IconEdit size={15} />
                {m.admin.settingsPage.edit}
              </Button>
            )}
          </div>
        ))}
      </div>

      {editing && (
        <EditModal
          setting={editing}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            void load();
          }}
        />
      )}
    </div>
  );
}

function EditModal({
  setting,
  onClose,
  onSaved,
}: {
  setting: Setting;
  onClose: () => void;
  onSaved: () => void;
}) {
  const isString = typeof setting.value === "string";
  const [value, setValue] = useState(
    isString ? (setting.value as string) : JSON.stringify(setting.value, null, 2),
  );
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const parsed = isString ? value : JSON.parse(value);
      await api(`/api/v1/admin/settings/${setting.key}`, {
        method: "PUT",
        body: JSON.stringify({ value: parsed }),
      });
      onSaved();
    } catch (err) {
      setError(err instanceof SyntaxError ? m.errors.validation : errText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={`${m.admin.settingsPage.editTitle}: ${setting.key}`}>
      <form onSubmit={submit} className="space-y-4">
        <textarea
          value={value}
          onChange={(e) => setValue(e.target.value)}
          rows={6}
          className="w-full rounded-control border border-line bg-surface px-3 py-2 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary/20"
        />
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
