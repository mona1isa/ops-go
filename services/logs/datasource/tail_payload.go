package datasource

import (
	_ "embed"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

//go:embed assets/tail.sh
var tailScript string

// TailScript 返回内置的增量拉取脚本内容
func TailScript() string {
	return tailScript
}

// TailResult 一次远程拉取的结果
type TailResult struct {
	Inode    int64
	Pos      int64 // 下次读取的起始偏移量
	Size     int64
	Dropped  int64  // 因单轮上限被跳过的字节数
	FilePath string // 实际读取的文件（通配符场景下为选中的那个）
	Lines    []string
	HalfLine bool // 末行不完整（已剔除，下轮重读）
}

const (
	metaPrefix = "OPS_META "
	errPrefix  = "OPS_ERR"
)

// ParseTailOutput 解析 tail.sh 的输出。
// 脚本输出 = 元信息行 + 原始日志字节；stderr 噪声可能混在前面，因此按前缀定位元信息行。
func ParseTailOutput(output string) (*TailResult, error) {
	if strings.TrimSpace(output) == "" {
		return nil, errors.New("远程采集无输出")
	}

	idx := strings.Index(output, metaPrefix)
	if idx < 0 {
		errIdx := strings.Index(output, errPrefix)
		if errIdx >= 0 {
			line := output[errIdx:]
			if end := strings.IndexByte(line, '\n'); end > 0 {
				line = line[:end]
			}
			return nil, fmt.Errorf("远程采集失败: %s", strings.TrimSpace(strings.TrimPrefix(line, errPrefix)))
		}
		return nil, errors.New("远程采集输出缺少元信息行")
	}

	metaLine := output[idx+len(metaPrefix):]
	if end := strings.IndexByte(metaLine, '\n'); end >= 0 {
		metaLine = metaLine[:end]
	}
	contentStart := idx + len(metaPrefix) + len(metaLine) + 1
	content := ""
	if contentStart <= len(output) {
		content = output[contentStart:]
	}

	result := &TailResult{}
	if err := parseMeta(metaLine, result); err != nil {
		return nil, err
	}

	// 末行不完整时不投递，回退偏移量等下一轮补齐，避免把半行日志写进存储
	if content != "" && !strings.HasSuffix(content, "\n") {
		if cut := strings.LastIndexByte(content, '\n'); cut >= 0 {
			trailing := content[cut+1:]
			result.Pos -= int64(len(trailing))
			result.HalfLine = true
			content = content[:cut+1]
		} else {
			// 整个片段都没有换行：可能是超长单行，也可能还没写完，一律等下一轮
			result.Pos -= int64(len(content))
			result.HalfLine = true
			content = ""
		}
		if result.Pos < 0 {
			result.Pos = 0
		}
	}

	for _, line := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
		if line == "" {
			continue
		}
		result.Lines = append(result.Lines, line)
	}
	return result, nil
}

// parseMeta 解析 "inode=1 pos=2 size=3 dropped=4 file=/path" 形式的元信息。
// file 段固定在末尾，取值保留剩余全部内容，兼容带空格的路径。
func parseMeta(line string, result *TailResult) error {
	if idx := strings.Index(line, "file="); idx >= 0 {
		result.FilePath = strings.TrimSpace(line[idx+len("file="):])
		line = line[:idx]
	}
	for _, field := range strings.Fields(line) {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 {
			continue
		}
		value, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}
		switch parts[0] {
		case "inode":
			result.Inode = value
		case "pos":
			result.Pos = value
		case "size":
			result.Size = value
		case "dropped":
			result.Dropped = value
		}
	}
	if result.Pos < 0 {
		return errors.New("采集偏移量非法")
	}
	return nil
}
