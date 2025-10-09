package utils

import (
	"github.com/google/uuid"
	"strings"
)

// GenUuid 生成UUID
func GenUuid() string {
	// 生成一个随机的UUID（版本4）
	id := uuid.New()
	return strings.ReplaceAll(id.String(), "-", "")
}
