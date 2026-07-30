"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Button,
  Input,
  Select,
  Badge,
  Modal,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  IconStore,
  IconPhone,
  IconUser,
  IconStatus,
  IconSearch,
  IconAdd,
  IconEdit,
  IconLocation,
  IconSettings,
  IconOrder as IconMenu,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);

interface Category {
  id: string;
  name: string;
  icon: string;
  sort_order: number;
  active: boolean;
}

interface Merchant {
  id: string;
  name: string;
  description: string;
  category_id: string;
  category_name: string;
  category_icon: string;
  phone: string;
  address_text: string;
  owner_phone: string | null;
  status: string;
  created_at: string;
}

interface MerchantPage {
  merchants: Merchant[];
  total: number;
  page: number;
  per_page: number;
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

export default function MerchantsPage() {
  const { user: me } = useAuth();
  const router = useRouter();
  const isAdmin = !!me?.roles.includes("admin");

  const [data, setData] = useState<MerchantPage | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [query, setQuery] = useState("");
  const [category, setCategory] = useState("");
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState<Merchant | null | "new">(null);
  const [catsOpen, setCatsOpen] = useState(false);
  const [view, setView] = useViewMode("merchants", "cards");

  const loadCategories = useCallback(async () => {
    try {
      setCategories(await api<Category[]>("/api/v1/admin/categories"));
    } catch (err) {
      setError(errText(err));
    }
  }, []);

  const load = useCallback(async () => {
    try {
      const params = new URLSearchParams({
        query,
        category_id: category,
        status,
        page: String(page),
        per_page: "9",
      });
      setData(await api<MerchantPage>(`/api/v1/admin/merchants?${params}`));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [query, category, status, page]);

  useEffect(() => {
    void loadCategories();
  }, [loadCategories]);

  useEffect(() => {
    const t = setTimeout(load, 250);
    return () => clearTimeout(t);
  }, [load]);

  async function toggleStatus(mr: Merchant) {
    try {
      await api(`/api/v1/admin/merchants/${mr.id}`, {
        method: "PATCH",
        body: JSON.stringify({ status: mr.status === "active" ? "inactive" : "active" }),
      });
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  const totalPages = data ? Math.max(1, Math.ceil(data.total / data.per_page)) : 1;

  const columns: DataColumn<Merchant>[] = [
    {
      id: "name",
      header: m.admin.merchants.name,
      icon: <IconStore />,
      primary: true,
      cell: (mr) => (
        <span className="inline-flex items-center gap-1.5">
          <span>{mr.category_icon}</span>
          {mr.name}
        </span>
      ),
    },
    {
      id: "category",
      header: m.admin.merchants.category,
      cell: (mr) => <Badge variant="primary">{mr.category_name}</Badge>,
    },
    {
      id: "phone",
      header: m.admin.merchants.merchantPhone,
      icon: <IconPhone />,
      cell: (mr) =>
        mr.phone ? (
          <span dir="ltr" className="font-medium">
            {mr.phone}
          </span>
        ) : (
          "—"
        ),
    },
    {
      id: "address",
      header: m.admin.merchants.address,
      icon: <IconLocation />,
      cell: (mr) => <span className="text-ink-muted">{mr.address_text || "—"}</span>,
    },
    {
      id: "owner",
      header: m.admin.merchants.owner,
      icon: <IconUser />,
      cell: (mr) =>
        mr.owner_phone ? (
          <span dir="ltr">{mr.owner_phone}</span>
        ) : (
          <span className="text-ink-muted">{m.admin.merchants.noOwner}</span>
        ),
    },
    {
      id: "status",
      header: m.admin.merchants.status,
      icon: <IconStatus />,
      cell: (mr) => (
        <Badge variant={mr.status === "active" ? "success" : "danger"}>
          {mr.status === "active" ? m.admin.merchants.active : m.admin.merchants.inactive}
        </Badge>
      ),
    },
  ];

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold">{m.admin.merchants.title}</h1>
        {isAdmin && (
          <div className="flex gap-2">
            <Button
              variant="secondary"
              onClick={() => setCatsOpen(true)}
              className="flex items-center gap-1.5"
            >
              <IconSettings size={16} />
              {m.admin.merchants.manageCategories}
            </Button>
            <Button onClick={() => setEditing("new")} className="flex items-center gap-1.5">
              <IconAdd size={16} />
              {m.admin.merchants.create}
            </Button>
          </div>
        )}
      </div>

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="w-64">
          <Input
            icon={<IconSearch />}
            placeholder={m.admin.merchants.searchPlaceholder}
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setPage(1);
            }}
          />
        </div>
        <div className="w-44">
          <Select
            value={category}
            onChange={(e) => {
              setCategory(e.target.value);
              setPage(1);
            }}
          >
            <option value="">{m.admin.merchants.allCategories}</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.icon} {c.name}
              </option>
            ))}
          </Select>
        </div>
        <div className="w-36">
          <Select
            value={status}
            onChange={(e) => {
              setStatus(e.target.value);
              setPage(1);
            }}
          >
            <option value="">{m.admin.merchants.allStatuses}</option>
            <option value="active">{m.admin.merchants.active}</option>
            <option value="inactive">{m.admin.merchants.inactive}</option>
          </Select>
        </div>
        <div className="ms-auto">
          <ViewToggle
            view={view}
            onChange={setView}
            tableLabel={m.common.viewTable}
            cardsLabel={m.common.viewCards}
          />
        </div>
      </div>

      {error && (
        <p className="mb-4 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      <DataView
        items={data?.merchants ?? []}
        getKey={(mr) => mr.id}
        columns={columns}
        view={view}
        empty={m.admin.merchants.noResults}
        actions={
          isAdmin
            ? (mr) => (
                <>
                  <Button
                    variant="secondary"
                    onClick={() => router.push(`/dashboard/merchants/${mr.id}/menu`)}
                    className="flex items-center gap-1.5"
                  >
                    <IconMenu size={15} />
                    {m.admin.menu.manageMenu}
                  </Button>
                  <Button
                    variant="ghost"
                    onClick={() => setEditing(mr)}
                    className="flex items-center gap-1.5"
                  >
                    <IconEdit size={15} />
                    {m.admin.merchants.edit}
                  </Button>
                  <Button
                    variant={mr.status === "active" ? "danger" : "secondary"}
                    onClick={() => toggleStatus(mr)}
                  >
                    {mr.status === "active"
                      ? m.admin.merchants.deactivate
                      : m.admin.merchants.activate}
                  </Button>
                </>
              )
            : undefined
        }
      />

      {data && (
        <div className="mt-4 flex items-center justify-between text-sm text-ink-muted">
          <span>{m.admin.users.totalCount.replace("{count}", String(data.total))}</span>
          <div className="flex items-center gap-2">
            <Button variant="secondary" disabled={page <= 1} onClick={() => setPage(page - 1)}>
              {m.admin.users.prev}
            </Button>
            <span>
              {page} / {totalPages}
            </span>
            <Button
              variant="secondary"
              disabled={page >= totalPages}
              onClick={() => setPage(page + 1)}
            >
              {m.admin.users.next}
            </Button>
          </div>
        </div>
      )}

      {editing && (
        <MerchantModal
          merchant={editing === "new" ? null : editing}
          categories={categories}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null);
            void load();
          }}
        />
      )}
      <CategoriesModal
        open={catsOpen}
        categories={categories}
        onClose={() => setCatsOpen(false)}
        onChanged={loadCategories}
      />
    </div>
  );
}

function MerchantModal({
  merchant,
  categories,
  onClose,
  onSaved,
}: {
  merchant: Merchant | null;
  categories: Category[];
  onClose: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(merchant?.name ?? "");
  const [description, setDescription] = useState(merchant?.description ?? "");
  const [categoryId, setCategoryId] = useState(merchant?.category_id ?? categories[0]?.id ?? "");
  const [phone, setPhone] = useState(merchant?.phone ?? "");
  const [address, setAddress] = useState(merchant?.address_text ?? "");
  const [ownerPhone, setOwnerPhone] = useState(merchant?.owner_phone ?? "");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    const body = {
      name,
      description,
      category_id: categoryId,
      phone,
      address_text: address,
      owner_phone: ownerPhone,
    };
    try {
      if (merchant) {
        await api(`/api/v1/admin/merchants/${merchant.id}`, {
          method: "PATCH",
          body: JSON.stringify(body),
        });
      } else {
        await api("/api/v1/admin/merchants", { method: "POST", body: JSON.stringify(body) });
      }
      onSaved();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      title={merchant ? m.admin.merchants.editTitle : m.admin.merchants.createTitle}
    >
      <form onSubmit={submit} className="space-y-4">
        <Input
          id="m-name"
          label={m.admin.merchants.name}
          icon={<IconStore />}
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <Select
          id="m-cat"
          label={m.admin.merchants.category}
          value={categoryId}
          onChange={(e) => setCategoryId(e.target.value)}
        >
          {categories.map((c) => (
            <option key={c.id} value={c.id}>
              {c.icon} {c.name}
            </option>
          ))}
        </Select>
        <Input
          id="m-phone"
          label={m.admin.merchants.merchantPhone}
          icon={<IconPhone />}
          dir="ltr"
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          className="text-end"
        />
        <Input
          id="m-address"
          label={m.admin.merchants.address}
          icon={<IconLocation />}
          value={address}
          onChange={(e) => setAddress(e.target.value)}
          placeholder={m.admin.merchants.addressPlaceholder}
        />
        <Input
          id="m-desc"
          label={m.admin.merchants.descriptionField}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
        <div>
          <Input
            id="m-owner"
            label={m.admin.merchants.ownerPhone}
            icon={<IconUser />}
            dir="ltr"
            value={ownerPhone}
            onChange={(e) => setOwnerPhone(e.target.value)}
            className="text-end"
            placeholder="09xxxxxxxx"
          />
          <p className="mt-1 text-xs text-ink-muted">{m.admin.merchants.ownerHint}</p>
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

function CategoriesModal({
  open,
  categories,
  onClose,
  onChanged,
}: {
  open: boolean;
  categories: Category[];
  onClose: () => void;
  onChanged: () => Promise<void> | void;
}) {
  const [name, setName] = useState("");
  const [icon, setIcon] = useState("");
  const [error, setError] = useState("");

  async function addCategory(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await api("/api/v1/admin/categories", {
        method: "POST",
        body: JSON.stringify({ name, icon, sort_order: categories.length + 1 }),
      });
      setName("");
      setIcon("");
      await onChanged();
    } catch (err) {
      setError(errText(err));
    }
  }

  async function toggleActive(c: Category) {
    setError("");
    try {
      await api(`/api/v1/admin/categories/${c.id}`, {
        method: "PATCH",
        body: JSON.stringify({ active: !c.active }),
      });
      await onChanged();
    } catch (err) {
      setError(errText(err));
    }
  }

  return (
    <Modal open={open} onClose={onClose} title={m.admin.merchants.categoriesTitle}>
      <ul className="mb-4 space-y-2">
        {categories.map((c) => (
          <li
            key={c.id}
            className="flex items-center justify-between rounded-control border border-line px-3 py-2"
          >
            <span className={c.active ? "" : "text-ink-muted line-through"}>
              {c.icon} {c.name}
            </span>
            <Button variant={c.active ? "danger" : "secondary"} onClick={() => toggleActive(c)}>
              {c.active ? m.admin.merchants.deactivate : m.admin.merchants.activate}
            </Button>
          </li>
        ))}
      </ul>
      <form onSubmit={addCategory} className="flex items-end gap-2">
        <div className="flex-1">
          <Input
            id="cat-name"
            label={m.admin.merchants.categoryName}
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        <div className="w-24">
          <Input
            id="cat-icon"
            label={m.admin.merchants.categoryIcon}
            value={icon}
            onChange={(e) => setIcon(e.target.value)}
            placeholder="🍕"
          />
        </div>
        <Button type="submit" className="flex items-center gap-1">
          <IconAdd size={15} />
          {m.admin.merchants.addCategory}
        </Button>
      </form>
      {error && (
        <p className="mt-3 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}
    </Modal>
  );
}
