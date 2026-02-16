package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SessionRecorder 会话录制器
type SessionRecorder struct {
	sessionID    string
	filePath     string
	file         *os.File
	startTime    time.Time
	lastWriteTime time.Time
	width        int
	height       int
	mu           sync.Mutex
}

// AsciinemaHeader asciinema v2 格式头部
type AsciinemaHeader struct {
	Version   int    `json:"version"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Timestamp int64  `json:"timestamp"`
	Title     string `json:"title,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
}

// AsciinemaFrame asciinema v2 格式帧
type AsciinemaFrame struct {
	Time    float64 `json:"time"`    // 相对开始时间的秒数
	EventType string `json:"-"`      // "o" (output) 或 "i" (input)
	Data    string  `json:"-"`       // 数据内容
}

// NewSessionRecorder 创建会话录制器
func NewSessionRecorder(sessionID string, width, height int, storagePath string) (*SessionRecorder, error) {
	// 确保存储目录存在
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("创建存储目录失败: %v", err)
	}

	// 生成文件名：按日期分组
	now := time.Now()
	dateDir := now.Format("2006-01-02")
	sessionDir := filepath.Join(storagePath, dateDir)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return nil, fmt.Errorf("创建会话目录失败: %v", err)
	}

	// 文件名：sessionId.cast
	fileName := fmt.Sprintf("%s.cast", sessionID)
	filePath := filepath.Join(sessionDir, fileName)

	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("创建录像文件失败: %v", err)
	}

	recorder := &SessionRecorder{
		sessionID:    sessionID,
		filePath:     filePath,
		file:         file,
		startTime:    now,
		lastWriteTime: now,
		width:        width,
		height:       height,
	}

	// 写入头部
	if err := recorder.writeHeader(); err != nil {
		file.Close()
		os.Remove(filePath)
		return nil, fmt.Errorf("写入录像头部失败: %v", err)
	}

	return recorder, nil
}

// writeHeader 写入 asciinema 文件头部
func (r *SessionRecorder) writeHeader() error {
	header := AsciinemaHeader{
		Version:   2,
		Width:     r.width,
		Height:    r.height,
		Timestamp: r.startTime.Unix(),
		Title:     fmt.Sprintf("Session %s", r.sessionID),
		Env: map[string]string{
			"TERM": "xterm-256color",
			"SHELL": "/bin/bash",
		},
	}

	data, err := json.Marshal(header)
	if err != nil {
		return err
	}

	_, err = r.file.WriteString(string(data) + "\n")
	return err
}

// RecordOutput 记录输出数据
func (r *SessionRecorder) RecordOutput(data string) error {
	return r.recordFrame("o", data)
}

// RecordInput 记录输入数据
func (r *SessionRecorder) RecordInput(data string) error {
	return r.recordFrame("i", data)
}

// recordFrame 记录帧数据
func (r *SessionRecorder) recordFrame(eventType, data string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file == nil {
		return fmt.Errorf("录像文件已关闭")
	}

	now := time.Now()
	elapsed := now.Sub(r.startTime).Seconds()

	// 使用 JSON 编码来正确转义数据（符合 JSON 标准）
	// JSON 标准支持 \uXXXX 转义，不支持 \xXX 转义
	eventTypeJSON, _ := json.Marshal(eventType)
	dataJSON, _ := json.Marshal(data)

	// asciinema v2 格式：[time, event_type, data]
	frame := fmt.Sprintf("[%.6f, %s, %s]\n", elapsed, string(eventTypeJSON), string(dataJSON))

	_, err := r.file.WriteString(frame)
	if err != nil {
		return err
	}

	r.lastWriteTime = now
	return nil
}

// Resize 调整终端大小
func (r *SessionRecorder) Resize(width, height int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.width = width
	r.height = height
}

// Close 关闭录制器
func (r *SessionRecorder) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file == nil {
		return nil
	}

	// 刷新并关闭文件
	if err := r.file.Sync(); err != nil {
		r.file.Close()
		return err
	}

	err := r.file.Close()
	r.file = nil
	return err
}

// GetFilePath 获取录像文件路径
func (r *SessionRecorder) GetFilePath() string {
	return r.filePath
}

// GetFileSize 获取录像文件大小
func (r *SessionRecorder) GetFileSize() (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file == nil {
		// 文件已关闭，从文件系统获取大小
		info, err := os.Stat(r.filePath)
		if err != nil {
			return 0, err
		}
		return info.Size(), nil
	}

	// 文件还打开，刷新后获取大小
	if err := r.file.Sync(); err != nil {
		return 0, err
	}
	
	info, err := r.file.Stat()
	if err != nil {
		return 0, err
	}
	
	return info.Size(), nil
}

// GetDuration 获取会话时长（秒）
func (r *SessionRecorder) GetDuration() int {
	return int(time.Since(r.startTime).Seconds())
}
