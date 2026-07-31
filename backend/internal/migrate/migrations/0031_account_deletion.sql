-- حذف الحساب نهائياً — حق لصاحبه، لكن **لا يُمحى الصفّ**.
--
-- الحساب مرتبط بطلبات ودفاتر قيود وتذاكر وسجل تدقيق: محوُه يمحو تاريخاً محاسبياً
-- لا يخصّه وحده (طلب اشتراه من متجر، ونقد بحوزة سائق، وعمولة قُيّدت لمندوب).
-- فالحذف = تجريد الحساب من هويته الشخصية وإقفاله:
--   * الحالة تصبح deleted فيُرفض دخوله من الوسيط فوراً
--   * الاسم والصورة يُمحيان، والهاتف يُستبدل برمز مجهول
--   * رقمه الأصلي يتحرّر فيستطيع التسجيل من جديد إن شاء
--   * الطلبات والقيود تبقى سليمة منسوبة إلى «حساب محذوف»
ALTER TABLE users DROP CONSTRAINT users_status_check;
ALTER TABLE users ADD CONSTRAINT users_status_check
    CHECK (status IN ('active', 'suspended', 'blocked', 'deleted'));

ALTER TABLE users ADD COLUMN deleted_at timestamptz;
