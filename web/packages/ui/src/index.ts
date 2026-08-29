// **والألوانُ ليست هنا** — مصدرُها `theme.css` وحدَه، ومن احتاجها قيمةً
// يقرؤها بـ`themeColor`. (كانت لوحةً ثانيةً شاخت — انظر `tokens.ts`.)
export { fontFamily, radius, breakpoints } from "./tokens";
export { cssVar, themeColor } from "./cssvar";
// **التغذيةُ الراجعة** — رسالةٌ في موضعها، وهيكلٌ قبل المحتوى، وخبرٌ يمرّ،
// وتأكيدٌ لما لا يُستدرَك. (كانت مرتجَلةً في ٨٩ موضعاً بثمانِ صياغات.)
export {
  Alert,
  Skeleton,
  SkeletonText,
  SkeletonList,
  SkeletonStats,
  ToastStack,
  useToast,
  Confirm,
  type AlertTone,
  type ToastMsg,
} from "./feedback";
// **التنقّلُ داخل الصفحة** — تبويبٌ كان مرتجَلاً في ٦ ملفّات، وترقيمٌ في ٥٦
// موضعاً بلا مكوّن، وفتاتُ خبزٍ للوحةٍ بعمق ثلاثة مستويات.
export { Tabs, Chips, Pagination, Breadcrumb, type TabDef, type ChipDef } from "./navigation";
// **ما يعلو الصفحة** — ورقةٌ تصعد على الجوّال، وتلميحٌ لِما قُصّ، ومفتاحٌ
// لِما يقع فوراً.
export { Sheet, Tooltip, Switch } from "./overlay";
// **التنقّلُ السفليُّ على الجوّال** — الإبهامُ يصل الثلثَ السفليَّ وحدَه،
// وسائقُنا يمسك هاتفَه بيدٍ وهو واقفٌ في الشارع.
export { MobileNav, MobileNavSpacer, type NavItem } from "./MobileNav";
export {
  Button,
  Input,
  Textarea,
  Radio,
  Select,
  Badge,
  ButtonLink,
  CountBadge,
  IconTile,
  Modal,
  FormSection,
  OtpInput,
  PasswordMeter,
  passwordScore,
} from "./components";
// **ذيلُ النافذة** — موضعُ «حفظ» كان يتنقّل بين نافذةٍ وأخرى.
export { FormActions } from "./FormActions";
export { DataView, ViewToggle, useViewMode, type DataColumn, type ViewMode } from "./dataview";
/** **رفعُ الصور ومصغَّرُها** — كانا في لوحة الإدارة، **فبوّابةُ المتجر لا
    تراهما**، وصورةُ الصنف مبنيّةٌ في المحرّك بلا يدٍ ترفعها. */
export { ImageUpload, MediaThumb, type MediaKind } from "./ImageUpload";
export { FileUpload } from "./FileUpload";
export { AccountSettings } from "./AccountSettings";
/** **كرتُ الشكوى والبلاغ — واحدٌ في الخمس.** (قرارُ المالك ٢٠٢٦-٠٨-٠٧.) */
export { ComplaintCard, ComplaintGrid, type ComplaintTicket } from "./ComplaintCard";
export { NotificationsPage } from "./NotificationsPage";
export { WalletPage } from "./WalletPage";
export { AddressBook, type SavedAddress } from "./AddressBook";
export { Invoice, type InvoiceOrder, type InvoiceItem } from "./Invoice";
export {
  MenuManager,
  type MenuPaths,
  type MenuSection,
  type MenuItem,
} from "./MenuManager";
export {
  StatementSheet,
  currentMonthRange,
  type StatementData,
  type StatementTx,
} from "./Statement";
export {
  ReputationReviews,
  ReputationComplaints,
  type ReputationLabels,
} from "./Reputation";
export { DashboardChrome, type ChromeNavItem } from "./DashboardChrome";
export {
  PageContainer,
  PageHeader,
  Card,
  EmptyState,
  LoadingState,
  BootScreen,
  ReloadState,
  ListRow,
  StatGrid,
  StatCard,
  Stars,
  TabCards,
  EntityCard,
  type TabItem,
  type EntityStat,
} from "./layout";
export { Checkbox } from "./components";
export {
  CategoryIcon,
  CategoryIconPicker,
  CATEGORY_ICONS,
  CATEGORY_ICON_KEYS,
  categoryIconLabel,
  type CategoryIconKey,
} from "./CategoryIcon";
/** هويّةُ المنصة — **من الإعدادات لا من المعجم**، وعلامةٌ ترثها الخمسة. */
export { NetworkFx } from "./NetworkFx";
export { PlatformProvider, usePlatform, BrandMark, brandLetter, type Platform } from "./platform";
/** جالبُ الهويّة في الخادم — **ملفٌّ غيرُ عميلٍ عمداً.** */
export { fetchPlatform } from "./platform-server";
export { AuthTransition, type AuthTransitionKind } from "./AuthTransition";
export {
  TopBar,
  TopBarChip,
  TopBarLink,
  AppDownloadChip,
  WalletPill,
  TopBarActions,
  AccountMenu,
  type AccountMenuItem,
  TOPBAR_ICON,
  TOPBAR_AVATAR,
  Avatar,
  type ChipTone,
} from "./topbar";
export {
  LiveNotifications,
  NotificationBell,
  NotificationToast,
  useLiveNotifications,
  useLiveRefresh,
  useLiveEvent,
  useLiveStatus,
  useLiveData,
  emitLocal,
  type AppNotification,
  type LiveEvent,
} from "./Notifications";
export * from "./timeline";
export * from "./money";
export * from "./orderref";
export * from "./ordertrack";
export * from "./ordertrackv";
export * from "./MyAddresses";
// **حديثُ الطلب** — مكوّنٌ واحدٌ لطرفيه، بلا رقمٍ بينهما.
export * from "./OrderChat";
// **فقّاعةُ المحادثة** — تطفو ولا تسكن بطاقة.
export * from "./ChatBubble";
export * from "./ChatArchive";
// **توثيقُ واتساب** — صندوقٌ واحدٌ يُنادى حيث يُحتاج.
export * from "./WhatsAppVerify";
// **وقتُ السائق** — حسابٌ واحدٌ لبطاقتيه.
export * from "./drivereta";
export { StoreHours, type DayHours } from "./StoreHours";
export * from "./icons";
/** علاماتُ منصّات التواصل — ما نزعته `lucide`. */
export * from "./brand-icons";
/** عرضُ الصفحة الرئيسيّة. */

/** نغمةُ تنبيهٍ تُولَّد في المتصفّح — ومكرّرةٌ لمهمّةٍ وقعت بلا طلب. */
export { useChime, useRepeatingChime } from "./chime";

/** نبضةُ موضعِ السائق ومسافةٌ مقروءة — **والدورُ عدلٌ في الوقت أعمى في المكان.** */
export { useLocationBeacon, fmtDistance } from "./useLocationBeacon";

/** هدفي ومكافآتي — **وحافزٌ لا يُرى لا يحفّز.** */
export { MyIncentives } from "./Incentives";

/** سلايدرُ اللافتات — **ومن لا يسحب لا يرى إلّا الأولى.** */
export { BannerSlider, type SlideItem } from "./BannerSlider";
/** شريطُ الأقسام — **صورٌ دائريّةٌ تمشي وحدَها وتُساق باليد.** */
export { SectionRail, type RailItem } from "./SectionRail";
export { FavoriteButton, useFavorites } from "./Favorites";
export { CopyCode } from "./CopyCode";
