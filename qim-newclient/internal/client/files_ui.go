package client

// 文件箱 UI（对齐 qim-client components/apps/FileManagementApp.vue 布局）：
// 头部（返回+标题+新建+上传）→ 筛选栏（面包屑+搜索+星标+排序+计数）→
// 左栏文件夹树+存储用量条 / 右侧文件列表（勾选/图标/名称/类型/大小/日期/星标/悬停操作）→
// 批量操作条 + 上传进度浮层 + 三个小弹窗（新建文件夹/移动/重命名）。

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
	"qim-newclient/internal/theme"
)

// filesView 整屏文件箱视图。
func (a *app) filesView(c *ui.Context) {
	ui.Column(c).Fill().Background(theme.Gray50).Children(func() {
		// 头部：返回 + 标题 + 新建文件夹 + 上传
		ui.Row(c).FillWidth().Height(56).Padding(0, 20).Gap(12).AlignItems(ui.Center).
			Background(theme.White).BorderWidth(0, 0, 1, 0).BorderColor(theme.Gray200).Children(func() {
			back := ui.Box(c).Height(32).Radius(6).Padding(0, 12).Center()
			if back.Clicked() {
				a.closeFiles()
			}
			back.Children(func() {
				ui.Text(c, "← 返回").FontSize(13).TextColor(theme.Primary)
			})
			ui.Icon(c, iconFolder).FontSize(17).TextColor(theme.Primary)
			ui.Text(c, "文件箱").FontWeight(600).FontSize(16).TextColor(theme.Gray800)
			ui.Spacer(c)
			btnHeaderAction(c, "新建文件夹", a.filesBusy, func() {
				a.mutate(func() {
					a.filesModal = "create"
					a.newFolderName = ""
				})
			})
			up := ui.Box(c).Height(26).Radius(6).Padding(0, 10).Background(theme.Primary).Center()
			if up.Clicked() {
				a.pickAndUpload()
			}
			up.Children(func() {
				ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconUpload).FontSize(12).TextColor(theme.White)
					ui.Text(c, "上传文件").FontSize(12).TextColor(theme.White)
				})
			})
		})
		// 筛选栏
		a.filesFilterBar(c)
		// 主体：左树 + 右列表
		ui.Row(c).Grow(1).FillWidth().Children(func() {
			a.filesTreePanel(c)
			a.filesListPanel(c)
		})
		// 批量操作条 + 上传浮层
		a.filesBatchBar(c)
		a.filesUploadOverlay(c)
		// 弹窗
		switch a.filesModal {
		case "create":
			a.filesCreateModal(c)
		case "move":
			a.filesMoveModal(c)
		case "rename":
			a.filesRenameModal(c)
		}
	})
}

// filesFilterBar 面包屑 + 搜索 + 星标 + 排序 + 计数。
func (a *app) filesFilterBar(c *ui.Context) {
	ui.Row(c).FillWidth().Height(44).Padding(0, 20).Gap(10).AlignItems(ui.Center).
		Background(theme.White).BorderWidth(0, 0, 1, 0).BorderColor(theme.Gray100).Children(func() {
		// 面包屑：全部文件 / seg…（星标/搜索视图隐藏路径，切回会重置）
		crumb := ui.Row(c).Gap(4).AlignItems(ui.Center)
		_ = crumb
		crumb.Children(func() {
			root := ui.Box(c).Height(24).Radius(5).Padding(0, 8).Center()
			if len(a.filesCrumb) == 0 {
				root = root.Background(theme.Primary50)
			}
			if root.Clicked() {
				a.enterFolder(-1, "")
			}
			root.Children(func() {
				ui.Text(c, "全部文件").FontSize(12).TextColor(theme.Gray800)
			})
			for i, seg := range a.filesCrumb {
				ui.Icon(c, iconChevRight).FontSize(10).TextColor(theme.Gray300)
				segBtn := ui.Box(c).Height(24).Radius(5).Padding(0, 8).Center()
				if i == len(a.filesCrumb)-1 {
					segBtn = segBtn.Background(theme.Primary50)
				}
				cur := seg
				if segBtn.Clicked() {
					a.enterFolder(cur.ID, cur.Name)
				}
				segBtn.Children(func() {
					ui.Text(c, seg.Name).FontSize(12).TextColor(theme.Gray800).
						MaxLines(1).Ellipsis("…").Width(110)
				})
			}
		})
		ui.Spacer(c)
		// 搜索框
		search := ui.Row(c).Width(180).Height(28).Radius(8).Background(theme.Gray100).
			Padding(0, 8).Gap(6).AlignItems(ui.Center)
		search.Children(func() {
			ui.Icon(c, iconSearch).FontSize(12).TextColor(theme.Gray500)
			ui.TextArea(c, &a.filesSearch).Placeholder("搜索文件名...").Grow(1).
				Height(28).MinHeight(28).FontSize(12).TextColor(theme.Gray800).Background(ui.Transparent).
				OnTextKey(func(k ui.TextKey) bool {
					if k.Key == ui.KeyEnter {
						a.applyFilesSearch()
						return true
					}
					return false
				})
			if a.filesSearchQ != "" {
				x := ui.Box(c).Width(16).Height(16).Center()
				if x.Clicked() {
					a.clearFilesSearch()
				}
				x.Children(func() {
					ui.Icon(c, iconClose).FontSize(10).TextColor(theme.Gray500)
				})
			}
		})
		// 星标过滤
		star := ui.Box(c).Width(28).Height(28).Radius(6).Center()
		starBg, starCol := theme.Gray100, theme.Gray500
		if a.filesStarred {
			starBg, starCol = theme.WarningBg, theme.Warning
		}
		star = star.Background(starBg)
		if star.Clicked() {
			a.setFilesStarred(!a.filesStarred)
		}
		star.Children(func() {
			ui.Icon(c, iconStar).FontSize(13).TextColor(starCol)
		})
		// 排序循环按钮
		sortBtn := ui.Box(c).Height(28).Radius(6).Padding(0, 10).Background(theme.Gray100).Center()
		if sortBtn.Clicked() {
			a.cycleSort()
		}
		sortBtn.Children(func() {
			ui.Text(c, filesSortPresets[a.filesSortIdx%len(filesSortPresets)].label).
				FontSize(12).TextColor(theme.Gray800)
		})
		// 计数
		ui.Textf(c, "%d 个文件", a.filesTotal).FontSize(12).TextColor(theme.Gray500)
	})
}

// filesTreePanel 左栏：文件夹树 + 存储用量条。
func (a *app) filesTreePanel(c *ui.Context) {
	ui.Column(c).Width(240).FillHeight().Background(theme.White).
		BorderWidth(0, 1, 0, 0).BorderColor(theme.Gray200).Children(func() {
		// 树（懒加载展开）
		ui.Column(c).Grow(1).FillWidth().Padding(8, 6).Gap(1).AlignItems(ui.Start).Children(func() {
			if len(a.treeRoot) == 0 && !a.filesLoading {
				ui.Text(c, "暂无文件夹").FontSize(12).TextColor(theme.Gray500).Padding(8, 8)
			}
			a.treeNodes(c, a.treeRoot, 0)
		})
		// 存储用量条（对齐 StorageUsageBar.vue）
		ui.Column(c).FillWidth().Padding(10, 14).Gap(6).
			BorderWidth(1, 0, 0, 0).BorderColor(theme.Gray200).Children(func() {
			ui.Row(c).FillWidth().Gap(5).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconFolder).FontSize(11).TextColor(theme.Gray500)
				ui.Textf(c, "已用 %s / %s", humanSize(a.filesUsageUsed), humanSize(a.filesUsageQuota)).
					FontSize(11).TextColor(theme.Gray500)
			})
			track := ui.Box(c).FillWidth().Height(4).Radius(2).Background(theme.Gray200)
			track.Children(func() {
				pct := 0.0
				if a.filesUsageQuota > 0 {
					pct = float64(a.filesUsageUsed) / float64(a.filesUsageQuota)
					if pct > 1 {
						pct = 1
					}
				}
				fillCol := theme.Primary
				if pct >= 0.9 {
					fillCol = theme.Error
				}
				// mygo 无百分比宽度，用 200px 基准按比例取整
				w := float32(pct * 200)
				if w < 0 {
					w = 0
				}
				ui.Box(c).Width(w).Height(4).Radius(2).Background(fillCol)
			})
		})
	})
}

// treeNodes 递归画文件夹树节点。
func (a *app) treeNodes(c *ui.Context, nodes []qapi.FolderItem, depth int) {
	for _, f := range nodes {
		node := f
		indent := 8 + depth*16
		open := a.treeExpanded[node.ID]
		row := ui.Row(c).FillWidth().Height(30).Padding(0, 6, 0, float32(indent)).Gap(5).
			AlignItems(ui.Center).Radius(6)
		selected := a.filesFolderID == node.ID && !a.filesStarred
		bg := ui.Transparent
		if selected {
			bg = theme.Primary50
		} else if row.Hovered() {
			bg = theme.Gray100
		}
		row = row.Background(bg)
		if row.Clicked() {
			a.enterFolder(node.ID, node.Name)
		}
		row.Children(func() {
			// 展开箭头（有子级才显示；点击只切展开不进入目录）
			if node.HasChildren {
				ch := ui.Box(c).Width(14).Height(14).Center()
				if ch.Clicked() {
					a.toggleTreeExpand(node.ID)
				}
				ch.Children(func() {
					icon := iconChevRight
					if open {
						icon = iconChevDown
					}
					ui.Icon(c, icon).FontSize(10).TextColor(theme.Gray500)
				})
			} else {
				ui.Box(c).Width(14).Height(14)
			}
			ui.Icon(c, iconFolder).FontSize(13).TextColor(theme.Primary)
			ui.Text(c, node.Name).FontSize(12).TextColor(theme.Gray800).
				MaxLines(1).Ellipsis("…").Grow(1)
		})
		if open {
			a.treeNodes(c, a.treeKids[node.ID], depth+1)
		}
	}
}

// filesListPanel 右侧文件列表。
func (a *app) filesListPanel(c *ui.Context) {
	ui.Column(c).Grow(1).FillHeight().Padding(12, 16).Children(func() {
		switch {
		case a.filesLoading && len(a.files) == 0:
			ui.Column(c).Grow(1).FillWidth().Center().Children(func() {
				ui.Text(c, "正在加载文件…").FontSize(13).TextColor(theme.Gray500)
			})
		case len(a.files) == 0:
			hint := "暂无文件\n点右上角「上传文件」试试"
			if a.filesStarred {
				hint = "暂无星标文件"
			} else if a.filesSearchQ != "" {
				hint = "没有匹配「" + a.filesSearchQ + "」的文件"
			}
			ui.Column(c).Grow(1).FillWidth().Center().Gap(10).Children(func() {
				ui.Icon(c, iconFolder).FontSize(40).TextColor(theme.Gray300)
				ui.Text(c, hint).FontSize(12).TextColor(theme.Gray500).TextAlign(ui.Center)
			})
		default:
			a.filesListState.Key = func(i int) any { return a.files[i].ID }
			ui.List(c, &a.filesListState, len(a.files), func(i int) {
				a.fileRow(c, a.files[i])
			}).Grow(1)
		}
	})
}

// fileRow 单个文件行：勾选框 + 图标 + 名称 + 类型 + 大小 + 日期 + 星标 + 悬停操作。
func (a *app) fileRow(c *ui.Context, f qapi.FileItem) {
	checked := a.filesSel[f.ID]
	row := ui.Row(c).FillWidth().Height(48).Padding(0, 12).Gap(10).AlignItems(ui.Center).
		Radius(8).Background(theme.White).BorderWidth(0, 0, 1, 0).BorderColor(theme.Gray100)
	if row.Hovered() {
		row = row.Background(theme.Primary50)
	}
	row.Children(func() {
		// 勾选框
		cb := ui.Box(c).Width(16).Height(16).Radius(4).
			BorderWidth(1).BorderColor(theme.Gray300).Center()
		if checked {
			cb = cb.Background(theme.Primary).BorderWidth(0)
		}
		if cb.Clicked() {
			a.toggleFileSel(f.ID)
		}
		cb.Children(func() {
			if checked {
				ui.Text(c, "✓").FontSize(10).TextColor(theme.White)
			}
		})
		// 类型图标（颜色对齐 qim-client getFileIconColor）
		ui.Icon(c, iconDoc).FontSize(16).TextColor(ui.Hex(fileKindColor(f.MimeType)))
		// 名称
		ui.Text(c, f.Name).FontSize(13).TextColor(theme.Gray800).
			MaxLines(1).Ellipsis("…").Grow(1)
		// 类型标签
		ui.Text(c, fileKindLabel(f.MimeType)).FontSize(11).TextColor(theme.Gray500).Width(48)
		// 大小
		ui.Text(c, humanSize(f.Size)).FontSize(12).TextColor(theme.Gray500).Width(64)
		// 日期
		ui.Text(c, chTimeLabel(f.UpdatedAt)).FontSize(12).TextColor(theme.Gray500).Width(96)
		// 星标
		if f.IsStarred {
			ui.Icon(c, iconStar).FontSize(13).TextColor(theme.Warning)
		} else {
			ui.Box(c).Width(13).Height(13)
		}
		// 悬停操作：下载 / 星标 / 重命名 / 删除
		acts := ui.Row(c).Gap(2)
		if row.Hovered() || checked {
			acts.Children(func() {
				a.fileActBtn(c, iconDownload, "下载", func() { a.downloadFile(f) })
				a.fileActBtn(c, iconStar, "星标", func() { a.toggleFileStar(f.ID) })
				a.fileActBtn(c, iconMore, "重命名", func() { a.beginRenameFile(f.ID, f.Name) })
				a.fileActBtn(c, iconTrash, "删除", func() { a.deleteFileNow(f.ID) })
			})
		} else {
			acts.Width(96)
		}
	})
}

// fileActBtn 文件行上的小操作按钮。
func (a *app) fileActBtn(c *ui.Context, icon *ui.SVG, title string, onClick func()) {
	btn := ui.Box(c).Width(22).Height(22).Radius(5).Center()
	if btn.Clicked() {
		onClick()
	}
	btn.Children(func() {
		ui.Icon(c, icon).FontSize(13).TextColor(theme.Gray500)
	})
}

// filesBatchBar 底部批量操作条（有勾选时出现）。
func (a *app) filesBatchBar(c *ui.Context) {
	sel := len(a.filesSel)
	if sel == 0 {
		return
	}
	// Absolute 容器不按内容撑高，显式给高度
	ui.Box(c).FillWidth().Height(46).Absolute().Left(0).Bottom(0).Children(func() {
		ui.Row(c).Fill().Height(46).Padding(0, 20).Gap(10).AlignItems(ui.Center).
			Background(theme.White).BorderWidth(1, 0, 0, 0).BorderColor(theme.Gray200).Children(func() {
			ui.Textf(c, "已选 %d 项", sel).FontSize(13).FontWeight(600).TextColor(theme.Gray800)
			ui.Spacer(c)
			a.batchBtn(c, "添加星标", false, func() { a.batchStarSelected(true) })
			a.batchBtn(c, "取消星标", false, func() { a.batchStarSelected(false) })
			a.batchBtn(c, "移动到…", false, func() { a.beginMoveFiles(0) })
			a.batchBtn(c, "删除", true, func() { a.batchDeleteSelected() })
			a.batchBtn(c, "取消选择", false, func() { a.clearFileSel() })
		})
	})
}

// batchBtn 批量条按钮；danger 红字。
func (a *app) batchBtn(c *ui.Context, label string, danger bool, onClick func()) {
	col := theme.Gray800
	if danger {
		col = theme.Error
	}
	btn := ui.Box(c).Height(28).Radius(6).Padding(0, 12).Background(theme.Gray100).Center()
	if btn.Clicked() {
		onClick()
	}
	btn.Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(col)
	})
}

// filesUploadOverlay 右下角上传进度浮层（对齐 UploadProgressBar.vue）。
func (a *app) filesUploadOverlay(c *ui.Context) {
	if len(a.uploadTasks) == 0 {
		return
	}
	// Absolute 容器不按内容撑高：头部 26 + 每任务（标题行+进度条+操作行）58 + padding 20
	h := 46 + len(a.uploadTasks)*58
	// 批量操作条出现时让出它的 46px 高度
	bottom := float32(16)
	if sel := len(a.filesSel); sel > 0 {
		bottom = 62
	}
	ui.Box(c).Width(300).Height(float32(h)).Absolute().Right(16).Bottom(bottom).Children(func() {
		ui.Column(c).Fill().Radius(10).Background(theme.White).Shadow(0, 4, 16, 0, theme.Shadow).
			Padding(10, 12).Gap(8).Children(func() {
			ui.Text(c, "上传队列").FontSize(12).FontWeight(600).TextColor(theme.Gray800)
			for _, t := range a.uploadTasks {
				task := t
				ui.Column(c).FillWidth().Gap(4).Children(func() {
					ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Text(c, task.name).FontSize(12).TextColor(theme.Gray800).
							MaxLines(1).Ellipsis("…").Grow(1)
						switch task.state {
						case 1:
							ui.Text(c, "完成").FontSize(11).TextColor(theme.Online)
						case 2:
							ui.Text(c, "失败").FontSize(11).TextColor(theme.Error)
						case 3:
							ui.Text(c, "已取消").FontSize(11).TextColor(theme.Gray500)
						default:
							ui.Textf(c, "%d%%", task.percent()).FontSize(11).TextColor(theme.Primary)
						}
					})
					// 进度条
					ui.Box(c).FillWidth().Height(4).Radius(2).Background(theme.Gray200).Children(func() {
						if task.state == 0 {
							w := float32(float64(task.percent()) / 100 * 274)
							ui.Box(c).Width(w).Height(4).Radius(2).Background(theme.Primary)
						}
					})
					if task.state == 2 {
						ui.Text(c, task.err).FontSize(11).TextColor(theme.Error).
							MaxLines(1).Ellipsis("…")
					}
					// 操作：上传中→取消；已结束→移除
					ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
						ui.Spacer(c)
						var op ui.Element
						if task.state == 0 {
							op = ui.Box(c).Height(18).Radius(4).Padding(0, 6).Center()
							if op.Clicked() {
								a.cancelUploadTask(task.id)
							}
							op.Children(func() {
								ui.Text(c, "取消").FontSize(11).TextColor(theme.Gray500)
							})
						} else {
							op = ui.Box(c).Height(18).Radius(4).Padding(0, 6).Center()
							if op.Clicked() {
								a.dismissUploadTask(task.id)
							}
							op.Children(func() {
								ui.Text(c, "移除").FontSize(11).TextColor(theme.Gray500)
							})
						}
					})
				})
			}
		})
	})
}

// filesCreateModal 新建文件夹弹窗。
func (a *app) filesCreateModal(c *ui.Context) {
	modalMask(c, func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Column(c).Width(360).Radius(12).Background(theme.White).
				Padding(20, 20).Gap(12).Children(func() {
				ui.Text(c, "新建文件夹").FontSize(15).FontWeight(600).TextColor(theme.Gray800)
				where := "全部文件"
				if len(a.filesCrumb) > 0 {
					where = a.filesCrumb[len(a.filesCrumb)-1].Name
				}
				ui.Textf(c, "位置：%s", where).FontSize(12).TextColor(theme.Gray500)
				ui.Box(c).FillWidth().Height(34).Radius(8).Background(theme.Gray100).Padding(0, 10).Children(func() {
					ui.TextArea(c, &a.newFolderName).Placeholder("文件夹名称").FillWidth().
						Height(34).MinHeight(34).FontSize(13).TextColor(theme.Gray800).Background(ui.Transparent)
				})
				ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Spacer(c)
					btnHeaderAction(c, "取消", false, func() { a.mutate(func() { a.filesModal = "" }) })
					ok := ui.Box(c).Height(26).Radius(6).Padding(0, 12).Background(theme.Primary).Center()
					disabled := strings.TrimSpace(a.newFolderName) == "" || a.filesBusy
					if ok.Clicked() && !disabled {
						a.submitCreateFolder()
					}
					okCol := theme.White
					if disabled {
						okCol = theme.WhiteOverlay
					}
					ok.Children(func() {
						ui.Text(c, "创建").FontSize(12).TextColor(okCol)
					})
				})
			})
		})
	})
}

// filesMoveModal 移动文件弹窗：目标文件夹单选（含根目录）。
func (a *app) filesMoveModal(c *ui.Context) {
	folders := a.filesTreeFlat()
	modalMask(c, func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Column(c).Width(360).MaxHeight(460).Radius(12).Background(theme.White).
				Padding(20, 20).Gap(12).Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
					ui.Text(c, "移动文件").FontSize(15).FontWeight(600).TextColor(theme.Gray800)
					ui.Spacer(c)
					x := ui.Box(c).Width(24).Height(24).Radius(6).Center()
					if x.Clicked() {
						a.mutate(func() { a.filesModal = "" })
					}
					x.Children(func() {
						ui.Icon(c, iconClose).FontSize(14).TextColor(theme.Gray500)
					})
				})
				// 目标单选列表（后端 batch move 只收真实文件夹，不含根目录）
				ui.Column(c).FillWidth().Grow(1).Gap(2).AlignItems(ui.Start).Children(func() {
					for _, f := range folders {
						a.moveTargetRow(c, f.ID, f.Name, 0)
					}
				})
				ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Spacer(c)
					btnHeaderAction(c, "取消", false, func() { a.mutate(func() { a.filesModal = "" }) })
					ok := ui.Box(c).Height(26).Radius(6).Padding(0, 12).Background(theme.Primary).Center()
					if ok.Clicked() {
						a.confirmMoveFiles()
					}
					ok.Children(func() {
						ui.Text(c, "移动").FontSize(12).TextColor(theme.White)
					})
				})
			})
		})
	})
}

// moveTargetRow 移动弹窗里的目标行。
func (a *app) moveTargetRow(c *ui.Context, id int64, name string, indent int) {
	row := ui.Row(c).FillWidth().Height(32).Padding(0, 10, 0, float32(8+indent)).Gap(8).
		AlignItems(ui.Center).Radius(6)
	if a.moveSel == id {
		row = row.Background(theme.Primary50)
	} else if row.Hovered() {
		row = row.Background(theme.Gray100)
	}
	if row.Clicked() {
		a.mutate(func() { a.moveSel = id })
	}
	row.Children(func() {
		radio := ui.Box(c).Width(14).Height(14).Radius(7).
			BorderWidth(1).BorderColor(theme.Gray300).Center()
		if a.moveSel == id {
			radio = radio.Background(theme.Primary).BorderWidth(0)
		}
		radio.Children(func() {
			if a.moveSel == id {
				ui.Box(c).Width(6).Height(6).Radius(3).Background(theme.White)
			}
		})
		ui.Icon(c, iconFolder).FontSize(12).TextColor(theme.Primary)
		ui.Text(c, name).FontSize(12).TextColor(theme.Gray800).MaxLines(1).Ellipsis("…").Grow(1)
	})
}

// filesRenameModal 重命名弹窗。
func (a *app) filesRenameModal(c *ui.Context) {
	modalMask(c, func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Column(c).Width(360).Radius(12).Background(theme.White).
				Padding(20, 20).Gap(12).Children(func() {
				ui.Text(c, "重命名").FontSize(15).FontWeight(600).TextColor(theme.Gray800)
				ui.Box(c).FillWidth().Height(34).Radius(8).Background(theme.Gray100).Padding(0, 10).Children(func() {
					ui.TextArea(c, &a.renameName).Placeholder("文件名").FillWidth().
						Height(34).MinHeight(34).FontSize(13).TextColor(theme.Gray800).Background(ui.Transparent)
				})
				ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Spacer(c)
					btnHeaderAction(c, "取消", false, func() { a.mutate(func() { a.filesModal = "" }) })
					ok := ui.Box(c).Height(26).Radius(6).Padding(0, 12).Background(theme.Primary).Center()
					if ok.Clicked() && strings.TrimSpace(a.renameName) != "" {
						a.confirmRenameFile()
					}
					ok.Children(func() {
						ui.Text(c, "保存").FontSize(12).TextColor(theme.White)
					})
				})
			})
		})
	})
}
