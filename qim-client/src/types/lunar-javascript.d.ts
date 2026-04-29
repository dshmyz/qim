declare module 'lunar-javascript' {
  export class Solar {
    static fromYmd(year: number, month: number, day: number): Solar
    getLunar(): Lunar
  }

  export class Lunar {
    getDayInChinese(): string
    getMonthInChinese(): string
    getMonth(): number
    getDay(): number
    getJieQi(): string
    getYearInChinese(): string
    getFestivals(): string[]
  }
}