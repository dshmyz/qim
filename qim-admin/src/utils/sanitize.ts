import DOMPurify from 'dompurify'

const MARKDOWN_CONFIG = {
  ALLOWED_TAGS: [
    'a', 'abbr', 'b', 'blockquote', 'br', 'code', 'del', 'details', 'div',
    'em', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'hr', 'i', 'img', 'kbd',
    'li', 'ol', 'p', 'pre', 's', 'span', 'strong', 'summary', 'table',
    'tbody', 'td', 'th', 'thead', 'tr', 'ul',
  ],
  ALLOWED_ATTR: ['href', 'title', 'target', 'rel', 'class', 'src', 'alt'],
}

export function sanitizeMarkdown(html: string): string {
  return DOMPurify.sanitize(html, MARKDOWN_CONFIG)
}
