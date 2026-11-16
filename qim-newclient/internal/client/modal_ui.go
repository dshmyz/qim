package client

import (
	"github.com/egoist/mygo/ui"
	"qim-newclient/internal/theme"
)

// modalMask 是所有弹层共用的全屏半透明底座（对齐原版 .modal-overlay）：
// 铺满当前窗口、绝对定位在左上角，点遮罩本身不关闭弹层。
// 居中的白色面板由调用方在 content 里画，样式由各弹层自己决定。
func modalMask(c *ui.Context, content func()) {
	ui.Box(c).FillWidth().FillHeight().Absolute().Left(0).Top(0).
		Background(theme.Mask).Children(content)
}
