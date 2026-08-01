/**
 * مجموعة الأيقونات المعتمدة (Lucide — BRAND.md) بأسماء دلالية موحدة.
 * كل التطبيقات تستورد الأيقونات من هنا حصراً — لضمان مجموعة واحدة
 * بسماكة واحدة في كل المنظومة، وتسهيل أي استبدال مستقبلي.
 */
export {
  // التنقل والأقسام
  LayoutDashboard as IconDashboard,
  Users as IconUsers,
  Store as IconStore,
  Map as IconZones,
  TicketPercent as IconPromos,
  MessageCircle as IconWhatsApp,
  Settings as IconSettings,
  // الحقول والبيانات
  User as IconUser,
  Phone as IconPhone,
  Lock as IconLock,
  Shield as IconRoles,
  Activity as IconStatus,
  Calendar as IconDate,
  Search as IconSearch,
  MapPin as IconLocation,
  Wallet as IconWallet,
  Truck as IconDriver,
  // **الطلباتُ سجلّاتٌ لا تسوّق.**
  //
  // كانت `ShoppingBasket` — سلّةً بجانب عربة السلّة في شريط الزبون، فلا يفرّق
  // بينهما ناظر. والرقمُ على إحداهما يُقرأ «طلبات» وهو عدد أصناف السلّة.
  // (اكتشفه صاحب المنصة في أوّل تجربةٍ بشرية: «٤ أنواع لا تُعدّ ٤ طلبات».)
  //
  // والفاتورة أصدق في كل موضع: في لوحة الإدارة والمتجر والسائق «الطلبات»
  // سجلّاتٌ تُدار وتُقرأ، لا بضاعةٌ تُشترى. **والعربةُ وحدها للسلّة.**
  ReceiptText as IconOrder,
  // **حقيبةٌ لا عربة.**
  //
  // `ShoppingCart` عجلاتُها وقبضتُها خطوطٌ رفيعة: تُقرأ في ١٧ بكسل وتتفكّك في
  // ٥٦ — والسلّة العائمة كبيرة. والحقيبةُ شكلٌ مصمَتٌ بسيط يثبت في كل مقاس،
  // وهي ما تستعمله منصات التوصيل الكبيرة لهذا السبب نفسه.
  ShoppingBag as IconCart,
  LifeBuoy as IconSupport,
  Eye as IconView,
  EyeOff as IconViewOff,
  LocateFixed as IconLocateMe,
  Scale as IconBalance,
  Bell as IconBell,
  ChevronDown as IconChevronDown,
  Link2 as IconLink,
  QrCode as IconQr,
  LayoutDashboard as IconOverview,
  Star as IconStar,
  MessageSquare as IconReply,
  StickyNote as IconNote,
  Copy as IconCopy,
  BellOff as IconBellOff,
  TrendingUp as IconTrendUp,
  TrendingDown as IconTrendDown,
  Minus as IconTrendFlat,
  KeyRound as IconKey,
  UserPlus as IconSignup,
  BadgeCheck as IconVerified,
  Printer as IconPrint,
  // تصنيفات المتاجر — أيقونة لكل نشاط بدل الإيموجي
  UtensilsCrossed as IconCatFood,
  ShoppingBag as IconCatGrocery,
  Pill as IconCatPharmacy,
  CakeSlice as IconCatSweets,
  Gift as IconCatGifts,
  CupSoda as IconCatDrinks,
  Shirt as IconCatClothes,
  Croissant as IconCatBakery,
  Beef as IconCatButcher,
  Apple as IconCatProduce,
  Flower2 as IconCatFlowers,
  Smartphone as IconCatElectronics,
  BookOpen as IconCatBooks,
  Baby as IconCatBaby,
  Sparkles as IconCatBeauty,
  Package as IconCatOther,
  // الإجراءات
  Menu as IconHamburger,
  Plus as IconAdd,
  Pencil as IconEdit,
  Trash2 as IconDelete,
  Ban as IconBlock,
  RotateCcw as IconUnblock,
  LogOut as IconLogout,
  ChevronRight as IconPrev,
  ChevronLeft as IconNext,
  X as IconClose,
  Check as IconCheck,
  // الحالات
  CircleCheck as IconSuccess,
  CircleAlert as IconWarning,
  CircleX as IconError,
  Loader as IconLoading,
} from "lucide-react";
