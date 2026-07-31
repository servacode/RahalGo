-- أيقونات التصنيفات تصبح مفاتيح لا إيموجي: الإيموجي يُرسم بخط النظام فيختلف بين
-- أبل وأندرويد وويندوز ولا يرث لون التوكنز — أما المفتاح فيقابله أيقونة من
-- مجموعتنا الواحدة. الترجمة تحفظ ما اختاره المستخدم سابقاً.
UPDATE categories SET icon = CASE icon
    WHEN '🍔' THEN 'food'
    WHEN '🍕' THEN 'food'
    WHEN '🍟' THEN 'food'
    WHEN '🛒' THEN 'grocery'
    WHEN '🏪' THEN 'grocery'
    WHEN '💊' THEN 'pharmacy'
    WHEN '🏥' THEN 'pharmacy'
    WHEN '🍰' THEN 'sweets'
    WHEN '🧁' THEN 'sweets'
    WHEN '🎁' THEN 'gifts'
    WHEN '🥤' THEN 'drinks'
    WHEN '☕' THEN 'drinks'
    WHEN '👕' THEN 'clothes'
    WHEN '🥐' THEN 'bakery'
    WHEN '🍞' THEN 'bakery'
    WHEN '🥩' THEN 'butcher'
    WHEN '🍎' THEN 'produce'
    WHEN '🥬' THEN 'produce'
    WHEN '💐' THEN 'flowers'
    WHEN '🌸' THEN 'flowers'
    WHEN '📱' THEN 'electronics'
    WHEN '📚' THEN 'books'
    WHEN '🍼' THEN 'baby'
    WHEN '💄' THEN 'beauty'
    ELSE 'other'
END
WHERE icon !~ '^[a-z]+$';
