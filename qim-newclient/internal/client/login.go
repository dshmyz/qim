package client

// 原生登录页。此前登录靠网页里的 Login.vue 通过桥把 token 报给 Go，
// 现在表单、2FA、错误提示、发起请求全在 Go 侧，token 自始至终不出 qapi.Client。

import (
	"github.com/egoist/mygo/ui"
	"qim-newclient/internal/theme"
)

func (a *app) loginView(c *ui.Context) {
	ui.Column(c).Grow(1).FillWidth().Background(theme.Primary50).AlignItems(ui.Center).Center().Children(func() {
		ui.Column(c).Width(380).Padding(30, 34, 34, 34).Radius(14).Background(theme.White).
			Shadow(0, 12, 36, 0, theme.Shadow).Gap(12).Children(func() {
			ui.Text(c, "QIM 青雀").FontSize(24).FontWeight(600).TextColor(theme.Gray800)
			ui.Text(c, "登录到消息服务器").FontSize(13).TextColor(theme.Gray500)

			a.loginField(c, "服务器地址", "http://localhost:8080", &a.loginServer, false, "server")
			a.loginField(c, "账号", "用户名", &a.loginUser, false, "username")
			if a.need2FA {
				a.loginField(c, "二次验证", "6 位验证码", &a.loginCode, false, "code")
			} else {
				a.loginField(c, "密码", "密码", &a.loginPass, true, "password")
			}

			if a.loginErr != "" {
				ui.Text(c, a.loginErr).FontSize(12).TextColor(theme.ErrorText).MaxLines(3)
			}

			label := "登录"
			if a.loginBusy {
				label = "登录中…"
			} else if a.need2FA {
				label = "验证并登录"
			}
			btn := ui.Button(c, label).FillWidth().Height(38).Radius(8).FontSize(14).
				Background(theme.Primary).TextColor(theme.White).Disabled(a.loginBusy)
			if btn.Clicked() {
				a.submitLogin()
			}

			ui.Text(c, "按 Enter 也可以提交").FontSize(11).TextColor(theme.Gray500)
		})
	})
}

// loginField 是登录卡片里的一行输入框。Enter 直接提交，
// 密码用 mygo 的 Password() 遮蔽，不会明文回显到界面上。
func (a *app) loginField(c *ui.Context, title, placeholder string, value *string, secret bool, label string) {
	ui.Column(c).FillWidth().Gap(5).Children(func() {
		ui.Text(c, title).FontSize(12).TextColor(theme.Gray800)
		in := ui.TextInput(c, value).Label(label).Placeholder(placeholder).FillWidth().Height(36).
			FontSize(13).Radius(6).BorderWidth(1).BorderColor(theme.Gray200).Padding(8, 12)
		if secret {
			in = in.Password()
		}
		if label == "server" && a.loginUser == "" && a.loginPass == "" {
			in = in.AutoFocus()
		}
		if in.Submitted() {
			a.submitLogin()
		}
	})
}
