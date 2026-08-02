"use client";

/**
 * تفاصيل عميل — شفافية العمولة.
 *
 * المندوب يقبض نسبةً من طلبات متجرٍ جلبه، ولم يكن يرى **من أين** جاءت: رقم
 * مجمَّع في لوحته وكفى. وثقةُ من يعمل بالعمولة تُبنى على أن يُراجع بنفسه لا على
 * أن يُصدّق. فهنا كل طلب: رقمه، تاريخه، حالته، قيمته، عمولة المنصة عليه،
 * ونصيبه منه.
 *
 * والطلب المُسترجَع **يبقى في الجدول** لا يُحذف سطره: إخفاؤه يجعل المجموع لا
 * يُطابَق، وهو نقيض الشفافية. ونصيبه يُعرض كما هو في الدفتر — صفراً إن عُكست
 * العمولة، وبقيمتها إن لم تُعكس بعد. الجدول يعرض **حقيقة القيود لا قاعدتها**،
 * فيكشف بذلك أي دَين معلّق بدل أن يُجمّله.
 */

import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtDate } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  Select,
  CategoryIcon,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  PageContainer,
  PageHeader,
  LoadingState,
  StatGrid,
  StatCard,
  useLiveRefresh,
  IconStore,
  IconPrev,
  IconNext,
  IconOrder,
  IconSuccess,
  IconError,
  IconWallet,
  IconDate,
  IconStatus,
  IconNote,
  IconBalance,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";

const m = getMessages(defaultLocale);
const D = m.rep.merchantDetail;
const CANCELLED = new Set(["rejected", "cancelled", "failed", "refunded"]);

/**
 * ثلاث دِلاء بدل أربع عشرة حالة.
 *
 * «السائق في المتجر» و«جارٍ إسناد سائق» تفاصيل تشغيل لا تعني المندوب ولا يملك
 * تغييرها، وعرضُها عليه يُثقل الشاشة بما لا يفيده. يعنيه سؤالان: أتمّ الطلب
 * فقبضتُ عمولته؟ أم ضاع؟ — وما بينهما «قيد التنفيذ» يُبقي المجاميع متطابقة
 * بلا أن يكشف عملياتٍ ليست شأنه.
 */
function bucketOf(status: string): {
  key: "delivered" | "cancelled" | "running";
  label: string;
  tone: "success" | "danger" | "warning";
} {
  if (status === "delivered") return { key: "delivered", label: D.stDelivered, tone: "success" };
  if (CANCELLED.has(status)) return { key: "cancelled", label: D.stCancelled, tone: "danger" };
  return { key: "running", label: D.stRunning, tone: "warning" };
}

interface RepOrder {
  number: number;
  status: string;
  cancel_reason: string;
  total: number;
  subtotal: number;
  delivery_fee: number;
  platform_commission: number;
  forfeited_commission: number;
  forfeited_share: number;
  my_share: number;
  created_at: string;
  delivered_at: string | null;
}

interface Detail {
  merchant: {
    name: string;
    category_icon: string;
    category_name: string;
    logo_thumb_url: string | null;
    status: string;
    joined_at: string;
    owner_phone: string | null;
  };
  summary: {
    orders: number;
    delivered: number;
    cancelled: number;
    delivered_sales: number;
    my_earnings: number;
  };
  orders: RepOrder[];
  total: number;
  page: number;
  per_page: number;
}

/**
 * رقمٌ يُعرض حيّاً أو مشطوباً حسب مصير الطلب.
 *
 * الدفتر يُصفّر عمولة الطلب الملغى — وهو الصواب المحاسبي. لكن عرض صفرٍ للمندوب
 * يخفي عنه **حجم ما ضاع**، وهو رقم يعنيه: يعرف به أي عميلٍ يُهدر جهده. فيُعرض
 * مشطوباً بجانبه وسمٌ صريح، فلا يُحسب ولا يُخفى.
 */
function Forfeitable({
  order,
  live,
  lost,
  highlight = false,
}: {
  order: RepOrder;
  live: number;
  lost: number;
  highlight?: boolean;
}) {
  if (bucketOf(order.status).key === "cancelled") {
    return (
      <span className="inline-flex items-center gap-1.5">
        <span dir="ltr" className="text-ink-muted line-through decoration-danger/70">
          {fmtNum(lost)}
        </span>
        <span className="rounded-badge bg-danger/10 px-1.5 py-0.5 text-2xs font-medium text-danger">
          {D.forfeited}
        </span>
      </span>
    );
  }
  // الصفر رمادي لا أخضر: تلوينه يَعِد بما ليس
  return (
    <span
      dir="ltr"
      className={highlight && live > 0 ? "font-bold text-success" : "text-ink-muted"}
    >
      {fmtNum(live)}
    </span>
  );
}

export default function MerchantDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [view, setView] = useViewMode("rep-merchant-orders");
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState("");
  const [data, setData] = useState<Detail | null>(null);
  const [missing, setMissing] = useState(false);

  const load = useCallback(() => {
    api<Detail>(`/api/v1/rep/merchants/${id}?page=${page}&status=${status}`)
      .then((d) => {
        setData(d);
        setMissing(false);
      })
      .catch(() => setMissing(true));
  }, [id, page, status]);

  useEffect(load, [load]);
  useLiveRefresh(["order", "wallet"], load);

  if (missing) {
    return (
      <PageContainer>
        <p className="rounded-card border border-line bg-surface p-10 text-center text-ink-muted">
          {D.notMine}
        </p>
        <Button variant="secondary" onClick={() => router.push("/portal/merchants")}>
          {D.back}
        </Button>
      </PageContainer>
    );
  }
  if (!data) return <LoadingState />;

  const mr = data.merchant;
  const logo = mediaUrl(mr.logo_thumb_url);
  const pages = Math.max(1, Math.ceil(data.total / data.per_page));

  const columns: DataColumn<RepOrder>[] = [
    {
      id: "number",
      header: D.colNumber,
      icon: <IconOrder />,
      primary: true,
      cell: (o) => <span dir="ltr">#{fmtNum(o.number)}</span>,
    },
    {
      id: "date",
      header: D.colDate,
      icon: <IconDate />,
      cell: (o) => <span dir="ltr">{fmtDate(o.delivered_at ?? o.created_at)}</span>,
    },
    {
      id: "status",
      header: D.colStatus,
      icon: <IconStatus />,
      cell: (o) => {
        const b = bucketOf(o.status);
        return <Badge variant={b.tone}>{b.label}</Badge>;
      },
    },
    {
      id: "reason",
      header: D.colReason,
      icon: <IconNote />,
      // يظهر للملغى وحده: عمودُ سببٍ فارغ في كل الصفوف ضجيج
      cell: (o) =>
        bucketOf(o.status).key === "cancelled" ? (
          <span className="text-danger">{o.cancel_reason || D.noReason}</span>
        ) : (
          <span className="text-ink-muted">—</span>
        ),
    },
    {
      id: "total",
      header: D.colTotal,
      icon: <IconOrder />,
      cell: (o) => <span dir="ltr">{fmtNum(o.total)}</span>,
    },
    {
      id: "commission",
      header: D.colCommission,
      icon: <IconBalance />,
      cell: (o) => <Forfeitable order={o} live={o.platform_commission} lost={o.forfeited_commission} />,
    },
    {
      id: "share",
      header: D.colMyShare,
      icon: <IconWallet />,
      cell: (o) => (
        <Forfeitable order={o} live={o.my_share} lost={o.forfeited_share} highlight />
      ),
    },
  ];

  return (
    <PageContainer>
      <PageHeader
        icon={IconStore}
        title={mr.name}
        subtitle={mr.category_name}
        actions={
          <Button variant="secondary" onClick={() => router.push("/portal/merchants")}>
            {D.back}
          </Button>
        }
      />

      <div className="flex items-center gap-3 rounded-card border border-line bg-surface p-4">
        {logo ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={logo} alt="" className="h-14 w-14 rounded-control object-cover" />
        ) : (
          <span className="flex h-14 w-14 items-center justify-center rounded-control bg-primary-light">
            <CategoryIcon name={mr.category_icon} size={24} />
          </span>
        )}
        <div className="min-w-0 flex-1">
          <p className="font-bold">{mr.name}</p>
          <p className="text-xs text-ink-muted">
            {m.rep.joinedAt} <span dir="ltr">{fmtDate(mr.joined_at)}</span>
          </p>
        </div>
        <Badge variant={mr.status === "active" ? "success" : "danger"}>
          {mr.status === "active" ? m.terms.active : m.terms.suspended}
        </Badge>
      </div>

      <StatGrid>
        <StatCard icon={IconOrder} label={D.sumOrders} value={data.summary.orders} />
        <StatCard
          icon={IconSuccess}
          label={D.sumDelivered}
          value={data.summary.delivered}
          tone="success"
        />
        <StatCard icon={IconError} label={D.sumCancelled} value={data.summary.cancelled} />
        <StatCard
          icon={IconWallet}
          label={`${D.sumEarnings} (${m.common.currency})`}
          value={data.summary.my_earnings}
          tone="accent"
        />
      </StatGrid>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="font-bold">{D.ordersTitle}</h2>
        <div className="flex items-center gap-2">
          <Select
            id="st"
            value={status}
            onChange={(e) => {
              setStatus(e.target.value);
              setPage(1); // فلترة جديدة تبدأ من أولها — لا من صفحةٍ قد لا توجد
            }}
            className="w-44"
          >
            <option value="">{D.filterAll}</option>
            <option value="delivered">{D.stDelivered}</option>
            <option value="cancelled">{D.stCancelled}</option>
          </Select>
          <ViewToggle
            view={view}
            onChange={setView}
            tableLabel={m.common.viewTable}
            cardsLabel={m.common.viewCards}
          />
        </div>
      </div>

      <p className="text-xs leading-relaxed text-ink-muted">{D.hint}</p>

      <DataView
        items={data.orders}
        getKey={(o) => String(o.number)}
        columns={columns}
        empty={D.empty}
        view={view}
      />

      {pages > 1 && (
        <div className="flex items-center justify-center gap-3">
          <Button
            variant="secondary"
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
            className="flex items-center gap-1"
          >
            <IconPrev size={16} />
            {m.common.back}
          </Button>
          <span className="text-sm text-ink-muted">
            {D.page.replace("{n}", fmtNum(page)).replace("{t}", fmtNum(pages))}
          </span>
          <Button
            variant="secondary"
            disabled={page >= pages}
            onClick={() => setPage((p) => p + 1)}
            className="flex items-center gap-1"
          >
            {m.common.next}
            <IconNext size={16} />
          </Button>
        </div>
      )}
    </PageContainer>
  );
}
