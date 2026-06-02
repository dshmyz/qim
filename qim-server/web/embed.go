package web

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed webroot
var staticFS embed.FS

var devMode bool

func init() {
	devMode = os.Getenv("DEV_MODE") == "true"
}

func serveSPA(prefix string, useParam bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var path string
		if useParam {
			path = c.Param("filepath")
		} else {
			path = c.Request.URL.Path
			if strings.HasPrefix(path, "/") {
				path = path[1:]
			}
		}

		if path == "" || path == "/" {
			path = "index.html"
		}

		fullPath := filepath.Join(prefix, path)

		var file fs.File
		var err error

		if devMode {
			file, err = os.Open(filepath.Join("web", fullPath))
		} else {
			file, err = staticFS.Open(fullPath)
		}

		if err != nil {
			serveIndex(c, prefix)
			return
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			serveIndex(c, prefix)
			return
		}

		if stat.IsDir() {
			serveIndex(c, fullPath)
			return
		}

		content, err := readFile(devMode, fullPath)
		if err != nil {
			serveIndex(c, prefix)
			return
		}

		cacheControl := "public, max-age=3600"
		if devMode {
			cacheControl = "no-cache"
		}
		c.Header("Cache-Control", cacheControl)
		c.Data(http.StatusOK, getContentType(fullPath), content)
	}
}

func serveIndex(c *gin.Context, prefix string) {
	cacheControl := "no-cache"
	if !devMode {
		cacheControl = "public, max-age=300"
	}
	for _, name := range []string{"index.html", "landing.html"} {
		content, err := readFile(devMode, filepath.Join(prefix, name))
		if err == nil {
			c.Header("Cache-Control", cacheControl)
			c.Data(http.StatusOK, "text/html; charset=utf-8", content)
			return
		}
	}
	c.AbortWithStatus(http.StatusNotFound)
}

func readFile(fromDisk bool, path string) ([]byte, error) {
	if fromDisk {
		return os.ReadFile(filepath.Join("web", path))
	}
	return staticFS.ReadFile(path)
}

func getContentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(path, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(path, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".ico"):
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}

func ServeLanding() gin.HandlerFunc {
	return serveSPA("webroot/landing", false)
}

func ServeAdmin() gin.HandlerFunc {
	return serveSPA("webroot/admin", true)
}

var _ fs.FS = staticFS
