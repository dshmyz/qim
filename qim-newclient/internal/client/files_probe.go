package client

// -probe-files：对真实 qim-server 验证 D 组文件网盘链路。
// 建文件夹 → 分片上传 1MB 随机文件（真实分片+MD5）→ 目录内列表回读 →
// 星标 → 星标视图可见 → 重命名 → 移动到根 → 流式下载逐字节比对 →
// 批量删除 → 级联删文件夹（清理现场）。

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"qim-newclient/internal/qapi"
)

func runFilesProbe() int {
	server := strings.TrimSpace(*flagServer)
	user := strings.TrimSpace(*flagUser)
	pass := *flagPass
	if pass == "" {
		pass = os.Getenv("QIM_PASS")
	}
	if server == "" || user == "" || pass == "" {
		fmt.Println("用法: -probe-files -server http://host:port -user <用户名> -pass <密码>")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	c := qapi.New(server)
	fmt.Printf("服务端: %s\n", c.BaseURL())

	if _, err := c.Login(ctx, user, pass); err != nil {
		fmt.Printf("[登录] ✗ %v\n", err)
		return 1
	}
	fmt.Println("[登录] ✓")

	// 1. 建探针文件夹
	folderName := fmt.Sprintf("探针文件箱-%d", time.Now().Unix())
	folder, err := c.CreateFolder(ctx, folderName, 0)
	if err != nil {
		fmt.Printf("[建目录] ✗ %v\n", err)
		return 1
	}
	fmt.Printf("[建目录] ✓ #%d %s\n", folder.ID, folder.Name)
	var cleanupIDs []int64
	defer func() {
		for _, id := range cleanupIDs {
			_ = c.DeleteFolder(context.Background(), id, true) // 尽力清理
		}
	}()

	// 2. 分片上传 1MB 随机文件
	data := make([]byte, 1<<20)
	if _, err := rand.Read(data); err != nil {
		fmt.Printf("[上传] ✗ 造随机数据失败: %v\n", err)
		return 1
	}
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("probe-files-%d.bin", time.Now().Unix()))
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		fmt.Printf("[上传] ✗ 写临时文件失败: %v\n", err)
		return 1
	}
	defer os.Remove(tmp)

	init, err := c.InitChunkUpload(ctx, filepath.Base(tmp), int64(len(data)), folder.ID, "")
	if err != nil {
		fmt.Printf("[上传] ✗ init: %v\n", err)
		return 1
	}
	fmt.Printf("[上传] init ✓ upload_id=%s chunk_size=%d total=%d\n", init.UploadID, init.ChunkSize, init.TotalChunks)
	for i := 0; i*int(init.ChunkSize) < len(data); i++ {
		start := i * int(init.ChunkSize)
		end := start + int(init.ChunkSize)
		if end > len(data) {
			end = len(data)
		}
		part := data[start:end]
		sum := md5.Sum(part)
		if err := c.UploadChunkFile(ctx, init.UploadID, i, hex.EncodeToString(sum[:]), part); err != nil {
			fmt.Printf("[上传] ✗ 分片 %d: %v\n", i, err)
			return 1
		}
	}
	uploaded, err := c.CompleteChunkUpload(ctx, init.UploadID)
	if err != nil {
		fmt.Printf("[上传] ✗ complete: %v\n", err)
		return 1
	}
	if uploaded.Size != int64(len(data)) {
		fmt.Printf("[上传] ✗ 大小不符: %d != %d\n", uploaded.Size, len(data))
		return 1
	}
	fmt.Printf("[上传] ✓ 文件 #%d %s (%s)\n", uploaded.ID, uploaded.Name, humanSize(uploaded.Size))

	// 3. 目录内列表回读
	list, err := c.ListFiles(ctx, qapi.FileListParams{FolderID: folder.ID, Page: 1, PageSize: 50})
	if err != nil {
		fmt.Printf("[列表] ✗ %v\n", err)
		return 1
	}
	if len(list.Files) != 1 || list.Files[0].ID != uploaded.ID {
		fmt.Printf("[列表] ✗ 目录内应有且仅有刚上传的文件: %d 条\n", len(list.Files))
		return 1
	}
	fmt.Printf("[列表] ✓ 目录内 1 条\n")

	// 4. 星标 → 全局星标视图可见
	if _, err := c.ToggleStar(ctx, uploaded.ID); err != nil {
		fmt.Printf("[星标] ✗ %v\n", err)
		return 1
	}
	starred, err := c.ListFiles(ctx, qapi.FileListParams{Starred: true, Page: 1, PageSize: 50})
	if err != nil {
		fmt.Printf("[星标] ✗ 星标列表: %v\n", err)
		return 1
	}
	found := false
	for _, f := range starred.Files {
		if f.ID == uploaded.ID {
			found = true
		}
	}
	if !found {
		fmt.Printf("[星标] ✗ 星标视图未见文件 #%d\n", uploaded.ID)
		return 1
	}
	fmt.Println("[星标] ✓ 星标视图可见")

	// 5. 重命名
	newName := fmt.Sprintf("探针重命名-%d.bin", time.Now().Unix())
	if err := c.UpdateFile(ctx, uploaded.ID, newName, nil); err != nil {
		fmt.Printf("[重命名] ✗ %v\n", err)
		return 1
	}
	fmt.Printf("[重命名] ✓ %s\n", newName)

	// 6. 移动到另一个文件夹（后端 batch move 只收真实文件夹，不支持根目录）
	dst, err := c.CreateFolder(ctx, fmt.Sprintf("探针目标-%d", time.Now().Unix()), 0)
	if err != nil {
		fmt.Printf("[移动] ✗ 建目标目录: %v\n", err)
		return 1
	}
	cleanupIDs = append(cleanupIDs, dst.ID)
	if err := c.BatchOperation(ctx, "move", []int64{uploaded.ID}, dst.ID); err != nil {
		fmt.Printf("[移动] ✗ %v\n", err)
		return 1
	}
	dstList, err := c.ListFiles(ctx, qapi.FileListParams{FolderID: dst.ID, Search: newName, Page: 1, PageSize: 50})
	if err != nil || len(dstList.Files) != 1 {
		fmt.Printf("[移动] ✗ 目标目录未命中: err=%v n=%d\n", err, len(dstList.Files))
		return 1
	}
	fmt.Printf("[移动] ✓ 已到目录 #%d\n", dst.ID)

	// 7. 流式下载逐字节比对
	dl := filepath.Join(os.TempDir(), fmt.Sprintf("probe-dl-%d.bin", time.Now().Unix()))
	defer os.Remove(dl)
	n, err := c.DownloadFileTo(ctx, uploaded.ID, dl)
	if err != nil {
		fmt.Printf("[下载] ✗ %v\n", err)
		return 1
	}
	got, err := os.ReadFile(dl)
	if err != nil || n != int64(len(data)) || !bytes.Equal(got, data) {
		fmt.Printf("[下载] ✗ 内容比对失败: n=%d err=%v equal=%v\n", n, err, err == nil && bytes.Equal(got, data))
		return 1
	}
	fmt.Printf("[下载] ✓ %s 逐字节一致\n", humanSize(n))

	// 8. 批量删除（清理现场；文件夹由 defer 级联删）
	if err := c.BatchOperation(ctx, "delete", []int64{uploaded.ID}, 0); err != nil {
		fmt.Printf("[清理] ✗ 批量删除: %v\n", err)
		return 1
	}
	after, err := c.ListFiles(ctx, qapi.FileListParams{FolderID: 0, Search: newName, Page: 1, PageSize: 50})
	if err != nil || len(after.Files) != 0 {
		fmt.Printf("[清理] ✗ 删除后仍可见: %d\n", len(after.Files))
		return 1
	}
	fmt.Println("[清理] ✓ 批量删除生效")

	// 9. 存储用量可达
	used, quota, err := c.StorageUsage(ctx)
	if err != nil || quota <= 0 {
		fmt.Printf("[用量] ✗ used=%d quota=%d err=%v\n", used, quota, err)
		return 1
	}
	fmt.Printf("[用量] ✓ %s / %s\n", humanSize(used), humanSize(quota))

	fmt.Println("=== 探针全过 ===")
	return 0
}
