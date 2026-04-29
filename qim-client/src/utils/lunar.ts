import { Solar } from 'lunar-javascript'

export interface LunarDayInfo {
  lunarDayName: string
  lunarMonthName: string
  festival: string | null
  solarTerm: string | null
  isFestival: boolean
  isSolarTerm: boolean
}

const LUNAR_FESTIVALS: Record<string, string> = {
  '正月初一': '春节',
  '正月十五': '元宵节',
  '五月初五': '端午节',
  '七月初七': '七夕节',
  '七月十五': '中元节',
  '八月十五': '中秋节',
  '九月初九': '重阳节',
  '腊月初八': '腊八节',
  '腊月三十': '除夕',
  '腊月廿九': '除夕',
}

const SOLAR_FESTIVALS: Record<string, string> = {
  '1-1': '元旦',
  '2-14': '情人节',
  '3-8': '妇女节',
  '3-12': '植树节',
  '4-1': '愚人节',
  '5-1': '劳动节',
  '5-4': '青年节',
  '6-1': '儿童节',
  '7-1': '建党节',
  '8-1': '建军节',
  '9-10': '教师节',
  '10-1': '国庆节',
  '10-2': '国庆节',
  '10-3': '国庆节',
  '12-25': '圣诞节',
}

export function getLunarDayInfo(year: number, month: number, day: number): LunarDayInfo {
  const solar = Solar.fromYmd(year, month, day)
  const lunar = solar.getLunar()

  const lunarDayName = lunar.getDayInChinese()
  const lunarMonthName = lunar.getMonthInChinese()

  const key = `${lunarMonthName}${lunarDayName}`
  const solarKey = `${month}-${day}`

  const lunarFestival = LUNAR_FESTIVALS[key] || null
  const solarFestival = SOLAR_FESTIVALS[solarKey] || null
  const festival = lunarFestival || solarFestival || null

  const solarTerm = lunar.getJieQi() || null

  const displayText = festival || solarTerm || lunarDayName

  return {
    lunarDayName: displayText,
    lunarMonthName,
    festival,
    solarTerm,
    isFestival: festival !== null,
    isSolarTerm: solarTerm !== null,
  }
}

export function getLunarMonthDay(year: number, month: number, day: number): string {
  const info = getLunarDayInfo(year, month, day)
  return info.lunarDayName
}