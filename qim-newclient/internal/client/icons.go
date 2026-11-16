// 图标构造与全部 SVG 图标变量。
// 从 main.go 按职责拆出，逻辑未改动。
package client

import (
	"github.com/egoist/mygo/ui"
)

// mustIcon 用 lucide 风格的 path 构造一个描边图标（工具栏、小图标用）。
func mustIcon(inner string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + inner + `</svg>`))
}

// mustSolid 构造一个实心图标，对齐 qim-client 左栏用的 Font Awesome 实心风格：
// 18px 也有足够的视觉重量，和原版导航栏观感一致。
func mustSolid(inner string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor">` + inner + `</svg>`))
}

var (
	// 左栏导航：实心风格（对齐 Font Awesome 图标）
	iconChat     = mustSolid(`<path fill-rule="evenodd" d="M12 2.5C6.2 2.5 1.5 6.4 1.5 11.3c0 2.8 1.5 5.3 3.9 6.9V22l4.2-2.2c.8.1 1.6.2 2.4.2 5.8 0 10.5-3.9 10.5-8.7S17.8 2.5 12 2.5zm-5 10a1.4 1.4 0 1 1 0-2.8 1.4 1.4 0 0 1 0 2.8zm5 0a1.4 1.4 0 1 1 0-2.8 1.4 1.4 0 0 1 0 2.8zm5 0a1.4 1.4 0 1 1 0-2.8 1.4 1.4 0 0 1 0 2.8z"/>`)
	iconContacts = mustSolid(`<rect x="8.5" y="2" width="7" height="5.5" rx="1.5"/><rect x="1.5" y="16.5" width="7" height="5.5" rx="1.5"/><rect x="15.5" y="16.5" width="7" height="5.5" rx="1.5"/><rect x="11" y="7.5" width="2" height="5.4"/><rect x="5" y="12.6" width="14" height="2"/><rect x="5" y="13.2" width="2" height="3.4"/><rect x="17" y="13.2" width="2" height="3.4"/>`)
	iconUsers    = mustSolid(`<circle cx="8.5" cy="7" r="4"/><path d="M1 20.5c0-3.6 3.4-6.5 7.5-6.5s7.5 2.9 7.5 6.5v.5H1z"/><circle cx="17.5" cy="8" r="3.2"/><path d="M16.2 13.5c3.7.3 6.3 2.7 6.3 5.5v2h-4.6v-.5c0-2.7-1.1-5-3-6.6.4-.2.9-.3 1.3-.4z"/>`)
	iconUserPlus = mustIcon(`<circle cx="9" cy="8" r="4"/><path d="M2 21c0-3.9 3.1-7 7-7 1.5 0 2.9.5 4 1.3"/><line x1="19" y1="15" x2="19" y2="21"/><line x1="16" y1="18" x2="22" y2="18"/>`)
	iconApps     = mustSolid(`<path fill-rule="evenodd" d="M12 2 3 6.8v10.4L12 22l9-4.8V6.8L12 2zm0 2.6 6.4 3.4L12 11.4 5.6 8 12 4.6zM5 9.7l6 3.2v6.5l-6-3.2V9.7zm14 0v6.5l-6 3.2v-6.5l6-3.2z"/>`)
	iconChannel  = mustSolid(`<path d="M4 9.5h2.6L14 4.2v15.6l-7.4-5.3H4a1.5 1.5 0 0 1-1.5-1.5v-2A1.5 1.5 0 0 1 4 9.5z"/><path d="M16.6 8.3a5.4 5.4 0 0 1 0 7.4l-1.5-1.5a3.3 3.3 0 0 0 0-4.4z"/>`)
	iconPalette  = mustSolid(`<path fill-rule="evenodd" d="M12 2C6.5 2 2 6.5 2 12s4.5 10 10 10c.9 0 1.6-.7 1.6-1.6 0-.4-.15-.8-.4-1.1-.25-.3-.4-.7-.4-1.1a1.6 1.6 0 0 1 1.6-1.6h1.9c3.1 0 5.7-2.6 5.7-5.7C22 6.3 17.6 2 12 2zM7 14.2a1.5 1.5 0 1 1 0-3 1.5 1.5 0 0 1 0 3zm2.4-4.6a1.5 1.5 0 1 1 0-3 1.5 1.5 0 0 1 0 3zm5.2 0a1.5 1.5 0 1 1 0-3 1.5 1.5 0 0 1 0 3z"/>`)
	iconSettings = mustSolid(`<path fill-rule="evenodd" d="M10.3 2h3.4l.4 2.4c.7.2 1.4.5 2 .9l2.2-1.1 1.7 2.9-1.8 1.6c.1.4.2.9.2 1.3s-.1.9-.2 1.3l1.8 1.6-1.7 2.9-2.2-1.1c-.6.4-1.3.7-2 .9L13.7 22h-3.4l-.4-2.4c-.7-.2-1.4-.5-2-.9l-2.2 1.1-1.7-2.9 1.8-1.6c-.1-.4-.2-.9-.2-1.3s.1-.9.2-1.3L4 11.1l1.7-2.9 2.2 1.1c.6-.4 1.3-.7 2-.9L10.3 2zM12 8.6a3.4 3.4 0 1 0 0 6.8 3.4 3.4 0 0 0 0-6.8z"/>`)
	iconComments = mustSolid(`<path d="M8.5 2C4.4 2 1 5 1 8.7c0 1.9.9 3.6 2.4 4.8V17l3.2-1.7c.5.1 1.1.2 1.6.2 0-.3-.1-.6-.1-.9 0-3.8 3.4-6.9 7.6-6.9h.6C15.6 4.5 12.4 2 8.5 2zm9.7 6.2c-3.1 0-5.7 2.3-5.7 5.1 0 1.5.8 2.8 2 3.7v2.5l2.7-1.4c.3.1.7.1 1 .1 3.1 0 5.7-2.3 5.7-5.1s-2.6-4.9-5.7-4.9z"/>`)
	iconExpand   = mustIcon(`<polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/><line x1="21" x2="14" y1="3" y2="10"/><line x1="3" x2="10" y1="21" y2="14"/><line x1="15" x2="21" y1="3" y2="9"/><line x1="3" x2="10" y1="15" y2="21"/>`)

	// 窗口三色灯里的白色小图形
	iconLampMinus  = mustIcon(`<line x1="5" x2="19" y1="12" y2="12"/>`)
	iconLampSquare = mustIcon(`<rect x="7" y="7" width="10" height="10" rx="1.5"/>`)
	iconLampClose  = mustIcon(`<line x1="7" x2="17" y1="7" y2="17"/><line x1="17" x2="7" y1="7" y2="17"/>`)

	// 聊天区/输入区/成员面板用的小图标
	iconMonitor   = mustIcon(`<rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" x2="16" y1="21" y2="21"/><line x1="12" x2="12" y1="17" y2="21"/>`)
	iconGrid      = mustIcon(`<rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/>`)
	iconClock     = mustIcon(`<circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>`)
	iconChevDown  = mustIcon(`<polyline points="6 9 12 15 18 9"/>`)
	iconChevRight = mustIcon(`<polyline points="9 6 15 12 9 18"/>`)
	iconZap       = mustSolid(`<path d="M13 2 3 14h7l-1 8 10-12h-7l1-8z"/>`)
	iconSearch    = mustIcon(`<circle cx="11" cy="11" r="7"/><line x1="21" x2="16.5" y1="21" y2="16.5"/>`)
	iconCrown     = mustSolid(`<path d="M3 8l4 4 5-6 5 6 4-4v9H3V8z"/>`)
	iconChevLeft  = mustIcon(`<polyline points="15 18 9 12 15 6"/>`)
	iconClose     = mustIcon(`<line x1="6" x2="18" y1="6" y2="18"/><line x1="18" x2="6" y1="6" y2="18"/>`)

	// 小图标：描边风格（对齐 qim-client 头部的 FA 线性小图标）
	iconBell      = mustIcon(`<path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/><path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/>`)
	iconBellSlash = mustIcon(`<path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/><path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/><line x1="2" y1="2" x2="22" y2="22"/>`)
	iconPencil    = mustIcon(`<path d="M12 20h9"/><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"/>`)
	iconVolume    = mustIcon(`<path d="M11 5 6 9H2v6h4l5 4V5Z"/><path d="M15.5 8.5a5 5 0 0 1 0 7"/><path d="M18.5 5.5a9 9 0 0 1 0 13"/>`)
	iconPlus      = mustIcon(`<path d="M5 12h14"/><path d="M12 5v14"/>`)
	iconCompress  = mustIcon(`<polyline points="4 14 10 14 10 20"/><polyline points="20 10 14 10 14 4"/><line x1="14" x2="21" y1="10" y2="3"/><line x1="3" x2="10" y1="21" y2="14"/>`)
	iconBot       = mustSolid(`<path d="M12 2a1.6 1.6 0 0 1 1.6 1.6V5h2.9A3.5 3.5 0 0 1 20 8.5v6A3.5 3.5 0 0 1 16.5 18h-9A3.5 3.5 0 0 1 4 14.5v-6A3.5 3.5 0 0 1 7.5 5h2.9V3.6A1.6 1.6 0 0 1 12 2zM8.6 10.2a1.4 1.4 0 1 0 0 2.8 1.4 1.4 0 0 0 0-2.8zm6.8 0a1.4 1.4 0 1 0 0 2.8 1.4 1.4 0 0 0 0-2.8zm-6.5 5.2h6.2a3.1 3.1 0 0 1-6.2 0z"/><rect x="1.5" y="10" width="2" height="4" rx="1"/><rect x="20.5" y="10" width="2" height="4" rx="1"/>`)
	iconFolder    = mustIcon(`<path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/>`)
	// AI 侧边栏输入框：发送（纸飞机）与停止生成（方块）。
	iconSend  = mustIcon(`<path d="M21 3 3 10.5l7 2.5 2.5 7L21 3z"/><path d="M10 13 21 3"/>`)
	iconStop  = mustIcon(`<rect x="6" y="6" width="12" height="12" rx="2"/>`)
	iconLink  = mustIcon(`<path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>`)
	iconMore  = mustIcon(`<circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/>`)
	iconPhone = mustIcon(`<path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z"/>`)
	iconVideo = mustIcon(`<path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2"/>`)
	iconSmile = mustIcon(`<circle cx="12" cy="12" r="10"/><path d="M8 14s1.5 2 4 2 4-2 4-2"/><line x1="9" x2="9.01" y1="9" y2="9"/><line x1="15" x2="15.01" y1="9" y2="9"/>`)
	iconClip  = mustIcon(`<path d="m21.44 11.05-9.19 9.19a6 6 0 0 1-8.49-8.49l8.57-8.57A4 4 0 1 1 18 8.84l-8.59 8.57a2 2 0 0 1-2.83-2.83l8.49-8.48"/>`)
	iconImage = mustIcon(`<rect width="18" height="18" x="3" y="3" rx="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/>`)
	iconCode  = mustIcon(`<polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>`)
	iconPin   = mustIcon(`<path d="M12 17v5"/><path d="M9 10.76a2 2 0 0 1-1.11 1.79l-1.78.9A2 2 0 0 0 5 15.24V16a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-.76a2 2 0 0 0-1.11-1.79l-1.78-.9A2 2 0 0 1 15 10.76V7a1 1 0 0 1 1-1 2 2 0 0 0 0-4H8a2 2 0 0 0 0 4 1 1 0 0 1 1 1z"/>`)
	iconStar  = mustIcon(`<polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>`)

	// 文件箱（D 组）小图标
	iconUpload   = mustIcon(`<path d="M12 16V4"/><path d="m6 9 6-6 6 6"/><path d="M4 20h16"/>`)
	iconDownload = mustIcon(`<path d="M12 4v12"/><path d="m6 11 6 6 6-6"/><path d="M4 20h16"/>`)
	iconTrash    = mustIcon(`<path d="M3 6h18"/><path d="M8 6V4a1 1 0 0 1 1-1h6a1 1 0 0 1 1 1v2"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/><line x1="10" x2="10" y1="11" y2="17"/><line x1="14" x2="14" y1="11" y2="17"/>`)
	iconDoc      = mustIcon(`<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6"/>`)
)
