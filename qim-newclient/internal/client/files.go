package client

// 文件箱（对齐 qim-client components/apps/FileManagementApp.vue + composables/useFileUpload.ts）：
// 个人文件网盘 = 文件列表（文件夹过滤/搜索/星标/排序）+ 文件夹树（懒加载展开）+
// 分片上传队列（并发 + 重试 + 取消 + 进度）+ 下载到本地 + 批量操作（星标/移动/删除）。

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"

	"qim-newclient/internal/qapi"
)

// filesSortPresets 排序预设（对齐 FileManagementApp 的下拉四项，循环切换）。
var filesSortPresets = []struct{ sortBy, sortOrder, label string }{
	{"created_at", "desc", "最新优先"},
	{"created_at", "asc", "最早优先"},
	{"name", "asc", "名称 A→Z"},
	{"name", "desc", "名称 Z→A"},
}

// fileUploadTask 一次分片上传任务的进度状态（UI 浮层渲染用）。
type fileUploadTask struct {
	id       int
	name     string
	size     int64
	done     int64 // 已传字节数
	state    int   // 0 上传中 / 1 完成 / 2 失败 / 3 已取消
	err      string
	uploadID string
	cancel   atomic.Bool
}

// percent 上传进度百分比。
func (t *fileUploadTask) percent() int {
	if t.size <= 0 {
		return 100
	}
	p := int(t.done * 100 / t.size)
	if p > 100 {
		p = 100
	}
	return p
}

// openFiles 打开整屏文件箱；首次进入拉文件夹树 + 根目录文件 + 存储用量。
func (a *app) openFiles() {
	a.mutate(func() {
		a.settingsOpen = false
		a.notifsOpen = false
		a.channelsOpen = false
		a.filesOpen = true
	})
	if a.filesLoaded {
		return
	}
	a.filesLoadAll()
}

// closeFiles 关闭文件箱；目录位置与列表保留，回来还在原地。
func (a *app) closeFiles() {
	a.mutate(func() { a.filesOpen = false })
}

// resetFiles 登录/登出时清空文件箱全部状态。
func (a *app) resetFiles() {
	a.filesOpen = false
	a.filesLoaded = false
	a.filesLoading = false
	a.files = nil
	a.filesTotal = 0
	a.filesFolderID = -1
	a.filesCrumb = nil
	a.treeRoot = nil
	a.treeKids = make(map[int64][]qapi.FolderItem)
	a.treeExpanded = make(map[int64]bool)
	a.filesStarred = false
	a.filesSortIdx = 0
	a.filesSearch = ""
	a.filesSearchQ = ""
	a.filesSel = make(map[int64]bool)
	a.filesBusy = false
	a.filesModal = ""
	a.newFolderName = ""
	a.renameID = 0
	a.renameName = ""
	a.moveSel = 0
	a.moveOneID = 0
	a.uploadTasks = nil
	a.uploadSeq = 0
	a.filesUsageUsed = 0
	a.filesUsageQuota = 0
}

// filesLoadAll 首次进入的三个并发加载：树 / 文件 / 用量。
func (a *app) filesLoadAll() {
	if a.client == nil {
		return
	}
	a.mutate(func() { a.filesLoading = true })
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		a.filesReloadTree()
	}()
	go func() {
		defer wg.Done()
		a.filesReloadList()
	}()
	go func() {
		defer wg.Done()
		a.filesReloadUsage()
	}()
	wg.Wait()
	a.mutate(func() {
		a.filesLoading = false
		a.filesLoaded = true
	})
}

// filesReloadList 按当前过滤条件拉文件列表。
func (a *app) filesReloadList() {
	if a.client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	p := qapi.FileListParams{
		FolderID:  a.filesFolderID,
		Starred:   a.filesStarred,
		Search:    a.filesSearchQ,
		SortBy:    filesSortPresets[a.filesSortIdx%len(filesSortPresets)].sortBy,
		SortOrder: filesSortPresets[a.filesSortIdx%len(filesSortPresets)].sortOrder,
		Page:      1,
		PageSize:  100,
	}
	out, err := a.client.ListFiles(ctx, p)
	if err != nil {
		a.setNotice("加载文件失败：" + friendlyError(err))
		return
	}
	a.mutate(func() {
		a.files = out.Files
		a.filesTotal = out.Total
	})
}

// filesReloadTree 拉文件夹树根层（懒加载，子层按需展开）。
func (a *app) filesReloadTree() {
	if a.client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	folders, err := a.client.FolderTree(ctx, -1)
	if err != nil {
		return // 树失败不提示，文件列表仍可用
	}
	a.mutate(func() {
		a.treeRoot = folders
	})
}

// filesReloadUsage 拉存储用量。
func (a *app) filesReloadUsage() {
	if a.client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	used, quota, err := a.client.StorageUsage(ctx)
	if err != nil {
		return
	}
	a.mutate(func() {
		a.filesUsageUsed = used
		a.filesUsageQuota = quota
	})
}

// toggleTreeExpand 展开/收起文件夹；首次展开时懒加载子层。
func (a *app) toggleTreeExpand(id int64) {
	a.mutate(func() {
		if a.treeExpanded[id] {
			delete(a.treeExpanded, id)
			return
		}
		a.treeExpanded[id] = true
	})
	if _, ok := a.treeKids[id]; ok {
		return
	}
	if a.client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	kids, err := a.client.FolderTree(ctx, id)
	if err != nil {
		return
	}
	a.mutate(func() {
		a.treeKids[id] = kids
	})
}

// enterFolder 点面包屑或树节点进入文件夹（id=-1 回根目录）。
func (a *app) enterFolder(id int64, name string) {
	a.mutate(func() {
		a.filesFolderID = id
		a.filesStarred = false
		a.filesSearchQ = ""
		a.filesSearch = ""
		a.filesSel = make(map[int64]bool)
		if id < 0 {
			a.filesCrumb = nil
		} else {
			a.filesCrumb = a.folderPathTo(id, name)
		}
	})
	a.filesReloadList()
}

// folderPathTo 从树缓存反查 id 的路径（含自身）；找不到时退化为单段。
// 树是懒加载的，深层目录可能只拿到自身一段，展开后再点即可补全。
func (a *app) folderPathTo(id int64, name string) []qapi.FolderItem {
	var path []qapi.FolderItem
	var walk func(nodes []qapi.FolderItem) bool
	walk = func(nodes []qapi.FolderItem) bool {
		for _, f := range nodes {
			path = append(path, f)
			if f.ID == id {
				return true
			}
			if walk(a.treeKids[f.ID]) {
				return true
			}
			path = path[:len(path)-1]
		}
		return false
	}
	if walk(a.treeRoot) {
		return path
	}
	return []qapi.FolderItem{{ID: id, Name: name}}
}

// setFilesStarred 切换「只看星标」（跨文件夹全局视图）。
func (a *app) setFilesStarred(on bool) {
	a.mutate(func() {
		a.filesStarred = on
		a.filesSel = make(map[int64]bool)
	})
	a.filesReloadList()
}

// cycleSort 循环切换排序预设并重拉列表。
func (a *app) cycleSort() {
	a.mutate(func() { a.filesSortIdx = (a.filesSortIdx + 1) % len(filesSortPresets) })
	a.filesReloadList()
}

// applyFilesSearch 应用搜索词（回车触发）。
func (a *app) applyFilesSearch() {
	a.mutate(func() { a.filesSearchQ = strings.TrimSpace(a.filesSearch) })
	a.filesReloadList()
}

// clearFilesSearch 清空搜索。
func (a *app) clearFilesSearch() {
	a.mutate(func() {
		a.filesSearch = ""
		a.filesSearchQ = ""
	})
	a.filesReloadList()
}

// submitCreateFolder 在当前目录下新建文件夹。
func (a *app) submitCreateFolder() {
	name := strings.TrimSpace(a.newFolderName)
	if name == "" || a.filesBusy {
		return
	}
	a.mutate(func() { a.filesBusy = true })
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()
		parent := a.filesFolderID
		if parent < 0 {
			parent = 0
		}
		_, err := a.client.CreateFolder(ctx, name, parent)
		a.mutate(func() {
			a.filesBusy = false
			a.filesModal = ""
			a.newFolderName = ""
		})
		if err != nil {
			a.setNotice("新建文件夹失败：" + friendlyError(err))
			return
		}
		a.filesReloadTree()
		a.filesReloadList()
		log.Printf("[files] 新建文件夹 %q (parent=%d)", name, parent)
	}()
}

// filesTreeFlat 平铺全部文件夹（移动目标选择用）：深度优先展开缓存里的树。
func (a *app) filesTreeFlat() []qapi.FolderItem {
	var out []qapi.FolderItem
	var walk func(nodes []qapi.FolderItem, depth int)
	walk = func(nodes []qapi.FolderItem, depth int) {
		for _, f := range nodes {
			out = append(out, f)
			walk(a.treeKids[f.ID], depth+1)
		}
	}
	walk(a.treeRoot, 0)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// beginMoveFiles 打开移动弹窗（ids 为空表示移动当前选中集）。
func (a *app) beginMoveFiles(oneID int64) {
	a.mutate(func() {
		a.filesModal = "move"
		a.moveSel = 0
		a.moveOneID = oneID
	})
}

// confirmMoveFiles 执行移动（BatchOperation move；目标必须是真实文件夹）。
func (a *app) confirmMoveFiles() {
	if a.filesBusy || a.moveSel <= 0 {
		return
	}
	ids := a.filesSelectedIDs()
	if a.moveOneID > 0 {
		ids = []int64{a.moveOneID}
	}
	if len(ids) == 0 {
		return
	}
	a.mutate(func() { a.filesBusy = true })
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
		defer cancel()
		err := a.client.BatchOperation(ctx, "move", ids, a.moveSel)
		a.mutate(func() {
			a.filesBusy = false
			a.filesModal = ""
			a.filesSel = make(map[int64]bool)
		})
		if err != nil {
			a.setNotice("移动失败：" + friendlyError(err))
			return
		}
		a.filesReloadList()
		log.Printf("[files] 移动 %d 个文件到 %d", len(ids), a.moveSel)
	}()
}

// confirmRenameFile 重命名文件。
func (a *app) confirmRenameFile() {
	name := strings.TrimSpace(a.renameName)
	if name == "" || a.renameID <= 0 || a.filesBusy {
		return
	}
	a.mutate(func() { a.filesBusy = true })
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()
		err := a.client.UpdateFile(ctx, a.renameID, name, nil)
		a.mutate(func() {
			a.filesBusy = false
			a.filesModal = ""
			a.renameID = 0
			a.renameName = ""
		})
		if err != nil {
			a.setNotice("重命名失败：" + friendlyError(err))
			return
		}
		a.filesReloadList()
	}()
}

// beginRenameFile 打开重命名弹窗。
func (a *app) beginRenameFile(id int64, name string) {
	a.mutate(func() {
		a.filesModal = "rename"
		a.renameID = id
		a.renameName = name
	})
}

// toggleFileStar 单文件星标切换（服务端 toggle）。
func (a *app) toggleFileStar(id int64) {
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()
		if _, err := a.client.ToggleStar(ctx, id); err != nil {
			a.setNotice("星标失败：" + friendlyError(err))
			return
		}
		a.filesReloadList()
	}()
}

// deleteFileNow 删除单个文件（软删除，后端可恢复，不做二次确认）。
func (a *app) deleteFileNow(id int64) {
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
		defer cancel()
		if err := a.client.DeleteFile(ctx, id); err != nil {
			a.setNotice("删除失败：" + friendlyError(err))
			return
		}
		a.mutate(func() { delete(a.filesSel, id) })
		a.filesReloadList()
		a.filesReloadUsage()
	}()
}

// filesSelectedIDs 当前勾选的文件 id（稳定顺序：按列表出现序）。
func (a *app) filesSelectedIDs() []int64 {
	var ids []int64
	for _, f := range a.files {
		if a.filesSel[f.ID] {
			ids = append(ids, f.ID)
		}
	}
	return ids
}

// toggleFileSel 勾选/取消勾选一个文件。
func (a *app) toggleFileSel(id int64) {
	a.mutate(func() {
		if a.filesSel[id] {
			delete(a.filesSel, id)
		} else {
			a.filesSel[id] = true
		}
	})
}

// clearFileSel 清空勾选。
func (a *app) clearFileSel() {
	a.mutate(func() { a.filesSel = make(map[int64]bool) })
}

// batchStarSelected 批量加/取消星标。
func (a *app) batchStarSelected(star bool) {
	ids := a.filesSelectedIDs()
	if len(ids) == 0 || a.filesBusy {
		return
	}
	op := "unstar"
	if star {
		op = "star"
	}
	a.mutate(func() { a.filesBusy = true })
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
		defer cancel()
		err := a.client.BatchOperation(ctx, op, ids, 0)
		a.mutate(func() { a.filesBusy = false })
		if err != nil {
			a.setNotice("批量星标失败：" + friendlyError(err))
			return
		}
		a.filesReloadList()
	}()
}

// batchDeleteSelected 批量删除。
func (a *app) batchDeleteSelected() {
	ids := a.filesSelectedIDs()
	if len(ids) == 0 || a.filesBusy {
		return
	}
	a.mutate(func() { a.filesBusy = true })
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
		defer cancel()
		err := a.client.BatchOperation(ctx, "delete", ids, 0)
		a.mutate(func() {
			a.filesBusy = false
			a.filesSel = make(map[int64]bool)
		})
		if err != nil {
			a.setNotice("批量删除失败：" + friendlyError(err))
			return
		}
		a.filesReloadList()
		a.filesReloadUsage()
	}()
}

// —— 分片上传队列 ——

// pickAndUpload 弹多选文件框 → 逐个入上传队列。Dialog.Open 必须在非 UI 线程。
func (a *app) pickAndUpload() {
	if a.client == nil || !a.loggedIn {
		return
	}
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:       "选择要上传的文件",
			ButtonLabel: "上传",
			Multiple:    true,
			Parent:      a.win,
		})
		if err != nil || len(paths) == 0 {
			return
		}
		for _, p := range paths {
			a.startFileUpload(p)
		}
	}()
}

// startFileUpload 为一个本地文件建上传任务并起分片 worker。
func (a *app) startFileUpload(path string) {
	st, err := os.Stat(path)
	if err != nil {
		a.setNotice("读取文件失败：" + err.Error())
		return
	}
	t := &fileUploadTask{
		id:   0,
		name: filepath.Base(path),
		size: st.Size(),
	}
	a.mutate(func() {
		a.uploadSeq++
		t.id = a.uploadSeq
		a.uploadTasks = append(a.uploadTasks, t)
	})
	go a.uploadWorker(t, path)
}

// uploadWorker 分片上传主流程：init → 并发传分片（每片重试 3 次）→ complete。
// folderID 捕获启动时刻的当前目录（上传中切目录不影响归属）。
func (a *app) uploadWorker(t *fileUploadTask, path string) {
	if a.client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Minute)
	defer cancel()

	folderID := a.filesFolderID
	if folderID < 0 {
		folderID = 0
	}

	init, err := a.client.InitChunkUpload(ctx, t.name, t.size, folderID, "")
	if err != nil {
		a.mutate(func() {
			t.state = 2
			t.err = friendlyError(err)
		})
		return
	}
	a.mutate(func() { t.uploadID = init.UploadID })

	// 单文件直接走普通上传通道（后端 /upload 上限 64MB；分片通道服务小文件也兼容，
	// 但为了进度一致，这里统一走分片协议）。
	chunkSize := init.ChunkSize
	total := init.TotalChunks
	if total <= 0 {
		total = int((t.size + chunkSize - 1) / chunkSize)
	}

	f, err := os.Open(path)
	if err != nil {
		a.mutate(func() { t.state = 2; t.err = "打开文件: " + err.Error() })
		return
	}
	defer f.Close()

	// 断点续传：init 返回的已传分片直接计入进度。
	uploaded := make(map[int]bool)
	for _, idx := range init.UploadedChunks {
		uploaded[idx] = true
		if idx+1 < total {
			a.mutate(func() {
				end := int64(idx+1) * chunkSize
				if end > t.size {
					end = t.size
				}
				t.done = end
			})
		}
	}

	// 简单并发：2 个 worker 从任务号通道领分片。
	const workers = 2
	idxCh := make(chan int)
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var firstErr error // 第一个遇到的错误
	setErr := func(e error) {
		errMu.Lock()
		if firstErr == nil {
			firstErr = e
		}
		errMu.Unlock()
	}
	getErr := func() error {
		errMu.Lock()
		defer errMu.Unlock()
		return firstErr
	}
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for idx := range idxCh {
				if t.cancel.Load() || getErr() != nil {
					continue // 排空通道
				}
				off := int64(idx) * chunkSize
				end := off + chunkSize
				if end > t.size {
					end = t.size
				}
				buf := make([]byte, end-off)
				if _, e := f.ReadAt(buf, off); e != nil && e != io.EOF {
					setErr(fmt.Errorf("读分片 %d: %w", idx, e))
					continue
				}
				sum := md5.Sum(buf)
				hash := hex.EncodeToString(sum[:])
				var lastE error
				for attempt := 1; attempt <= 3; attempt++ {
					if t.cancel.Load() {
						break
					}
					if e := a.client.UploadChunkFile(ctx, init.UploadID, idx, hash, buf); e == nil {
						lastE = nil
						break
					} else {
						lastE = e
						time.Sleep(time.Duration(attempt) * time.Second)
					}
				}
				if lastE != nil {
					setErr(fmt.Errorf("分片 %d: %s", idx, friendlyError(lastE)))
					continue
				}
				a.mutate(func() { t.done += int64(len(buf)) })
			}
		}()
	}
	for i := 0; i < total; i++ {
		if uploaded[i] {
			continue
		}
		idxCh <- i
	}
	close(idxCh)
	wg.Wait()

	// 取消：通知服务端清理已传分片。
	if t.cancel.Load() {
		a.cleanupUpload(init.UploadID)
		a.mutate(func() { t.state = 3 })
		return
	}
	if e := getErr(); e != nil {
		// 失败也清理服务端半成品，避免垃圾分片占配额
		a.cleanupUpload(init.UploadID)
		a.mutate(func() {
			t.state = 2
			t.err = e.Error()
		})
		return
	}

	file, err := a.client.CompleteChunkUpload(ctx, init.UploadID)
	if err != nil {
		a.mutate(func() {
			t.state = 2
			t.err = friendlyError(err)
		})
		return
	}
	a.mutate(func() {
		t.done = t.size
		t.state = 1
	})
	log.Printf("[files] 上传完成 %q → 文件 #%d", t.name, file.ID)
	if a.filesOpen {
		a.filesReloadList()
		a.filesReloadUsage()
	}
}

// cleanupUpload 尽力清理服务端半成品分片。
func (a *app) cleanupUpload(uploadID string) {
	if uploadID == "" || a.client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	_ = a.client.CancelChunkUpload(ctx, uploadID)
}

// cancelUploadTask 取消一个上传任务（分片间检查，已在传的分片作废）。
func (a *app) cancelUploadTask(taskID int) {
	a.mutate(func() {
		for _, t := range a.uploadTasks {
			if t.id == taskID && t.state == 0 {
				t.cancel.Store(true)
			}
		}
	})
}

// dismissUploadTask 从浮层移除一个已结束的任务。
func (a *app) dismissUploadTask(taskID int) {
	a.mutate(func() {
		out := a.uploadTasks[:0]
		for _, t := range a.uploadTasks {
			if t.id == taskID {
				continue
			}
			out = append(out, t)
		}
		a.uploadTasks = out
	})
}

// —— 下载 ——

// downloadFile 弹保存框 → 流式下载到本地。Dialog.Save 必须在非 UI 线程。
func (a *app) downloadFile(f qapi.FileItem) {
	if a.client == nil {
		return
	}
	go func() {
		dest, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Title:       "保存文件",
			DefaultPath: f.Name,
			Parent:      a.win,
		})
		if err != nil || dest == "" {
			return
		}
		ctx, cancel := context.WithTimeout(a.ctx, 30*time.Minute)
		defer cancel()
		if _, err := a.client.DownloadFileTo(ctx, f.ID, dest); err != nil {
			a.setNotice("下载失败：" + friendlyError(err))
			return
		}
		a.setNotice("已保存到 " + dest)
	}()
}

// —— 展示辅助 ——

// humanSize 字节 → 人类可读（对齐 qim-client formatFileSize）。
func humanSize(size int64) string {
	switch {
	case size >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(size)/(1<<30))
	case size >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(size)/(1<<20))
	case size >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(size)/(1<<10))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// fileKindLabel MIME → 类型标签（对齐 qim-client getFileTypeLabel 的主干分支）。
func fileKindLabel(mime string) string {
	m := strings.ToLower(mime)
	switch {
	case m == "":
		return "文件"
	case strings.HasPrefix(m, "image/"):
		return "图片"
	case strings.HasPrefix(m, "video/"):
		return "视频"
	case strings.HasPrefix(m, "audio/"):
		return "音频"
	case strings.Contains(m, "pdf"):
		return "PDF"
	case strings.Contains(m, "word"), strings.Contains(m, "document"):
		return "文档"
	case strings.Contains(m, "excel"), strings.Contains(m, "sheet"):
		return "表格"
	case strings.Contains(m, "powerpoint"), strings.Contains(m, "presentation"):
		return "演示"
	case strings.HasPrefix(m, "text/"):
		return "文本"
	case strings.Contains(m, "zip"), strings.Contains(m, "rar"), strings.Contains(m, "7z"):
		return "压缩包"
	default:
		return "文件"
	}
}

// fileKindColor MIME → 图标颜色（对齐 qim-client getFileIconColor 的关键分支）。
func fileKindColor(mime string) string {
	m := strings.ToLower(mime)
	switch {
	case strings.HasPrefix(m, "image/"):
		return "#22c55e" // success
	case strings.HasPrefix(m, "video/"):
		return "#ef4444" // error
	case strings.HasPrefix(m, "audio/"):
		return "#f59e0b" // warning
	case strings.Contains(m, "pdf"):
		return "#e74c3c"
	case strings.Contains(m, "word"), strings.Contains(m, "document"):
		return "#2b579a"
	case strings.Contains(m, "excel"), strings.Contains(m, "sheet"):
		return "#217346"
	case strings.Contains(m, "powerpoint"), strings.Contains(m, "presentation"):
		return "#d24726"
	case strings.HasPrefix(m, "text/"):
		return "#3b82f6" // primary
	case strings.Contains(m, "zip"), strings.Contains(m, "rar"), strings.Contains(m, "7z"):
		return "#f39c12"
	default:
		return "#6b7280" // gray-500
	}
}
