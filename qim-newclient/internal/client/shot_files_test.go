package client

// D 组文件网盘渲染截图：文件箱主视图（默认/含批量条）、上传浮层、移动弹窗。

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"qim-newclient/internal/qapi"
)

func TestShotFilesView(t *testing.T) {
	a := &app{
		active:        -1,
		loggedIn:      true,
		filesOpen:     true,
		filesLoaded:   true,
		filesFolderID: -1,
		treeKids:      make(map[int64][]qapi.FolderItem),
		treeExpanded:  make(map[int64]bool),
		filesSel:      make(map[int64]bool),
		treeRoot: []qapi.FolderItem{
			{ID: 3, Name: "工作文档", HasChildren: true},
			{ID: 4, Name: "随手记"},
		},
		files: []qapi.FileItem{
			{ID: 7, Name: "季度经营报告-2026Q4-final.pdf", Size: 10485760, MimeType: "application/pdf", IsStarred: true, UpdatedAt: "2026-11-15T10:00:00+08:00"},
			{ID: 8, Name: "产品封面图.png", Size: 409600, MimeType: "image/png", UpdatedAt: "2026-11-14T09:00:00+08:00"},
			{ID: 9, Name: "会议纪要.txt", Size: 512, MimeType: "text/plain", UpdatedAt: "2026-11-13T08:00:00+08:00"},
			{ID: 10, Name: "备份.zip", Size: 536870912, MimeType: "application/zip", UpdatedAt: "2026-11-12T08:00:00+08:00"},
		},
		filesTotal:      4,
		filesUsageUsed:  3 << 30,
		filesUsageQuota: 10 << 30,
		filesSortIdx:    0,
	}
	a.treeKids[3] = []qapi.FolderItem{{ID: 30, Name: "周报", ParentID: ptrInt64(3)}}
	a.treeExpanded[3] = true
	a.msgs = []message{}
	a.convs = []conversation{{ID: "1", Name: "测试群"}}

	tt := ui.NewTester(a.view, 1366, 850)
	tt.Frame()
	tt.Frame()
	write(t, tt, "/tmp/qim-files.png")

	// 含批量选择条 + 上传浮层
	a.filesSel[7] = true
	a.filesSel[8] = true
	a.uploadTasks = []*fileUploadTask{
		{id: 1, name: "大文件备份.iso", size: 1 << 30, done: 3 << 28, state: 0},
		{id: 2, name: "已完成.txt", size: 100, done: 100, state: 1},
	}
	tt.Frame()
	tt.Frame()
	write(t, tt, "/tmp/qim-files-batch.png")

	// 移动弹窗
	a.filesModal = "move"
	tt.Frame()
	tt.Frame()
	write(t, tt, "/tmp/qim-files-move.png")
}
