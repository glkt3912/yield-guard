import type { BuildingType } from "@/types/investment";

export const PREFECTURES = [
  { value: "01", label: "北海道" },
  { value: "02", label: "青森県" },
  { value: "03", label: "岩手県" },
  { value: "04", label: "宮城県" },
  { value: "05", label: "秋田県" },
  { value: "06", label: "山形県" },
  { value: "07", label: "福島県" },
  { value: "08", label: "茨城県" },
  { value: "09", label: "栃木県" },
  { value: "10", label: "群馬県" },
  { value: "11", label: "埼玉県" },
  { value: "12", label: "千葉県" },
  { value: "13", label: "東京都" },
  { value: "14", label: "神奈川県" },
  { value: "15", label: "新潟県" },
  { value: "16", label: "富山県" },
  { value: "17", label: "石川県" },
  { value: "18", label: "福井県" },
  { value: "19", label: "山梨県" },
  { value: "20", label: "長野県" },
  { value: "21", label: "岐阜県" },
  { value: "22", label: "静岡県" },
  { value: "23", label: "愛知県" },
  { value: "24", label: "三重県" },
  { value: "25", label: "滋賀県" },
  { value: "26", label: "京都府" },
  { value: "27", label: "大阪府" },
  { value: "28", label: "兵庫県" },
  { value: "29", label: "奈良県" },
  { value: "30", label: "和歌山県" },
  { value: "31", label: "鳥取県" },
  { value: "32", label: "島根県" },
  { value: "33", label: "岡山県" },
  { value: "34", label: "広島県" },
  { value: "35", label: "山口県" },
  { value: "36", label: "徳島県" },
  { value: "37", label: "香川県" },
  { value: "38", label: "愛媛県" },
  { value: "39", label: "高知県" },
  { value: "40", label: "福岡県" },
  { value: "41", label: "佐賀県" },
  { value: "42", label: "長崎県" },
  { value: "43", label: "熊本県" },
  { value: "44", label: "大分県" },
  { value: "45", label: "宮崎県" },
  { value: "46", label: "鹿児島県" },
  { value: "47", label: "沖縄県" },
];

export const BUILDING_TYPES: { value: BuildingType; label: string }[] = [
  { value: "木造", label: "木造（耐用22年）" },
  { value: "軽量鉄骨(3mm以下)", label: "軽量鉄骨・薄板（耐用19年）" },
  { value: "軽量鉄骨(4mm以下)", label: "軽量鉄骨（耐用27年）" },
  { value: "重量鉄骨", label: "重量鉄骨（耐用34年）" },
  { value: "RC造", label: "RC造（耐用47年）" },
  { value: "SRC造", label: "SRC造（耐用47年）" },
];

export const RENT_DECLINE_DEFAULTS: Record<BuildingType, number> = {
  木造: 0.01,
  "軽量鉄骨(3mm以下)": 0.008,
  "軽量鉄骨(4mm以下)": 0.008,
  重量鉄骨: 0.008,
  RC造: 0.005,
  SRC造: 0.005,
};

export function getPeriodLabel(): string {
  const year = new Date().getFullYear();
  return `${year - 2}〜${year - 1}年`;
}
