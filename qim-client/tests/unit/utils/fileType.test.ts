import { describe, it, expect } from 'vitest'
import {
  getFileCategory,
  getFileIcon,
  formatFileSize,
  isImageFile,
  isVideoFile,
  isAudioFile,
  isDocumentFile,
  isSpreadsheetFile,
  isPresentationFile,
  isArchiveFile,
  isCodeFile,
  type FileCategory,
} from '@/utils/fileType'

describe('getFileCategory', () => {
  describe('图片类型', () => {
    const imageTypes = [
      'image/jpeg',
      'image/png',
      'image/gif',
      'image/webp',
      'image/svg+xml',
      'image/bmp',
      'image/tiff',
      'image/x-icon',
      'image/avif',
      'image/heic',
      'image/heif',
    ]

    it.each(imageTypes)('应该将 %s 识别为 image', (mimeType) => {
      expect(getFileCategory(mimeType)).toBe('image')
    })

    it('应该将未知 image/* 类型识别为 image（前缀匹配）', () => {
      expect(getFileCategory('image/x-photoshop')).toBe('image')
      expect(getFileCategory('image/jp2')).toBe('image')
    })
  })

  describe('视频类型', () => {
    const videoTypes = [
      'video/mp4',
      'video/webm',
      'video/ogg',
      'video/quicktime',
      'video/x-msvideo',
      'video/x-matroska',
      'video/x-flv',
      'video/x-ms-wmv',
      'video/3gpp',
      'video/3gpp2',
      'video/avi',
      'video/mpeg',
      'video/mov',
    ]

    it.each(videoTypes)('应该将 %s 识别为 video', (mimeType) => {
      expect(getFileCategory(mimeType)).toBe('video')
    })

    it('应该将未知 video/* 类型识别为 video（前缀匹配）', () => {
      expect(getFileCategory('video/x-custom')).toBe('video')
    })
  })

  describe('音频类型', () => {
    const audioTypes = [
      'audio/mpeg',
      'audio/wav',
      'audio/ogg',
      'audio/webm',
      'audio/mp4',
      'audio/flac',
      'audio/aac',
      'audio/x-wav',
      'audio/x-flac',
      'audio/midi',
      'audio/x-midi',
      'audio/x-aiff',
      'audio/aacp',
      'audio/opus',
    ]

    it.each(audioTypes)('应该将 %s 识别为 audio', (mimeType) => {
      expect(getFileCategory(mimeType)).toBe('audio')
    })

    it('应该将未知 audio/* 类型识别为 audio（前缀匹配）', () => {
      expect(getFileCategory('audio/x-custom')).toBe('audio')
    })
  })

  describe('文档类型', () => {
    it('应该识别 PDF 为 document', () => {
      expect(getFileCategory('application/pdf')).toBe('document')
    })

    it('应该识别 Word 文档为 document', () => {
      expect(getFileCategory('application/msword')).toBe('document')
      expect(
        getFileCategory(
          'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
        )
      ).toBe('document')
    })

    it('应该识别纯文本为 document', () => {
      expect(getFileCategory('text/plain')).toBe('document')
    })

    it('应该识别 Markdown 为 document', () => {
      expect(getFileCategory('text/markdown')).toBe('document')
    })

    it('应该识别 CSV 为 document', () => {
      expect(getFileCategory('text/csv')).toBe('document')
    })

    it('应该识别 RTF 为 document', () => {
      expect(getFileCategory('text/rtf')).toBe('document')
      expect(getFileCategory('application/rtf')).toBe('document')
    })
  })

  describe('电子表格类型', () => {
    it('应该识别 Excel 97-2003 为 spreadsheet', () => {
      expect(getFileCategory('application/vnd.ms-excel')).toBe('spreadsheet')
    })

    it('应该识别 Excel OOXML 为 spreadsheet', () => {
      expect(
        getFileCategory(
          'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
        )
      ).toBe('spreadsheet')
    })

    it('应该识别 ODS 为 spreadsheet', () => {
      expect(
        getFileCategory('application/vnd.oasis.opendocument.spreadsheet')
      ).toBe('spreadsheet')
    })
  })

  describe('演示文稿类型', () => {
    it('应该识别 PowerPoint 97-2003 为 presentation', () => {
      expect(getFileCategory('application/vnd.ms-powerpoint')).toBe('presentation')
    })

    it('应该识别 PowerPoint OOXML 为 presentation', () => {
      expect(
        getFileCategory(
          'application/vnd.openxmlformats-officedocument.presentationml.presentation'
        )
      ).toBe('presentation')
    })

    it('应该识别 ODP 为 presentation', () => {
      expect(
        getFileCategory('application/vnd.oasis.opendocument.presentation')
      ).toBe('presentation')
    })
  })

  describe('压缩包类型', () => {
    const archiveTypes = [
      'application/zip',
      'application/x-tar',
      'application/gzip',
      'application/x-7z-compressed',
      'application/x-rar-compressed',
      'application/x-bzip2',
      'application/x-xz',
      'application/vnd.rar',
      'application/java-archive',
    ]

    it.each(archiveTypes)('应该将 %s 识别为 archive', (mimeType) => {
      expect(getFileCategory(mimeType)).toBe('archive')
    })
  })

  describe('代码类型', () => {
    it('应该识别 JSON 为 code', () => {
      expect(getFileCategory('application/json')).toBe('code')
    })

    it('应该识别 XML 为 code', () => {
      expect(getFileCategory('application/xml')).toBe('code')
      expect(getFileCategory('text/xml')).toBe('code')
    })

    it('应该识别 JavaScript 为 code', () => {
      expect(getFileCategory('application/javascript')).toBe('code')
      expect(getFileCategory('text/javascript')).toBe('code')
    })

    it('应该识别 HTML 为 code', () => {
      expect(getFileCategory('text/html')).toBe('code')
    })

    it('应该识别 CSS 为 code', () => {
      expect(getFileCategory('text/css')).toBe('code')
    })

    it('应该识别各种编程语言为 code', () => {
      expect(getFileCategory('text/x-python')).toBe('code')
      expect(getFileCategory('text/x-java')).toBe('code')
      expect(getFileCategory('text/x-c')).toBe('code')
      expect(getFileCategory('text/x-c++')).toBe('code')
      expect(getFileCategory('text/x-go')).toBe('code')
      expect(getFileCategory('text/x-ruby')).toBe('code')
      expect(getFileCategory('text/x-rust')).toBe('code')
    })
  })

  describe('字体类型', () => {
    const fontTypes = [
      'font/woff',
      'font/woff2',
      'font/ttf',
      'font/otf',
      'application/x-font-ttf',
      'application/x-font-otf',
    ]

    it.each(fontTypes)('应该将 %s 识别为 font', (mimeType) => {
      expect(getFileCategory(mimeType)).toBe('font')
    })

    it('应该将未知 font/* 类型识别为 font（前缀匹配）', () => {
      expect(getFileCategory('font/collection')).toBe('font')
    })
  })

  describe('文本类型', () => {
    it('应该将未知 text/* 类型识别为 text（前缀匹配）', () => {
      expect(getFileCategory('text/x-custom')).toBe('text')
      expect(getFileCategory('text/calendar')).toBe('text')
    })
  })

  describe('边界情况', () => {
    it('应该将空字符串识别为 unknown', () => {
      expect(getFileCategory('')).toBe('unknown')
    })

    it('应该将 undefined 识别为 unknown', () => {
      expect(getFileCategory(undefined as unknown as string)).toBe('unknown')
    })

    it('应该将 null 识别为 unknown', () => {
      expect(getFileCategory(null as unknown as string)).toBe('unknown')
    })

    it('应该将无法识别的 MIME 类型识别为 unknown', () => {
      expect(getFileCategory('application/octet-stream')).toBe('unknown')
      expect(getFileCategory('unknown/type')).toBe('unknown')
      expect(getFileCategory('application/x-custom')).toBe('unknown')
    })

    it('应该忽略大小写', () => {
      expect(getFileCategory('IMAGE/PNG')).toBe('image')
      expect(getFileCategory('Video/Mp4')).toBe('video')
      expect(getFileCategory('APPLICATION/PDF')).toBe('document')
    })

    it('应该忽略前后空格', () => {
      expect(getFileCategory('  image/png  ')).toBe('image')
      expect(getFileCategory(' video/mp4 ')).toBe('video')
    })
  })
})

describe('getFileIcon', () => {
  it('应该为图片返回正确的图标', () => {
    expect(getFileIcon('image/png')).toBe('fa-solid fa-file-image')
    expect(getFileIcon('image/jpeg')).toBe('fa-solid fa-file-image')
  })

  it('应该为视频返回正确的图标', () => {
    expect(getFileIcon('video/mp4')).toBe('fa-solid fa-file-video')
  })

  it('应该为音频返回正确的图标', () => {
    expect(getFileIcon('audio/mpeg')).toBe('fa-solid fa-file-audio')
  })

  it('应该为文档返回正确的图标', () => {
    expect(getFileIcon('application/pdf')).toBe('fa-solid fa-file-lines')
    expect(getFileIcon('text/plain')).toBe('fa-solid fa-file-lines')
  })

  it('应该为电子表格返回正确的图标', () => {
    expect(getFileIcon('application/vnd.ms-excel')).toBe('fa-solid fa-file-excel')
  })

  it('应该为演示文稿返回正确的图标', () => {
    expect(getFileIcon('application/vnd.ms-powerpoint')).toBe('fa-solid fa-file-powerpoint')
  })

  it('应该为压缩包返回正确的图标', () => {
    expect(getFileIcon('application/zip')).toBe('fa-solid fa-file-zipper')
  })

  it('应该为代码返回正确的图标', () => {
    expect(getFileIcon('application/json')).toBe('fa-solid fa-file-code')
    expect(getFileIcon('text/javascript')).toBe('fa-solid fa-file-code')
  })

  it('应该为字体返回正确的图标', () => {
    expect(getFileIcon('font/woff2')).toBe('fa-solid fa-font')
  })

  it('应该为未知类型返回通用文件图标', () => {
    expect(getFileIcon('application/octet-stream')).toBe('fa-solid fa-file')
    expect(getFileIcon('')).toBe('fa-solid fa-file')
    expect(getFileIcon('unknown/type')).toBe('fa-solid fa-file')
  })
})

describe('formatFileSize', () => {
  it('应该将 0 字节格式化为 "0 B"', () => {
    expect(formatFileSize(0)).toBe('0 B')
  })

  it('应该正确格式化字节单位', () => {
    expect(formatFileSize(500)).toBe('500 B')
    expect(formatFileSize(1023)).toBe('1023 B')
  })

  it('应该正确格式化 KB 单位', () => {
    expect(formatFileSize(1024)).toBe('1 KB')
    expect(formatFileSize(1536)).toBe('1.5 KB')
    expect(formatFileSize(10240)).toBe('10 KB')
    expect(formatFileSize(51200)).toBe('50 KB')
  })

  it('应该正确格式化 MB 单位', () => {
    expect(formatFileSize(1048576)).toBe('1 MB')
    expect(formatFileSize(1572864)).toBe('1.5 MB')
    expect(formatFileSize(10485760)).toBe('10 MB')
  })

  it('应该正确格式化 GB 单位', () => {
    expect(formatFileSize(1073741824)).toBe('1 GB')
    expect(formatFileSize(1610612736)).toBe('1.5 GB')
    expect(formatFileSize(10737418240)).toBe('10 GB')
  })

  it('应该正确格式化 TB 单位', () => {
    expect(formatFileSize(1099511627776)).toBe('1 TB')
    expect(formatFileSize(5497558138880)).toBe('5 TB')
  })

  it('应该将负数格式化为 "0 B"', () => {
    expect(formatFileSize(-1)).toBe('0 B')
    expect(formatFileSize(-1024)).toBe('0 B')
  })

  it('应该将 Infinity 格式化为 "0 B"', () => {
    expect(formatFileSize(Infinity)).toBe('0 B')
    expect(formatFileSize(-Infinity)).toBe('0 B')
  })

  it('应该将 NaN 格式化为 "0 B"', () => {
    expect(formatFileSize(NaN)).toBe('0 B')
  })

  it('应该正确格式化边界值', () => {
    // 刚好是单位边界
    expect(formatFileSize(1024)).toBe('1 KB')
    expect(formatFileSize(1024 * 1024)).toBe('1 MB')
    expect(formatFileSize(1024 * 1024 * 1024)).toBe('1 GB')
  })
})

describe('isImageFile', () => {
  it('应该对图片 MIME 类型返回 true', () => {
    expect(isImageFile('image/png')).toBe(true)
    expect(isImageFile('image/jpeg')).toBe(true)
    expect(isImageFile('image/gif')).toBe(true)
    expect(isImageFile('image/webp')).toBe(true)
  })

  it('应该对非图片 MIME 类型返回 false', () => {
    expect(isImageFile('video/mp4')).toBe(false)
    expect(isImageFile('application/pdf')).toBe(false)
    expect(isImageFile('application/octet-stream')).toBe(false)
    expect(isImageFile('')).toBe(false)
  })
})

describe('isVideoFile', () => {
  it('应该对视频 MIME 类型返回 true', () => {
    expect(isVideoFile('video/mp4')).toBe(true)
    expect(isVideoFile('video/webm')).toBe(true)
    expect(isVideoFile('video/quicktime')).toBe(true)
  })

  it('应该对非视频 MIME 类型返回 false', () => {
    expect(isVideoFile('image/png')).toBe(false)
    expect(isVideoFile('audio/mpeg')).toBe(false)
    expect(isVideoFile('')).toBe(false)
  })
})

describe('isAudioFile', () => {
  it('应该对音频 MIME 类型返回 true', () => {
    expect(isAudioFile('audio/mpeg')).toBe(true)
    expect(isAudioFile('audio/wav')).toBe(true)
    expect(isAudioFile('audio/flac')).toBe(true)
  })

  it('应该对非音频 MIME 类型返回 false', () => {
    expect(isAudioFile('video/mp4')).toBe(false)
    expect(isAudioFile('image/png')).toBe(false)
    expect(isAudioFile('')).toBe(false)
  })
})

describe('isDocumentFile', () => {
  it('应该对文档 MIME 类型返回 true', () => {
    expect(isDocumentFile('application/pdf')).toBe(true)
    expect(isDocumentFile('text/plain')).toBe(true)
    expect(isDocumentFile('text/markdown')).toBe(true)
  })

  it('应该对非文档 MIME 类型返回 false', () => {
    expect(isDocumentFile('image/png')).toBe(false)
    expect(isDocumentFile('application/zip')).toBe(false)
    expect(isDocumentFile('')).toBe(false)
  })
})

describe('isSpreadsheetFile', () => {
  it('应该对电子表格 MIME 类型返回 true', () => {
    expect(isSpreadsheetFile('application/vnd.ms-excel')).toBe(true)
    expect(
      isSpreadsheetFile(
        'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
      )
    ).toBe(true)
  })

  it('应该对非电子表格 MIME 类型返回 false', () => {
    expect(isSpreadsheetFile('application/pdf')).toBe(false)
    expect(isSpreadsheetFile('')).toBe(false)
  })
})

describe('isPresentationFile', () => {
  it('应该对演示文稿 MIME 类型返回 true', () => {
    expect(isPresentationFile('application/vnd.ms-powerpoint')).toBe(true)
    expect(
      isPresentationFile(
        'application/vnd.openxmlformats-officedocument.presentationml.presentation'
      )
    ).toBe(true)
  })

  it('应该对非演示文稿 MIME 类型返回 false', () => {
    expect(isPresentationFile('application/pdf')).toBe(false)
    expect(isPresentationFile('')).toBe(false)
  })
})

describe('isArchiveFile', () => {
  it('应该对压缩包 MIME 类型返回 true', () => {
    expect(isArchiveFile('application/zip')).toBe(true)
    expect(isArchiveFile('application/gzip')).toBe(true)
    expect(isArchiveFile('application/x-7z-compressed')).toBe(true)
  })

  it('应该对非压缩包 MIME 类型返回 false', () => {
    expect(isArchiveFile('application/pdf')).toBe(false)
    expect(isArchiveFile('')).toBe(false)
  })
})

describe('isCodeFile', () => {
  it('应该对代码 MIME 类型返回 true', () => {
    expect(isCodeFile('application/json')).toBe(true)
    expect(isCodeFile('text/javascript')).toBe(true)
    expect(isCodeFile('text/html')).toBe(true)
    expect(isCodeFile('text/css')).toBe(true)
  })

  it('应该对非代码 MIME 类型返回 false', () => {
    expect(isCodeFile('image/png')).toBe(false)
    expect(isCodeFile('application/pdf')).toBe(false)
    expect(isCodeFile('')).toBe(false)
  })
})
