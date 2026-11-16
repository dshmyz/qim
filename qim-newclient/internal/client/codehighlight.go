package client

// 代码块支持：语法高亮 + 右键「复制代码」的正文提取。
//
// qim-client 没有独立的代码消息类型，代码块只是 markdown 文本里的 ``` 围栏，
// 渲染走 marked → <pre><code> → highlight.js（GitHub Light 配色）。它没有把
// 语言标识传给高亮器（主消息路径整体占位后手写 <pre>，language class 丢失），
// 所以实际是「按内容自动检测」。这里同样不依赖语言字段，做一个轻量词法扫描，
// 覆盖字符串/注释/数字/关键字/类型/函数/内置这些 hljs 的主要着色类别，配色
// 直接取 GitHub Light。刻意不引第三方高亮库：消息气泡里只需视觉接近，且要
// 能在 mygo 的 RichText 里逐 run 上色。

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// codeSpan 是一段带颜色的代码文本，交给 RichText 逐段渲染。
type codeSpan struct {
	Text  string
	Color ui.Color
}

// GitHub Light 高亮配色（对齐 highlight.js/styles/github.css）。
// 定义成变量而非 theme token：这是代码高亮专用色，不参与界面主题。
var (
	hlKeyword  = ui.Hex("#d73a49") // 关键字 / 类型声明 / doctag
	hlFunction = ui.Hex("#6f42c1") // 函数名 / 类名 / 标题
	hlAttr     = ui.Hex("#005cc5") // 属性 / 数字 / 操作符 / 变量 / 选择器
	hlString   = ui.Hex("#032f62") // 字符串 / 正则
	hlBuiltin  = ui.Hex("#e36209") // 内置函数 / 符号
	hlComment  = ui.Hex("#6a737d") // 注释
	hlTag      = ui.Hex("#22863a") // 标签 / 属性名
)

// 跨语言关键字并集：Go / JS / TS / Python / Java / C / SQL 的常见关键字。
// 与 hljs 自动检测的结果不会逐个一致，但视觉类别（关键字/字符串/数字/注释）
// 对齐即可，这正是原版渲染出来的样子。
var codeKeywords = map[string]bool{
	// 流程控制
	"if": true, "else": true, "for": true, "while": true, "do": true, "switch": true,
	"case": true, "default": true, "break": true, "continue": true, "return": true,
	"goto": true, "defer": true, "go": true, "fallthrough": true,
	// 声明与类型
	"func": true, "var": true, "const": true, "let": true, "type": true, "struct": true,
	"interface": true, "map": true, "chan": true, "package": true, "import": true,
	"class": true, "def": true, "lambda": true, "new": true, "extends": true,
	"public": true, "private": true, "protected": true, "static": true, "final": true,
	"async": true, "await": true, "yield": true, "try": true, "catch": true,
	"finally": true, "throw": true, "raise": true, "with": true, "as": true,
	"select": true, "from": true, "where": true, "insert": true, "update": true,
	"delete": true, "create": true, "table": true, "join": true, "on": true,
	"order": true, "by": true, "group": true, "having": true, "limit": true,
	"offset": true, "and": true, "or": true, "not": true, "in": true, "is": true,
	"null": true, "nil": true, "true": true, "false": true, "undefined": true,
	"this": true, "super": true, "self": true, "pass": true, "void": true,
}

// 常见基础类型与内置函数，配色更接近 hljs 的 built_in / type 类别。
var codeBuiltins = map[string]bool{
	"string": true, "int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"bool": true, "byte": true, "rune": true, "float32": true, "float64": true,
	"error": true, "any": true, "interface{}": true,
	"println": true, "print": true, "len": true, "cap": true, "append": true,
	"make": true, "new": true, "copy": true, "delete": true, "panic": true, "recover": true,
	"console": true, "log": true, "Math": true, "JSON": true, "Object": true, "Array": true,
	"str": true, "list": true, "dict": true, "set": true, "tuple": true, "range": true,
	"enumerate": true, "isinstance": true, "super": true, "System": true, "String": true,
	"count": true, "sum": true, "avg": true, "max": true, "min": true, "now": true,
}

// highlightCodeLine 把一行代码切成带颜色的片段。纯词法扫描，不校验语法：
// 消息气泡里的代码块只要"看起来像高亮过的代码"即可，不追求编译级准确。
func highlightCodeLine(line string, base ui.Color) []codeSpan {
	if line == "" {
		return []codeSpan{{Text: "", Color: base}}
	}
	var spans []codeSpan
	add := func(text string, color ui.Color) {
		if text == "" {
			return
		}
		// 相邻同色片段合并，减少 RichText 的 run 数。
		if n := len(spans); n > 0 && spans[n-1].Color == color {
			spans[n-1].Text += text
			return
		}
		spans = append(spans, codeSpan{Text: text, Color: color})
	}

	rs := []rune(line)
	for i := 0; i < len(rs); {
		c := rs[i]
		// 行注释：// 与 #（# 只在非字符串位置出现，字符串分支在前已处理）。
		if c == '/' && i+1 < len(rs) && rs[i+1] == '/' {
			add(string(rs[i:]), hlComment)
			break
		}
		if c == '#' {
			add(string(rs[i:]), hlComment)
			break
		}
		// 字符串：处理 \" 转义，记录结束位置以识别 JSON 的 "key": 属性名。
		if c == '"' || c == '\'' || c == '`' {
			q := c
			j := i + 1
			for j < len(rs) {
				if rs[j] == '\\' && j+1 < len(rs) {
					j += 2
					continue
				}
				if rs[j] == q {
					j++
					break
				}
				j++
			}
			// 结束引号后紧跟冒号 → JSON/Object 的属性名，用绿色（hljs attr）。
			after := j
			for after < len(rs) && unicode.IsSpace(rs[after]) {
				after++
			}
			color := hlString
			if after < len(rs) && rs[after] == ':' {
				color = hlTag
			}
			add(string(rs[i:j]), color)
			i = j
			continue
		}
		// 数字：0x 十六进制、十进制与小数、科学计数法 e/E。
		if unicode.IsDigit(c) {
			j := i + 1
			for j < len(rs) {
				d := rs[j]
				if unicode.IsDigit(d) || d == 'x' || d == 'X' || d == 'b' || d == 'B' ||
					d == 'o' || d == 'O' || d == '.' || d == '_' || d == 'e' || d == 'E' ||
					(d == '+' && j > i && (rs[j-1] == 'e' || rs[j-1] == 'E')) {
					j++
					continue
				}
				break
			}
			add(string(rs[i:j]), hlAttr)
			i = j
			continue
		}
		// 标识符：关键字 / 内置 / 后跟左括号的函数名 / 普通文本。
		if unicode.IsLetter(c) || c == '_' {
			j := i + 1
			for j < len(rs) && (unicode.IsLetter(rs[j]) || unicode.IsDigit(rs[j]) || rs[j] == '_') {
				j++
			}
			word := string(rs[i:j])
			k := j
			for k < len(rs) && unicode.IsSpace(rs[k]) {
				k++
			}
			color := base
			switch {
			case codeKeywords[word]:
				color = hlKeyword
			case codeBuiltins[word]:
				color = hlBuiltin
			case k < len(rs) && rs[k] == '(':
				color = hlFunction
			}
			add(word, color)
			i = j
			continue
		}
		// 操作符着色；标点与空白用基础色，保持代码的可读性。
		if strings.ContainsRune("=+-*/%<>!&|^~?:", c) {
			add(string(c), hlAttr)
		} else {
			add(string(c), base)
		}
		i++
	}
	return spans
}

// 与 qim-client utils/codeBlock.ts 的 extractCodeBlocks 对齐：
// 正则 /```[a-zA-Z0-9]*\n([\s\S]*?)```/g，取所有围栏的代码正文。
var fencedCodeRe = regexp.MustCompile("(?s)```[A-Za-z0-9]*\r?\n(.*?)```")

// extractCodeBlocks 返回消息原文里所有围栏代码块正文（不含围栏与语言标识，
// 去掉尾部换行）。复制代码时多块以空行连接，对齐 qim-client copyCode()。
func extractCodeBlocks(src string) []string {
	ms := fencedCodeRe.FindAllStringSubmatch(src, -1)
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, strings.TrimRight(m[1], "\r\n"))
	}
	return out
}

// hasCodeBlock 判断消息是否含围栏代码块（右键「复制代码」的显示条件）。
// RawContent 是 markdown 原文；退回 Text 兼容本地构造的消息。
func hasCodeBlock(m message) bool {
	return len(extractCodeBlocks(m.RawContent)) > 0 || len(extractCodeBlocks(m.Text)) > 0
}

// codeBlocksForCopy 取要复制的代码正文：优先 markdown 原文，退回 Text。
func codeBlocksForCopy(m message) string {
	blocks := extractCodeBlocks(m.RawContent)
	if len(blocks) == 0 {
		blocks = extractCodeBlocks(m.Text)
	}
	return strings.Join(blocks, "\n\n")
}

// codeBlockBackground 代码块底色：对齐原版 color-mix(text-color, transparent 94%)
// ≈ 文字色 6% 透明度。浅底气泡与对方气泡用同一套 tint（见 mdCodeBlock 注释）。
func codeBlockBackground(fg ui.Color) ui.Color {
	return fg.Alpha(0.06)
}
