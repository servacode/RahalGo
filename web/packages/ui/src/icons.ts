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
  // **تبديلُ المتجر — سهمان متبادلان.**
  //
  // **والصورةُ تسبق اللفظ**: من يرى سهمين متبادلين يعرف أنّ شيئاً يحلّ محلَّ
  // شيء **قبل أن يقرأ الكلمة** — وسهمٌ واحدٌ يُقرأ «إرسالاً» لا «تبديلاً».
  Repeat as IconSwap,
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
  // **الدرّاجةُ النارية** — مركبةُ التوصيل في الرقّة، لا شاحنة. تُستعمل في
  // مسار الطلب عند الزبون (`OrderTrack`).
  Motorbike as IconMoto,
  // اتّجاهُ حركةِ المال — **داخلٌ أم خارج**: سهمٌ يقولها لمن لا يميّز اللون.
  ArrowDown as IconArrowIn,
  ArrowUp as IconArrowOut,
  // **الطلباتُ سجلّاتٌ لا تسوّق.**
  //
  // كانت `ShoppingBasket` — سلّةً بجانب عربة السلّة في شريط الزبون، فلا يفرّق
  // بينهما ناظر. والرقمُ على إحداهما يُقرأ «طلبات» وهو عدد أصناف السلّة.
  // (اكتشفه صاحب المنصة في أوّل تجربةٍ بشرية: «٤ أنواع لا تُعدّ ٤ طلبات».)
  //
  // والفاتورة أصدق في كل موضع: في لوحة الإدارة والمتجر والسائق «الطلبات»
  // سجلّاتٌ تُدار وتُقرأ، لا بضاعةٌ تُشترى. **والعربةُ وحدها للسلّة.**
  ReceiptText as IconOrder,
  // **عربةٌ لا حقيبة.**
  //
  // جرّبتُ `ShoppingBag` وحُكم عليها بالنظر: غير موفَّقة — تُقرأ صندوقاً لا
  // عربة. والعربةُ عندها عجلاتٌ تدور، وعليها تقوم حركةُ السلّة العائمة كلُّها
  // (تتقدّم ثم تلتفّ ثم تعود). **وحقيبةٌ لا تتدحرج.**
  ShoppingCart as IconCart,
  LifeBuoy as IconSupport,
  Eye as IconView,
  EyeOff as IconViewOff,
  LocateFixed as IconLocateMe,
  Scale as IconBalance,
  // **قائمةٌ وشبكة** — كانتا مرسومتين بالحرف داخل `dataview.tsx`.
  // **وأيقونةٌ تُرسم في مكوّنٍ تُرسم ثانيةً في غيره بخطٍّ مختلف** — فتفترق
  // سماكتُها ومقاسُها ولا يلاحظ أحد.
  List as IconList,
  LayoutGrid as IconGrid,
  Bell as IconBell,
  // **هلالٌ وشمس** — مبدّلُ سمة لوحة السائق، **والأيقونةُ تُظهر الوجهةَ
  // لا الحال**: من رأى الهلالَ عرف أنّ الضغطةَ تُغمّق. (وهو نفسُ ما في
  // التطبيق — `ic_theme.xml` و`ic_theme_light.xml`.)
  Moon as IconMoon,
  Sun as IconSun,
  ChevronDown as IconChevronDown,
  Link2 as IconLink,
  QrCode as IconQr,
  LayoutDashboard as IconOverview,
  Star as IconStar,
  // **المفضّلة قلبٌ لا نجمة.**
  //
  // **والنجمةُ هنا محجوزةٌ للتقييم** — تُستعمل في `Stars` وفي التقييمات كلِّها،
  // **ونجمةٌ تعني «قيّمتُه» ونجمةٌ تعني «أحببتُه» في شاشةٍ واحدة** تجعل من
  // يضغط إحداهما يتوقّع الأخرى.
  Heart as IconHeart,
  MessageSquare as IconReply,
  StickyNote as IconNote,
  // **حديثُ الطلب** — فقّاعةٌ مستديرةٌ لا مربّعة، **ليُفرَّق عن الردّ الإداريّ**
  // (`IconReply`): ذاك سطرٌ في تذكرة، وهذا حديثٌ بين طرفين.
  MessageCircle as IconChat,
  Send as IconSend,
  Copy as IconCopy,
  BellOff as IconBellOff,
  TrendingUp as IconTrendUp,
  TrendingDown as IconTrendDown,
  Minus as IconTrendFlat,
  KeyRound as IconKey,
  UserPlus as IconSignup,
  BadgeCheck as IconVerified,
  Printer as IconPrint,
  // **وتحميلُ التطبيق سهمٌ إلى هاتف** — لا سهمَ تنزيلٍ عامّاً: الزرُّ
  // بجانب زرِّ الدخول، **والسهمُ وحدَه يُقرأ «نزِّل ملفّاً» لا «خذ التطبيق».**
  Smartphone as IconApp,
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
  // **صحّان — «قُرئت».** (قرارُ المالك ٢٠٢٦-٠٨-١٠.)
  CheckCheck as IconCheckAll,
  // الحالات
  CircleCheck as IconSuccess,
  CircleAlert as IconWarning,
  CircleX as IconError,
  Loader as IconLoading,
  // **أيقونةُ الكاميرا** — لإثبات التسليم. **ولا تُضاف أيقونةٌ إلّا لمعنًى
  // جديد**: مجموعةٌ تتضخّم بالمترادفات تجعل كلَّ شاشةٍ تختار غيرَ ما اختارت
  // أختُها.
  Camera as IconCamera,
} from "lucide-react";
