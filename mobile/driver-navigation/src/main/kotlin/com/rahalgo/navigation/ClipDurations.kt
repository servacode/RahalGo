package com.rahalgo.navigation

/**
 * ══════════════════════════════════════════════════════════════════════
 * **كم تستغرق كلُّ جملةٍ منطوقة — مُولَّدٌ من الحزمة لا مكتوبٌ بيد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **مُولَّد** بـ`tools/navigation-voice-pack` — ولا يُحرَّر بيد.
 *
 * # ولماذا يعرفها المخطّط
 *
 * (بلاغُ المالك ٢٠٢٦-٠٨-٢٤: «الصوتُ متأخّرٌ عن الطريق، خصوصاً إذا كان
 *  هناك أكثرُ من انعطافٍ قريباتٍ على بعض».)
 *
 * **وعتبةُ «الآن» كانت خمسَ ثوانٍ قبل المناورة** — وهي زمنُ **بدء**
 * الكلام لا زمنُ انتهائه. **ومقطعٌ من ٦٫٧ ثانيةٍ يبدأ قبل المنعطف بخمسٍ
 * فينتهي بعد أن يكون السائقُ جاوزه بثانيةٍ ونصف.**
 *
 * **والتعليمةُ تنفع إن انتهت قبل المناورة لا إن بدأت قبلها** — فالعتبةُ
 * تصير: زمنَ التنفيذ **زائدَ** طولِ الجملة.
 *
 * # والقياسُ من حجم الملفّ
 *
 * **والحزمةُ كلُّها بمعدّلٍ ثابت** (`mp3_44100_128`) — فالبايتاتُ زمنٌ
 * مضروبٌ في ثابت. **ووسمُ `ID3` يُطرح** وإلّا زاد كلَّ مقطعٍ جزءاً من
 * الثانية.
 */
object ClipDurations {

    /** **الافتراضُ لما لا نعرفه** — الوسيطُ لا الأقصر. */
    const val DEFAULT_MS = 4336

    /** **وأطولُ مقطعٍ في الحزمة** — يُقرأ في الاختبار. */
    const val LONGEST_MS = 6740

    private val table: Map<String, Int> by lazy {
        val out = HashMap<String, Int>(568)
        for (part in PACKED.split(',')) {
            val i = part.lastIndexOf(':')
            if (i > 0) out[part.substring(0, i)] = part.substring(i + 1).toInt()
        }
        out
    }

    /** **طولُ المقطع بالميلّي** — والمجهولُ يُعطى الوسيط. */
    fun ms(clip: String?): Int = clip?.let { table[it] } ?: DEFAULT_MS

    /** **بالثواني** — لحساب العتبات. */
    fun seconds(clip: String?): Double = ms(clip) / 1000.0

    /** **كم مقطعاً نعرف** — يُقرأ في الحارس. */
    val size: Int get() = table.size

    private const val PACKED =
        "approaching_destination:2064,arrived:2194,arrived_dropoff:2351,arrived_pickup:2482,continu" +
        "e_straight:1828,continue_straight_in_1000m:3631,continue_straight_in_100m:3317,continue_st" +
        "raight_in_1500m:3448,continue_straight_in_150m:4101,continue_straight_in_2000m:3213,contin" +
        "ue_straight_in_200m:3605,continue_straight_in_250m:4467,continue_straight_in_300m:3866,con" +
        "tinue_straight_in_400m:3317,continue_straight_in_500m:3735,continue_straight_in_50m:3605,c" +
        "ontinue_straight_in_700m:3553,continue_straight_now:2194,depart:1410,destination_left:1828" +
        ",destination_left_in_1000m:3605,destination_left_in_100m:3735,destination_left_in_1500m:40" +
        "23,destination_left_in_150m:4336,destination_left_in_2000m:3448,destination_left_in_200m:3" +
        "396,destination_left_in_250m:4441,destination_left_in_300m:4101,destination_left_in_400m:3" +
        "866,destination_left_in_500m:4023,destination_left_in_50m:3605,destination_left_in_700m:39" +
        "18,destination_right:1776,destination_right_in_1000m:3735,destination_right_in_100m:3605,d" +
        "estination_right_in_1500m:3788,destination_right_in_150m:4232,destination_right_in_2000m:3" +
        "500,destination_right_in_200m:3317,destination_right_in_250m:4441,destination_right_in_300" +
        "m:4206,destination_right_in_400m:3918,destination_right_in_500m:3814,destination_right_in_" +
        "50m:3605,destination_right_in_700m:3866,end_of_road_left:3187,end_of_road_left_in_1000m:58" +
        "77,end_of_road_left_in_100m:4885,end_of_road_left_in_1500m:5695,end_of_road_left_in_150m:6" +
        "008,end_of_road_left_in_2000m:5068,end_of_road_left_in_200m:5277,end_of_road_left_in_250m:" +
        "6139,end_of_road_left_in_300m:5877,end_of_road_left_in_400m:5695,end_of_road_left_in_500m:" +
        "5642,end_of_road_left_in_50m:5172,end_of_road_left_in_700m:5303,end_of_road_left_now:3788," +
        "end_of_road_right:3265,end_of_road_right_in_1000m:5695,end_of_road_right_in_100m:4885,end_" +
        "of_road_right_in_1500m:5877,end_of_road_right_in_150m:6322,end_of_road_right_in_2000m:5042" +
        ",end_of_road_right_in_200m:5042,end_of_road_right_in_250m:6008,end_of_road_right_in_300m:5" +
        "825,end_of_road_right_in_400m:5407,end_of_road_right_in_500m:5590,end_of_road_right_in_50m" +
        ":5460,end_of_road_right_in_700m:5956,end_of_road_right_now:3814,exit_roundabout:1698,exit_" +
        "roundabout_in_1000m:3605,exit_roundabout_in_100m:3265,exit_roundabout_in_1500m:3683,exit_r" +
        "oundabout_in_150m:4232,exit_roundabout_in_2000m:3187,exit_roundabout_in_200m:3553,exit_rou" +
        "ndabout_in_250m:4519,exit_roundabout_in_300m:3683,exit_roundabout_in_400m:3448,exit_rounda" +
        "bout_in_500m:3683,exit_roundabout_in_50m:3500,exit_roundabout_in_700m:3448,exit_roundabout" +
        "_now:2064,follow_route:1358,fork:2534,fork_in_1000m:4519,fork_in_100m:3788,fork_in_1500m:4" +
        "571,fork_in_150m:5068,fork_in_2000m:4101,fork_in_200m:4388,fork_in_250m:5303,fork_in_300m:" +
        "4624,fork_in_400m:4284,fork_in_500m:4571,fork_in_50m:4388,fork_in_700m:4519,fork_left:3135" +
        ",fork_left_in_1000m:5407,fork_left_in_100m:4859,fork_left_in_1500m:6008,fork_left_in_150m:" +
        "6374,fork_left_in_2000m:4937,fork_left_in_200m:5224,fork_left_in_250m:6139,fork_left_in_30" +
        "0m:5590,fork_left_in_400m:5590,fork_left_in_500m:5590,fork_left_in_50m:5277,fork_left_in_7" +
        "00m:5904,fork_left_now:3970,fork_now:2900,fork_right:3213,fork_right_in_1000m:5460,fork_ri" +
        "ght_in_100m:4859,fork_right_in_1500m:6139,fork_right_in_150m:6243,fork_right_in_2000m:5224" +
        ",fork_right_in_200m:5224,fork_right_in_250m:6740,fork_right_in_300m:5695,fork_right_in_400" +
        "m:5355,fork_right_in_500m:5642,fork_right_in_50m:5120,fork_right_in_700m:5303,fork_right_n" +
        "ow:3683,gps_lost:2769,gps_restored:2246,gps_weak:2246,keep_left:2116,keep_left_in_1000m:42" +
        "32,keep_left_in_100m:3788,keep_left_in_1500m:4206,keep_left_in_150m:4624,keep_left_in_2000" +
        "m:3814,keep_left_in_200m:3683,keep_left_in_250m:4571,keep_left_in_300m:4441,keep_left_in_4" +
        "00m:4232,keep_left_in_500m:4049,keep_left_in_50m:3918,keep_left_in_700m:4153,keep_left_now" +
        ":2482,keep_right:2142,keep_right_in_1000m:4049,keep_right_in_100m:3553,keep_right_in_1500m" +
        ":3970,keep_right_in_150m:4702,keep_right_in_2000m:3631,keep_right_in_200m:3735,keep_right_" +
        "in_250m:4702,keep_right_in_300m:4232,keep_right_in_400m:4023,keep_right_in_500m:4206,keep_" +
        "right_in_50m:3970,keep_right_in_700m:4023,keep_right_now:2482,merge:1776,merge_in_1000m:38" +
        "14,merge_in_100m:3396,merge_in_1500m:3735,merge_in_150m:4284,merge_in_2000m:3082,merge_in_" +
        "200m:3265,merge_in_250m:4806,merge_in_300m:3605,merge_in_400m:3605,merge_in_500m:3500,merg" +
        "e_in_50m:3317,merge_in_700m:3631,merge_left:2664,merge_left_in_1000m:4624,merge_left_in_10" +
        "0m:4153,merge_left_in_1500m:4937,merge_left_in_150m:5120,merge_left_in_2000m:4101,merge_le" +
        "ft_in_200m:4153,merge_left_in_250m:5460,merge_left_in_300m:4937,merge_left_in_400m:4859,me" +
        "rge_left_in_500m:4624,merge_left_in_50m:4571,merge_left_in_700m:4624,merge_left_now:2795,m" +
        "erge_now:2011,merge_right:2612,merge_right_in_1000m:4702,merge_right_in_100m:4206,merge_ri" +
        "ght_in_1500m:4571,merge_right_in_150m:5355,merge_right_in_2000m:4153,merge_right_in_200m:3" +
        "814,merge_right_in_250m:5538,merge_right_in_300m:4937,merge_right_in_400m:4388,merge_right" +
        "_in_500m:4754,merge_right_in_50m:4441,merge_right_in_700m:4885,merge_right_now:2847,naviga" +
        "tion_started:1515,ramp:1410,ramp_in_1000m:3213,ramp_in_100m:3135,ramp_in_1500m:3213,ramp_i" +
        "n_150m:3631,ramp_in_2000m:2717,ramp_in_200m:3187,ramp_in_250m:4101,ramp_in_300m:3213,ramp_" +
        "in_400m:3213,ramp_in_500m:2978,ramp_in_50m:3265,ramp_in_700m:2978,ramp_left:2299,ramp_left" +
        "_in_1000m:4206,ramp_left_in_100m:3605,ramp_left_in_1500m:4232,ramp_left_in_150m:4624,ramp_" +
        "left_in_2000m:3500,ramp_left_in_200m:3631,ramp_left_in_250m:4467,ramp_left_in_300m:4232,ra" +
        "mp_left_in_400m:4206,ramp_left_in_500m:4153,ramp_left_in_50m:4153,ramp_left_in_700m:4101,r" +
        "amp_left_now:2534,ramp_now:1698,ramp_right:2064,ramp_right_in_1000m:4049,ramp_right_in_100" +
        "m:3631,ramp_right_in_1500m:4336,ramp_right_in_150m:4388,ramp_right_in_2000m:3683,ramp_righ" +
        "t_in_200m:3396,ramp_right_in_250m:4624,ramp_right_in_300m:4153,ramp_right_in_400m:4101,ram" +
        "p_right_in_500m:3866,ramp_right_in_50m:3970,ramp_right_in_700m:3866,ramp_right_now:2534,re" +
        "calculating_route:2534,reroute_failed:2952,roundabout_continue:2769,roundabout_continue_in" +
        "_1000m:4467,roundabout_continue_in_100m:4441,roundabout_continue_in_1500m:5120,roundabout_" +
        "continue_in_150m:5642,roundabout_continue_in_2000m:4153,roundabout_continue_in_200m:4441,r" +
        "oundabout_continue_in_250m:5407,roundabout_continue_in_300m:4885,roundabout_continue_in_40" +
        "0m:4650,roundabout_continue_in_500m:4519,roundabout_continue_in_50m:4336,roundabout_contin" +
        "ue_in_700m:4702,roundabout_continue_now:2900,roundabout_exit_1:2900,roundabout_exit_10:297" +
        "8,roundabout_exit_10_in_1000m:4989,roundabout_exit_10_in_100m:4650,roundabout_exit_10_in_1" +
        "500m:5355,roundabout_exit_10_in_150m:5590,roundabout_exit_10_in_2000m:4519,roundabout_exit" +
        "_10_in_200m:4702,roundabout_exit_10_in_250m:6139,roundabout_exit_10_in_300m:5538,roundabou" +
        "t_exit_10_in_400m:4859,roundabout_exit_10_in_500m:5068,roundabout_exit_10_in_50m:5042,roun" +
        "dabout_exit_10_in_700m:5172,roundabout_exit_10_now:3370,roundabout_exit_11:3317,roundabout" +
        "_exit_11_in_1000m:5407,roundabout_exit_11_in_100m:5042,roundabout_exit_11_in_1500m:6113,ro" +
        "undabout_exit_11_in_150m:6661,roundabout_exit_11_in_2000m:4754,roundabout_exit_11_in_200m:" +
        "5042,roundabout_exit_11_in_250m:6243,roundabout_exit_11_in_300m:5277,roundabout_exit_11_in" +
        "_400m:5460,roundabout_exit_11_in_500m:5355,roundabout_exit_11_in_50m:5277,roundabout_exit_" +
        "11_in_700m:5460,roundabout_exit_11_now:3683,roundabout_exit_12:3317,roundabout_exit_12_in_" +
        "1000m:5877,roundabout_exit_12_in_100m:4989,roundabout_exit_12_in_1500m:5120,roundabout_exi" +
        "t_12_in_150m:5721,roundabout_exit_12_in_2000m:4989,roundabout_exit_12_in_200m:4937,roundab" +
        "out_exit_12_in_250m:5956,roundabout_exit_12_in_300m:5956,roundabout_exit_12_in_400m:5538,r" +
        "oundabout_exit_12_in_500m:5590,roundabout_exit_12_in_50m:5068,roundabout_exit_12_in_700m:5" +
        "642,roundabout_exit_12_now:3788,roundabout_exit_1_in_1000m:4937,roundabout_exit_1_in_100m:" +
        "4284,roundabout_exit_1_in_1500m:5068,roundabout_exit_1_in_150m:5642,roundabout_exit_1_in_2" +
        "000m:4519,roundabout_exit_1_in_200m:4441,roundabout_exit_1_in_250m:5956,roundabout_exit_1_" +
        "in_300m:5042,roundabout_exit_1_in_400m:4806,roundabout_exit_1_in_500m:4702,roundabout_exit" +
        "_1_in_50m:4388,roundabout_exit_1_in_700m:4937,roundabout_exit_1_now:3082,roundabout_exit_2" +
        ":2847,roundabout_exit_2_in_1000m:5120,roundabout_exit_2_in_100m:4232,roundabout_exit_2_in_" +
        "1500m:5042,roundabout_exit_2_in_150m:5120,roundabout_exit_2_in_2000m:4519,roundabout_exit_" +
        "2_in_200m:4388,roundabout_exit_2_in_250m:5773,roundabout_exit_2_in_300m:5172,roundabout_ex" +
        "it_2_in_400m:4859,roundabout_exit_2_in_500m:4806,roundabout_exit_2_in_50m:4467,roundabout_" +
        "exit_2_in_700m:4937,roundabout_exit_2_now:3082,roundabout_exit_3:2952,roundabout_exit_3_in" +
        "_1000m:4754,roundabout_exit_3_in_100m:4624,roundabout_exit_3_in_1500m:4989,roundabout_exit" +
        "_3_in_150m:5460,roundabout_exit_3_in_2000m:4806,roundabout_exit_3_in_200m:4467,roundabout_" +
        "exit_3_in_250m:6243,roundabout_exit_3_in_300m:5277,roundabout_exit_3_in_400m:4989,roundabo" +
        "ut_exit_3_in_500m:4650,roundabout_exit_3_in_50m:4571,roundabout_exit_3_in_700m:4806,rounda" +
        "bout_exit_3_now:3370,roundabout_exit_4:2847,roundabout_exit_4_in_1000m:4859,roundabout_exi" +
        "t_4_in_100m:4284,roundabout_exit_4_in_1500m:5538,roundabout_exit_4_in_150m:5695,roundabout" +
        "_exit_4_in_2000m:4519,roundabout_exit_4_in_200m:4571,roundabout_exit_4_in_250m:6374,rounda" +
        "bout_exit_4_in_300m:4702,roundabout_exit_4_in_400m:4859,roundabout_exit_4_in_500m:4937,rou" +
        "ndabout_exit_4_in_50m:5120,roundabout_exit_4_in_700m:4885,roundabout_exit_4_now:3370,round" +
        "about_exit_5:2847,roundabout_exit_5_in_1000m:4885,roundabout_exit_5_in_100m:4284,roundabou" +
        "t_exit_5_in_1500m:5277,roundabout_exit_5_in_150m:5277,roundabout_exit_5_in_2000m:4467,roun" +
        "dabout_exit_5_in_200m:4650,roundabout_exit_5_in_250m:5877,roundabout_exit_5_in_300m:4624,r" +
        "oundabout_exit_5_in_400m:5277,roundabout_exit_5_in_500m:5224,roundabout_exit_5_in_50m:4624" +
        ",roundabout_exit_5_in_700m:4650,roundabout_exit_5_now:3213,roundabout_exit_6:2847,roundabo" +
        "ut_exit_6_in_1000m:5224,roundabout_exit_6_in_100m:4284,roundabout_exit_6_in_1500m:4702,rou" +
        "ndabout_exit_6_in_150m:5277,roundabout_exit_6_in_2000m:4388,roundabout_exit_6_in_200m:4336" +
        ",roundabout_exit_6_in_250m:5773,roundabout_exit_6_in_300m:5172,roundabout_exit_6_in_400m:4" +
        "885,roundabout_exit_6_in_500m:5407,roundabout_exit_6_in_50m:4624,roundabout_exit_6_in_700m" +
        ":5120,roundabout_exit_6_now:3448,roundabout_exit_7:2900,roundabout_exit_7_in_1000m:5277,ro" +
        "undabout_exit_7_in_100m:4702,roundabout_exit_7_in_1500m:5224,roundabout_exit_7_in_150m:569" +
        "5,roundabout_exit_7_in_2000m:4206,roundabout_exit_7_in_200m:4519,roundabout_exit_7_in_250m" +
        ":6060,roundabout_exit_7_in_300m:5224,roundabout_exit_7_in_400m:4754,roundabout_exit_7_in_5" +
        "00m:5277,roundabout_exit_7_in_50m:4388,roundabout_exit_7_in_700m:5042,roundabout_exit_7_no" +
        "w:3317,roundabout_exit_8:2795,roundabout_exit_8_in_1000m:5172,roundabout_exit_8_in_100m:42" +
        "84,roundabout_exit_8_in_1500m:4989,roundabout_exit_8_in_150m:5460,roundabout_exit_8_in_200" +
        "0m:4650,roundabout_exit_8_in_200m:4624,roundabout_exit_8_in_250m:6609,roundabout_exit_8_in" +
        "_300m:5538,roundabout_exit_8_in_400m:4885,roundabout_exit_8_in_500m:4806,roundabout_exit_8" +
        "_in_50m:4754,roundabout_exit_8_in_700m:4806,roundabout_exit_8_now:3317,roundabout_exit_9:2" +
        "952,roundabout_exit_9_in_1000m:5120,roundabout_exit_9_in_100m:4388,roundabout_exit_9_in_15" +
        "00m:4989,roundabout_exit_9_in_150m:6295,roundabout_exit_9_in_2000m:4806,roundabout_exit_9_" +
        "in_200m:4650,roundabout_exit_9_in_250m:6113,roundabout_exit_9_in_300m:5956,roundabout_exit" +
        "_9_in_400m:5486,roundabout_exit_9_in_500m:4806,roundabout_exit_9_in_50m:4806,roundabout_ex" +
        "it_9_in_700m:5068,roundabout_exit_9_now:3370,route_end:3814,route_updated:1724,sharp_left:" +
        "2560,sharp_left_in_1000m:4519,sharp_left_in_100m:4023,sharp_left_in_1500m:4441,sharp_left_" +
        "in_150m:4937,sharp_left_in_2000m:4441,sharp_left_in_200m:4101,sharp_left_in_250m:5460,shar" +
        "p_left_in_300m:4650,sharp_left_in_400m:4388,sharp_left_in_500m:4336,sharp_left_in_50m:4101" +
        ",sharp_left_in_700m:4441,sharp_left_now:2769,sharp_right:2429,sharp_right_in_1000m:4467,sh" +
        "arp_right_in_100m:4153,sharp_right_in_1500m:4519,sharp_right_in_150m:5172,sharp_right_in_2" +
        "000m:3970,sharp_right_in_200m:4101,sharp_right_in_250m:5355,sharp_right_in_300m:4702,sharp" +
        "_right_in_400m:4650,sharp_right_in_500m:4336,sharp_right_in_50m:4153,sharp_right_in_700m:4" +
        "441,sharp_right_now:2717,slight_left:2482,slight_left_in_1000m:4859,slight_left_in_100m:42" +
        "06,slight_left_in_1500m:4806,slight_left_in_150m:5120,slight_left_in_2000m:4023,slight_lef" +
        "t_in_200m:3970,slight_left_in_250m:5538,slight_left_in_300m:4624,slight_left_in_400m:4336," +
        "slight_left_in_500m:4754,slight_left_in_50m:4806,slight_left_in_700m:4284,slight_left_now:" +
        "2795,slight_right:2429,slight_right_in_1000m:4101,slight_right_in_100m:4023,slight_right_i" +
        "n_1500m:4232,slight_right_in_150m:4806,slight_right_in_2000m:4284,slight_right_in_200m:363" +
        "1,slight_right_in_250m:5277,slight_right_in_300m:4153,slight_right_in_400m:4624,slight_rig" +
        "ht_in_500m:4702,slight_right_in_50m:4153,slight_right_in_700m:4467,slight_right_now:2534,t" +
        "hen_continue_straight:2717,then_destination_left:2978,then_destination_right:3030,then_end" +
        "_of_road_left:4284,then_end_of_road_right:4388,then_exit_roundabout:2795,then_fork:3631,th" +
        "en_fork_left:4702,then_fork_right:4571,then_keep_left:3135,then_keep_right:3030,then_merge" +
        ":2847,then_merge_left:3735,then_merge_right:4101,then_ramp:2482,then_ramp_left:2978,then_r" +
        "amp_right:3135,then_roundabout_continue:4153,then_roundabout_exit_1:4232,then_roundabout_e" +
        "xit_10:4388,then_roundabout_exit_11:4519,then_roundabout_exit_12:4650,then_roundabout_exit" +
        "_2:4101,then_roundabout_exit_3:4232,then_roundabout_exit_4:4206,then_roundabout_exit_5:423" +
        "2,then_roundabout_exit_6:4284,then_roundabout_exit_7:4336,then_roundabout_exit_8:4101,then" +
        "_roundabout_exit_9:4467,then_sharp_left:3553,then_sharp_right:3814,then_slight_left:3396,t" +
        "hen_slight_right:3396,then_turn_left:2534,then_turn_right:2664,then_uturn:4023,turn_left:1" +
        "515,turn_left_in_1000m:3187,turn_left_in_100m:3030,turn_left_in_1500m:3187,turn_left_in_15" +
        "0m:4049,turn_left_in_2000m:2900,turn_left_in_200m:3265,turn_left_in_250m:4023,turn_left_in" +
        "_300m:3448,turn_left_in_400m:3187,turn_left_in_500m:3030,turn_left_in_50m:3082,turn_left_i" +
        "n_700m:3317,turn_left_now:1828,turn_right:1541,turn_right_in_1000m:3082,turn_right_in_100m" +
        ":2795,turn_right_in_1500m:3265,turn_right_in_150m:4153,turn_right_in_2000m:2847,turn_right" +
        "_in_200m:3082,turn_right_in_250m:4284,turn_right_in_300m:3265,turn_right_in_400m:3213,turn" +
        "_right_in_500m:2978,turn_right_in_50m:3082,turn_right_in_700m:3265,turn_right_now:1881,utu" +
        "rn:3082,uturn_in_1000m:5460,uturn_in_100m:4519,uturn_in_1500m:5355,uturn_in_150m:6322,utur" +
        "n_in_2000m:4885,uturn_in_200m:4624,uturn_in_250m:5721,uturn_in_300m:5642,uturn_in_400m:530" +
        "3,uturn_in_500m:5407,uturn_in_50m:5303,uturn_in_700m:5355,uturn_now:3553,wrong_way:2612"
}
