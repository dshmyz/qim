enum LogLevel {
  DEBUG = 0,
  INFO = 1,
  WARN = 2,
  ERROR = 3
}

interface LogConfig {
  level: LogLevel
  enableConsole: boolean
  prefix: string
}

class Logger {
  private config: LogConfig

  constructor(config: Partial<LogConfig> = {}) {
    // 开发环境默认 DEBUG，生产环境默认 ERROR（只记录错误）
    const isDev = import.meta.env.DEV
    this.config = {
      level: config.level ?? (isDev ? LogLevel.DEBUG : LogLevel.ERROR),
      enableConsole: config.enableConsole ?? isDev,
      prefix: config.prefix ?? '[QIM]'
    }
  }

  private formatMessage(level: string, ...args: any[]): string {
    return `${this.config.prefix} [${level}] ${new Date().toISOString()}`
  }

  debug(...args: any[]): void {
    if (this.config.level <= LogLevel.DEBUG && this.config.enableConsole) {
      console.debug(this.formatMessage('DEBUG'), ...args)
    }
  }

  log(...args: any[]): void {
    if (this.config.level <= LogLevel.INFO && this.config.enableConsole) {
      console.log(this.formatMessage('INFO'), ...args)
    }
  }

  info(...args: any[]): void {
    if (this.config.level <= LogLevel.INFO && this.config.enableConsole) {
      console.info(this.formatMessage('INFO'), ...args)
    }
  }

  warn(...args: any[]): void {
    if (this.config.level <= LogLevel.WARN && this.config.enableConsole) {
      console.warn(this.formatMessage('WARN'), ...args)
    }
  }

  error(...args: any[]): void {
    if (this.config.level <= LogLevel.ERROR && this.config.enableConsole) {
      console.error(this.formatMessage('ERROR'), ...args)
    }
  }

  group(label: string): void {
    if (this.config.enableConsole) {
      console.group(this.formatMessage('GROUP'), label)
    }
  }

  groupEnd(): void {
    if (this.config.enableConsole) {
      console.groupEnd()
    }
  }

  table(tabularData: any, columns?: string[]): void {
    if (this.config.level <= LogLevel.DEBUG && this.config.enableConsole) {
      console.table(tabularData, columns)
    }
  }

  setLevel(level: LogLevel): void {
    this.config.level = level
  }

  enable(): void {
    this.config.enableConsole = true
  }

  disable(): void {
    this.config.enableConsole = false
  }
}

export const logger = new Logger()
export { Logger, LogLevel }
export default logger
