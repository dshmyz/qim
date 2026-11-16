package client

import (
	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/model"
	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
)

// avatarStyle 是头像的显示规格。Radius / Weight / Bg 留空时取默认值，
// 这样大多数调用点只需写 Size 和 Font，特殊样式再显式覆盖。
type avatarStyle struct {
	Size   float32 // 边长
	Font   float32 // 首字字号
	Bg     string  // 兜底底色，留空按名字散列
	Radius float32 // 圆角，留空为 Size/2（正圆）
	Weight int     // 首字字重，留空为 500
	Border bool    // 1px 浅灰描边（消息头像用）
	Shrink bool    // 禁止被父行挤压
}

// avatar 统一画头像本体：服务端给了 URL 就走图片缓存，否则退化成
// 彩色底 + 白色首字（对齐 shared/Avatar.vue 的 src 回退）。
// 需要点击、状态点或类型角标的调用方，在外面再包一层 Box 即可。
func (a *app) avatar(c *ui.Context, name, url string, st avatarStyle) {
	radius, weight, bg := st.Radius, st.Weight, st.Bg
	if radius == 0 {
		radius = st.Size / 2
	}
	if weight == 0 {
		weight = 500
	}
	if bg == "" {
		bg = avatarColor(name)
	}
	box := ui.Box(c).Width(st.Size).Height(st.Size).Radius(radius).Background(ui.Hex(bg)).Center()
	if st.Border {
		box = box.BorderWidth(1).BorderColor(theme.Gray200)
	}
	if st.Shrink {
		box = box.Shrink(0)
	}
	box.Children(func() {
		if url != "" {
			if src, _, _ := a.imageState(qapi.Attachment{URL: url}, ""); src != nil {
				ui.Image(c, src).Size(st.Size, st.Size).Radius(radius)
				return
			}
		}
		ui.Text(c, model.FirstRune(name)).TextColor(theme.White).FontWeight(weight).FontSize(st.Font)
	})
}
