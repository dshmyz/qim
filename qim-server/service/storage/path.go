package storage

import "strings"

func ParsePath(storagePath string) (kind, key string) {
	if strings.HasPrefix(storagePath, "/s3/") {
		return "s3", strings.TrimPrefix(storagePath, "/s3/")
	}
	if strings.HasPrefix(storagePath, "/") {
		return "local", strings.TrimPrefix(storagePath, "/")
	}
	return "local", storagePath
}

func BuildPath(kind, key string) string {
	if kind == "s3" {
		return "/s3/" + key
	}
	return "/" + key
}

func ToNewKey(storagePath string) string {
	_, key := ParsePath(storagePath)
	return key
}

func FromNewKey(kind, key string) string {
	return BuildPath(kind, key)
}
