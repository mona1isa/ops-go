package instance

import (
	"fmt"
	"github.com/zhany/ops-go/models"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// RecordingCleanupService 审计录像自动清理服务
type RecordingCleanupService struct {
	// 录像保留天数
	RetentionDays int
	// 清理检查间隔（默认每天执行一次）
	Interval time.Duration
}

// NewRecordingCleanupService 创建录像清理服务
func NewRecordingCleanupService() *RecordingCleanupService {
	retentionDays := 90 // 默认保留90天
	if val := os.Getenv("RECORDING_RETENTION_DAYS"); val != "" {
		if days, err := strconv.Atoi(val); err == nil && days > 0 {
			retentionDays = days
		}
	}

	return &RecordingCleanupService{
		RetentionDays: retentionDays,
		Interval:      24 * time.Hour,
	}
}

// Start 启动录像清理定时任务
func (s *RecordingCleanupService) Start() {
	log.Printf("录像清理服务已启动，保留天数: %d 天，检查间隔: %v", s.RetentionDays, s.Interval)

	// 首次启动延迟1分钟执行，避免与其他初始化操作冲突
	time.Sleep(1 * time.Minute)
	s.cleanExpiredRecordings()

	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanExpiredRecordings()
	}
}

// cleanExpiredRecordings 清理过期的录像文件和数据库记录
func (s *RecordingCleanupService) cleanExpiredRecordings() {
	cutoffTime := time.Now().AddDate(0, 0, -s.RetentionDays)
	log.Printf("开始清理过期录像，清理时间截止点: %s", cutoffTime.Format("2006-01-02 15:04:05"))

	// 查询过期的已结束会话记录（状态为已完成或异常中断，且开始时间早于截止时间）
	var expiredRecords []models.OpsSessionRecord
	if err := models.DB.Where("status IN ? AND start_time < ?",
		[]int8{models.SessionStatusCompleted, models.SessionStatusAborted}, cutoffTime).
		Find(&expiredRecords).Error; err != nil {
		log.Printf("查询过期录像记录失败: %v", err)
		return
	}

	if len(expiredRecords) == 0 {
		log.Println("没有需要清理的过期录像")
		return
	}

	log.Printf("找到 %d 条过期录像记录，开始清理", len(expiredRecords))

	deletedFiles := 0
	deletedRecords := 0
	failedFiles := 0

	for _, record := range expiredRecords {
		// 删除录像文件
		if record.RecordingFile != "" {
			if err := s.deleteRecordingFile(record.RecordingFile); err != nil {
				log.Printf("删除录像文件失败 [%s]: %v", record.RecordingFile, err)
				failedFiles++
			} else {
				deletedFiles++
			}
		}

		// 删除数据库记录
		if err := models.DB.Delete(&models.OpsSessionRecord{}, record.ID).Error; err != nil {
			log.Printf("删除录像数据库记录失败 [ID=%d]: %v", record.ID, err)
		} else {
			deletedRecords++
		}
	}

	// 清理空目录
	s.cleanEmptyDirs()

	log.Printf("录像清理完成：删除文件 %d 个，删除记录 %d 条，文件删除失败 %d 个", deletedFiles, deletedRecords, failedFiles)
}

// deleteRecordingFile 删除单个录像文件
func (s *RecordingCleanupService) deleteRecordingFile(filePath string) error {
	// 安全检查：确保文件路径在 recordings 目录下，防止路径遍历攻击
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("获取绝对路径失败: %w", err)
	}

	recordingsDir, err := filepath.Abs("recordings")
	if err != nil {
		return fmt.Errorf("获取录像目录绝对路径失败: %w", err)
	}

	if !filepath.HasPrefix(absPath, recordingsDir+string(filepath.Separator)) {
		return fmt.Errorf("录像文件路径不在 recordings 目录下，拒绝删除: %s", absPath)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		// 文件不存在，无需删除
		return nil
	}

	return os.Remove(absPath)
}

// cleanEmptyDirs 清理 recordings 目录下的空日期目录
func (s *RecordingCleanupService) cleanEmptyDirs() {
	recordingsDir := "recordings"
	entries, err := os.ReadDir(recordingsDir)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("读取 recordings 目录失败: %v", err)
		}
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirPath := filepath.Join(recordingsDir, entry.Name())
		// 尝试删除空目录，如果目录非空则不会删除
		err := os.Remove(dirPath)
		if err == nil {
			log.Printf("清理空目录: %s", dirPath)
		}
	}
}
