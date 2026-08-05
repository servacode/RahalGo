// **والألوانُ ليست هنا** — مصدرُها `theme.css` وحدَه، ومن احتاجها قيمةً
// يقرؤها بـ`themeColor`. (كانت لوحةً ثانيةً شاخت — انظر `tokens.ts`.)
export { fontFamily, radius, breakpoints } from "./tokens";
export { cssVar, themeColor } from "./cssvar";
export {
  Button,
  Input,
  Textarea,
  Radio,
  Select,
  Badge,
  Modal,
  FormSection,
  OtpInput,
  PasswordMeter,
  passwordScore,
} from "./components";
export { DataView, ViewToggle, useViewMode, type DataColumn, type ViewMode } from "./dataview";
export { AccountSettings } from "./AccountSettings";
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
export {
  TopBar,
  TopBarChip,
  TopBarLink,
  WalletPill,
  TopBarActions,
  TOPBAR_ICON,
  TOPBAR_AVATAR,
  Avatar,
  CountBadge,
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
export * from "./ordertrack";
export * from "./MyAddresses";
export { StoreHours, type DayHours } from "./StoreHours";
export * from "./icons";

/** نغمةُ تنبيهٍ تُولَّد في المتصفّح — ومكرّرةٌ لمهمّةٍ وقعت بلا طلب. */
export { useChime, useRepeatingChime } from "./chime";

/** نبضةُ موضعِ السائق ومسافةٌ مقروءة — **والدورُ عدلٌ في الوقت أعمى في المكان.** */
export { useLocationBeacon, fmtDistance } from "./useLocationBeacon";

/** هدفي ومكافآتي — **وحافزٌ لا يُرى لا يحفّز.** */
export { MyIncentives } from "./Incentives";

/** سلايدرُ اللافتات — **ومن لا يسحب لا يرى إلّا الأولى.** */
export { BannerSlider, type SlideItem } from "./BannerSlider";
export {
  FavoritesPage,
  FavoriteButton,
  useFavorites,
  type FavoriteItem,
} from "./Favorites";
