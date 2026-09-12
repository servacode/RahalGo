"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **المحافظاتُ ومناطقُها — تُدار من اللوحة لا بهجرة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٣٠: «يجب أن يكون بلوحة الأدمن خيارٌ لإضافة
 *  المحافظات والمناطق وتعديلها وإيقافها وتفعيلها».)
 *
 * # ولماذا تبويبٌ في الإعدادات لا صفحةٌ مستقلّة
 *
 * **المدنُ والمناطقُ هناك أصلاً** — والتغطيةُ الجغرافيّةُ شيءٌ واحدٌ
 * يُقرأ معاً. **وصفحةٌ رابعةٌ في القائمة الجانبيّة تُفرّق ما يُفهم
 * مجتمعاً**، ومن بحث عن «أين نعمل» فتح ثلاثةَ أبواب.
 *
 * # ولوحان لا واحد
 *
 * **المحافظةُ أمٌّ والمنطقةُ بنت** — وجدولٌ واحدٌ يخلطهما يجعل «الترتيب»
 * عموداً لا يُقرأ: **أهو ترتيبُ المحافظة أم ترتيبُ المنطقة داخلها؟**
 *
 * # واختيارُ محافظةٍ يُصفّي مناطقَها
 *
 * **وثلاثٌ وستّون منطقةً في لوحٍ واحدٍ لا تُقرأ** — وقد تصير مئتين.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Input,
  Select,
  Modal,
  Badge,
  IconAdd,
  IconDelete,
  Confirm,
  FormActions,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const D = m.admin.divisions;

interface Governorate {
  id: string;
  name: string;
  active: boolean;
  sort_order: number;
  districts?: number;
}

interface District {
  id: string;
  name: string;
  governorate_id: string;
  governorate_name?: string;
  active: boolean;
  sort_order: number;
  cities?: number;
}

export default function DivisionsPanel() {
  const { user: me, can } = useAuth();
  // **ورسمُ المحافظات والنواحي جغرافيا** — `settings.general.manage`.
  const isAdmin = can("settings.general.manage");

  const [govs, setGovs] = useState<Governorate[]>([]);
  const [districts, setDistricts] = useState<District[]>([]);
  // **والمحافظةُ المختارةُ تُصفّي اللوحَ الثاني** — وفارغٌ يعني الكلّ.
  const [pick, setPick] = useState("");
  const [error, setError] = useState("");

  const [govDraft, setGovDraft] = useState<Governorate | null>(null);
  const [distDraft, setDistDraft] = useState<District | null>(null);
  const [confirmGov, setConfirmGov] = useState<Governorate | null>(null);
  const [confirmDist, setConfirmDist] = useState<District | null>(null);

  const load = useCallback(async () => {
    try {
      const g = await api<{ governorates: Governorate[] }>(
        "/api/v1/admin/governorates",
      );
      setGovs(g.governorates ?? []);
      const q = pick ? `?governorate_id=${pick}` : "";
      const d = await api<{ districts: District[] }>(
        `/api/v1/admin/districts${q}`,
      );
      setDistricts(d.districts ?? []);
      setError("");
    } catch (err) {
      setError(errorText(err));
    }
  }, [pick]);

  useEffect(() => {
    void load();
  }, [load]);

  // **والإطفاءُ فعلٌ واحدٌ لا نموذج** — ويُرسَل ما لا يتبدّل كما هو،
  // **فحقلٌ لا يُرسَل يُقرأ فراغاً فيمحو الاسم.**
  async function toggleGov(g: Governorate) {
    try {
      await api(`/api/v1/admin/governorates/${g.id}`, {
        method: "PUT",
        body: JSON.stringify({
          name: g.name,
          active: !g.active,
          sort_order: g.sort_order,
        }),
      });
      await load();
    } catch (err) {
      setError(errorText(err));
    }
  }

  async function toggleDist(d: District) {
    try {
      await api(`/api/v1/admin/districts/${d.id}`, {
        method: "PUT",
        body: JSON.stringify({
          name: d.name,
          governorate_id: d.governorate_id,
          active: !d.active,
          sort_order: d.sort_order,
        }),
      });
      await load();
    } catch (err) {
      setError(errorText(err));
    }
  }

  async function removeGov(g: Governorate) {
    try {
      await api(`/api/v1/admin/governorates/${g.id}`, { method: "DELETE" });
      setConfirmGov(null);
      await load();
    } catch (err) {
      setError(errorText(err));
      setConfirmGov(null);
    }
  }

  async function removeDist(d: District) {
    try {
      await api(`/api/v1/admin/districts/${d.id}`, { method: "DELETE" });
      setConfirmDist(null);
      await load();
    } catch (err) {
      setError(errorText(err));
      setConfirmDist(null);
    }
  }

  return (
    <div className="space-y-8">
      <p className="text-sm text-ink-muted">{D.hint}</p>
      {error && <Alert>{error}</Alert>}

      {/* ══════════════════ المحافظات ══════════════════ */}
      <section>
        <div className="mb-3 flex items-center justify-between gap-3">
          <h3 className="font-bold">{D.governorates}</h3>
          {isAdmin && (
            <Button
              onClick={() =>
                setGovDraft({ id: "", name: "", active: true, sort_order: 0 })
              }
              className="flex items-center gap-1.5"
            >
              <IconAdd size={16} />
              {D.newGovernorate}
            </Button>
          )}
        </div>
        {govs.length === 0 ? (
          <p className="text-sm text-ink-muted">{D.emptyGovernorates}</p>
        ) : (
          <ul className="divide-y divide-line rounded-card border border-line">
            {govs.map((g) => (
              <li
                key={g.id}
                className="flex flex-wrap items-center justify-between gap-3 p-3"
              >
                <div className="flex items-center gap-2">
                  <span className="font-medium">{g.name}</span>
                  <Badge variant={g.active ? "success" : "danger"}>
                    {g.active
                      ? m.admin.merchants.active
                      : m.admin.merchants.inactive}
                  </Badge>
                  <span className="text-sm text-ink-muted">
                    {D.districtCount.replace("{n}", String(g.districts ?? 0))}
                  </span>
                </div>
                {isAdmin && (
                  <div className="flex items-center gap-2">
                    <Button variant="secondary" onClick={() => setGovDraft(g)}>
                      {m.common.edit}
                    </Button>
                    <Button
                      variant={g.active ? "danger" : "secondary"}
                      onClick={() => void toggleGov(g)}
                    >
                      {g.active
                        ? m.admin.merchants.deactivate
                        : m.admin.merchants.activate}
                    </Button>
                    {/* **والحذفُ لمن لا مناطقَ تحته** — والمحرّكُ يردّ
                        السببَ بالعربيّة إن كان تحته شيء. */}
                    <Button variant="danger" onClick={() => setConfirmGov(g)}>
                      <IconDelete size={16} />
                    </Button>
                  </div>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* ══════════════════ المناطق ══════════════════ */}
      <section>
        <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
          <h3 className="font-bold">{D.districts}</h3>
          <div className="flex items-center gap-2">
            <Select
              value={pick}
              onChange={(e) => setPick(e.target.value)}
              aria-label={D.governorate}
            >
              <option value="">{D.allGovernorates}</option>
              {govs.map((g) => (
                <option key={g.id} value={g.id}>
                  {g.name}
                </option>
              ))}
            </Select>
            {isAdmin && (
              <Button
                onClick={() =>
                  setDistDraft({
                    id: "",
                    name: "",
                    governorate_id: pick || govs[0]?.id || "",
                    active: true,
                    sort_order: 0,
                  })
                }
                className="flex items-center gap-1.5"
              >
                <IconAdd size={16} />
                {D.newDistrict}
              </Button>
            )}
          </div>
        </div>
        {districts.length === 0 ? (
          <p className="text-sm text-ink-muted">{D.emptyDistricts}</p>
        ) : (
          <ul className="divide-y divide-line rounded-card border border-line">
            {districts.map((d) => (
              <li
                key={d.id}
                className="flex flex-wrap items-center justify-between gap-3 p-3"
              >
                <div className="flex items-center gap-2">
                  <span className="font-medium">{d.name}</span>
                  {!pick && d.governorate_name && (
                    <span className="text-sm text-ink-muted">
                      {d.governorate_name}
                    </span>
                  )}
                  <Badge variant={d.active ? "success" : "danger"}>
                    {d.active
                      ? m.admin.merchants.active
                      : m.admin.merchants.inactive}
                  </Badge>
                  <span className="text-sm text-ink-muted">
                    {d.cities
                      ? D.cityCount.replace("{n}", String(d.cities))
                      : D.noCities}
                  </span>
                </div>
                {isAdmin && (
                  <div className="flex items-center gap-2">
                    <Button variant="secondary" onClick={() => setDistDraft(d)}>
                      {m.common.edit}
                    </Button>
                    <Button
                      variant={d.active ? "danger" : "secondary"}
                      onClick={() => void toggleDist(d)}
                    >
                      {d.active
                        ? m.admin.merchants.deactivate
                        : m.admin.merchants.activate}
                    </Button>
                    <Button variant="danger" onClick={() => setConfirmDist(d)}>
                      <IconDelete size={16} />
                    </Button>
                  </div>
                )}
              </li>
            ))}
          </ul>
        )}
        <p className="mt-2 text-xs text-ink-muted">{D.offHint}</p>
      </section>

      {govDraft && (
        <GovModal
          draft={govDraft}
          onClose={() => setGovDraft(null)}
          onSaved={() => {
            setGovDraft(null);
            void load();
          }}
        />
      )}
      {distDraft && (
        <DistModal
          draft={distDraft}
          govs={govs}
          onClose={() => setDistDraft(null)}
          onSaved={() => {
            setDistDraft(null);
            void load();
          }}
        />
      )}
      {confirmGov && (
        <Confirm
          open
          title={D.deleteGovernorate}
          confirmLabel={m.common.delete}
          tone="danger"
          onConfirm={() => void removeGov(confirmGov)}
          onCancel={() => setConfirmGov(null)}
        />
      )}
      {confirmDist && (
        <Confirm
          open
          title={D.deleteDistrict}
          confirmLabel={m.common.delete}
          tone="danger"
          onConfirm={() => void removeDist(confirmDist)}
          onCancel={() => setConfirmDist(null)}
        />
      )}
    </div>
  );
}

function GovModal({
  draft,
  onClose,
  onSaved,
}: {
  draft: Governorate;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(draft.name);
  const [sort, setSort] = useState(String(draft.sort_order));
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const body = JSON.stringify({
        name,
        active: draft.id ? draft.active : true,
        sort_order: Number(sort) || 0,
      });
      if (draft.id) {
        await api(`/api/v1/admin/governorates/${draft.id}`, {
          method: "PUT",
          body,
        });
      } else {
        await api("/api/v1/admin/governorates", { method: "POST", body });
      }
      onSaved();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      title={draft.id ? D.editGovernorate : D.newGovernorate}
    >
      <form onSubmit={submit} className="space-y-4">
        {error && <Alert>{error}</Alert>}
        <Input
          label={D.name}
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
        />
        <Input
          label={D.sort}
          type="number"
          value={sort}
          onChange={(e) => setSort(e.target.value)}
        />
        <FormActions busy={busy} onCancel={onClose} />
      </form>
    </Modal>
  );
}

function DistModal({
  draft,
  govs,
  onClose,
  onSaved,
}: {
  draft: District;
  govs: Governorate[];
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(draft.name);
  const [gov, setGov] = useState(draft.governorate_id);
  const [sort, setSort] = useState(String(draft.sort_order));
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const body = JSON.stringify({
        name,
        governorate_id: gov,
        active: draft.id ? draft.active : true,
        sort_order: Number(sort) || 0,
      });
      if (draft.id) {
        await api(`/api/v1/admin/districts/${draft.id}`, {
          method: "PUT",
          body,
        });
      } else {
        await api("/api/v1/admin/districts", { method: "POST", body });
      }
      onSaved();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      title={draft.id ? D.editDistrict : D.newDistrict}
    >
      <form onSubmit={submit} className="space-y-4">
        {error && <Alert>{error}</Alert>}
        <Input
          label={D.name}
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
        />
        <Select
          label={D.governorate}
          value={gov}
          onChange={(e) => setGov(e.target.value)}
          required
        >
          <option value="">{D.pickGovernorate}</option>
          {govs.map((g) => (
            <option key={g.id} value={g.id}>
              {g.name}
            </option>
          ))}
        </Select>
        <Input
          label={D.sort}
          type="number"
          value={sort}
          onChange={(e) => setSort(e.target.value)}
        />
        <FormActions busy={busy} onCancel={onClose} />
      </form>
    </Modal>
  );
}
