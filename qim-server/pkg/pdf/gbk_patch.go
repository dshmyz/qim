package pdf

import (
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// gbkEncoder 按 GBK/GB18030 解码内容流中的双字节字符码。
//
// 知网/万方等中文论文 PDF 的 Type0 字体常声明 /Encoding /GBK-EUC-H（或
// GBK-EUC-V/GBKp-EUC-H/GBK2K-H 等 GBK 系 CMap）且不带 ToUnicode CMap，
// 此时内容流中写入的字符码就是原始 GBK 字节——按 GBK 解码即可还原正文。
// 无效字节序列由解码器替换为 U+FFFD（与库内其他解码器行为一致，不报错）。
type gbkEncoder struct{}

func (e *gbkEncoder) Decode(raw string) (text string) {
	out, _ := simplifiedchinese.GBK.NewDecoder().Bytes([]byte(raw))
	return string(out)
}

// isGBKNamedEncoding 判断具名编码是否属于 GBK 系 CMap（字符码即 GBK 字节）。
func isGBKNamedEncoding(name string) bool {
	switch name {
	case "GBK-EUC-H", "GBK-EUC-V", "GBKp-EUC-H", "GBKp-EUC-V", "GBK2K-H", "GBK2K-V", "GBK-LATIN":
		return true
	}
	return strings.HasPrefix(name, "GBK")
}
