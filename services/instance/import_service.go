package instance

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	"github.com/zhany/ops-go/controllers/instance/api"
	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
)

const (
	// ImportMaxRows 单次导入的最大数据行数
	ImportMaxRows = 1000
	// ImportMaxFileSize 导入文件大小上限
	ImportMaxFileSize = 5 << 20

	importSheetData  = "主机导入"
	importSheetGuide = "填写说明"
)

// importHeaders 模板表头，顺序即解析顺序
var importHeaders = []string{"主机名称*", "主机IP*", "CPU核数", "内存(GB)", "磁盘(GB)", "系统类型", "状态", "分组", "登录凭证", "主机描述"}

// importRow 一行待导入的主机数据
type importRow struct {
	RowNum int
	Name   string
	Ip     string
	Cpu    string
	MemGb  string
	DiskGb string
	Os     string
	Status string
	Group  string
	Keys   string
	Remark string
}

type InstanceImportService struct{}

// DownloadTemplate 生成并返回导入模板（xlsx）
func (s *InstanceImportService) DownloadTemplate() (api.ImportTemplateResponse, error) {
	content, err := buildTemplate()
	if err != nil {
		log.Println("生成主机导入模板失败：", err)
		return api.ImportTemplateResponse{}, errors.New("生成导入模板失败")
	}
	return api.ImportTemplateResponse{
		FileName: "主机批量导入模板.xlsx",
		Content:  base64.StdEncoding.EncodeToString(content),
	}, nil
}

// buildTemplate 构建带表头样式与填写说明的 xlsx 模板
func buildTemplate() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	if err := f.SetSheetName("Sheet1", importSheetData); err != nil {
		return nil, err
	}
	if _, err := f.NewSheet(importSheetGuide); err != nil {
		return nil, err
	}

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"409EFF"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return nil, err
	}

	for i, header := range importHeaders {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return nil, err
		}
		if err := f.SetCellValue(importSheetData, cell, header); err != nil {
			return nil, err
		}
	}
	if err := f.SetRowStyle(importSheetData, 1, 1, headerStyle); err != nil {
		return nil, err
	}
	if err := f.SetPanes(importSheetData, &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	}); err != nil {
		return nil, err
	}
	widths := []float64{22, 18, 10, 10, 10, 12, 8, 16, 20, 30}
	for i, width := range widths {
		col, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return nil, err
		}
		if err := f.SetColWidth(importSheetData, col, col, width); err != nil {
			return nil, err
		}
	}

	guide := [][]string{
		{"列名", "是否必填", "说明"},
		{"主机名称", "必填", "主机唯一名称，最长 32 个字符，重名行会被跳过"},
		{"主机IP", "必填", "IPv4 地址，如 192.168.1.10"},
		{"CPU核数", "选填", "正整数，留空默认 1"},
		{"内存(GB)", "选填", "数字，支持小数，留空默认 1GB，入库后按 MB 存储"},
		{"磁盘(GB)", "选填", "正整数，留空默认 10GB"},
		{"系统类型", "选填", "Linux / Windows / MacOS（不区分大小写），留空默认 Linux"},
		{"状态", "选填", "启用 / 禁用，留空默认启用"},
		{"分组", "选填", "主机分组名称，需与「分组管理」中已有分组一致，多个用英文分号 ; 分隔"},
		{"登录凭证", "选填", "凭证名称，需与「凭证管理」中已有凭证一致，多个用英文分号 ; 分隔"},
		{"主机描述", "选填", "备注信息，最长 500 个字符"},
		{},
		{"其他说明：", "", ""},
		{"1. 请勿修改「主机导入」工作表第一行表头，程序按表头顺序解析。", "", ""},
		{"2. 支持 .xlsx 与 .csv 两种格式，单次最多导入 1000 行，文件不超过 5MB。", "", ""},
		{"3. 导入失败的行会在结果中列出行号与原因，其余合法行照常入库。", "", ""},
	}
	guideStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"909399"}},
	})
	if err != nil {
		return nil, err
	}
	for r, row := range guide {
		for c, value := range row {
			cell, err := excelize.CoordinatesToCellName(c+1, r+1)
			if err != nil {
				return nil, err
			}
			if err := f.SetCellValue(importSheetGuide, cell, value); err != nil {
				return nil, err
			}
		}
	}
	if err := f.SetRowStyle(importSheetGuide, 1, 1, guideStyle); err != nil {
		return nil, err
	}
	for i, width := range []float64{46, 12, 60} {
		col, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return nil, err
		}
		if err := f.SetColWidth(importSheetGuide, col, col, width); err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Import 解析文件并批量导入主机，逐行校验，失败行跳过并返回明细
func (s *InstanceImportService) Import(fileName string, reader io.Reader, userId string) (api.ImportResult, error) {
	rows, err := ParseImportFile(fileName, reader)
	if err != nil {
		return api.ImportResult{}, err
	}
	if len(rows) == 0 {
		return api.ImportResult{}, errors.New("文件中没有可导入的主机数据")
	}

	ctx, err := newImportContext(rows, userId)
	if err != nil {
		return api.ImportResult{}, err
	}

	result := api.ImportResult{Total: len(rows)}
	for _, row := range rows {
		reason := ctx.importRow(row)
		if reason != "" {
			result.Failed++
			result.Errors = append(result.Errors, api.ImportRowError{Row: row.RowNum, Name: row.Name, Reason: reason})
			continue
		}
		result.Success++
	}
	return result, nil
}

// ParseImportFile 按扩展名解析 xlsx / csv，返回待导入的数据行
func ParseImportFile(fileName string, reader io.Reader) ([]importRow, error) {
	ext := strings.ToLower(filepath.Ext(fileName))
	var raw [][]string
	switch ext {
	case ".xlsx":
		f, err := excelize.OpenReader(reader)
		if err != nil {
			return nil, errors.New("文件解析失败，请确认为标准的 xlsx 文件")
		}
		defer f.Close()
		sheet := f.GetSheetName(0)
		if sheet == "" {
			return nil, errors.New("文件中没有工作表")
		}
		sheets, err := f.GetRows(sheet)
		if err != nil {
			return nil, errors.New("文件解析失败")
		}
		raw = sheets
	case ".csv":
		content, err := io.ReadAll(reader)
		if err != nil {
			return nil, errors.New("文件读取失败")
		}
		content = bytes.TrimPrefix(content, []byte("\ufeff"))
		r := csv.NewReader(bytes.NewReader(content))
		r.FieldsPerRecord = -1
		records, err := r.ReadAll()
		if err != nil {
			return nil, errors.New("CSV 解析失败，请检查分隔符与引号是否规范")
		}
		raw = records
	default:
		return nil, errors.New("仅支持 .xlsx 与 .csv 文件，请在 Excel/WPS 中另存为这两种格式")
	}

	if len(raw) == 0 {
		return nil, errors.New("文件内容为空")
	}

	header := normalizeHeaderRow(raw[0])
	if len(header) < 2 || !strings.HasPrefix(header[0], "主机名称") || !strings.HasPrefix(header[1], "主机IP") {
		return nil, errors.New("表头与模板不一致，请下载最新模板后重新填写")
	}

	var rows []importRow
	for i, line := range raw[1:] {
		rowNum := i + 2
		cells := normalizeRow(line)
		if isEmptyRow(cells) {
			continue
		}
		pick := func(idx int) string {
			if idx < len(cells) {
				return cells[idx]
			}
			return ""
		}
		rows = append(rows, importRow{
			RowNum: rowNum,
			Name:   pick(0),
			Ip:     pick(1),
			Cpu:    pick(2),
			MemGb:  pick(3),
			DiskGb: pick(4),
			Os:     pick(5),
			Status: pick(6),
			Group:  pick(7),
			Keys:   pick(8),
			Remark: pick(9),
		})
	}
	if len(rows) > ImportMaxRows {
		return nil, fmt.Errorf("单次最多导入 %d 行，当前 %d 行，请拆分后重试", ImportMaxRows, len(rows))
	}
	return rows, nil
}

func normalizeHeaderRow(line []string) []string {
	out := make([]string, len(line))
	for i, cell := range line {
		out[i] = strings.TrimSpace(cell)
	}
	return out
}

func normalizeRow(line []string) []string {
	out := make([]string, len(line))
	for i, cell := range line {
		out[i] = strings.TrimSpace(strings.Trim(cell, "'"))
	}
	return out
}

func isEmptyRow(cells []string) bool {
	for _, cell := range cells {
		if cell != "" {
			return false
		}
	}
	return true
}

// importContext 缓存本次导入需要用到的已存在数据，避免逐行查库
type importContext struct {
	userId       string
	existingName map[string]bool
	usedName     map[string]bool
	usedIp       map[string]bool
	groups       map[string]int
	keys         map[string]int
}

func newImportContext(rows []importRow, userId string) (*importContext, error) {
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Name != "" {
			names = append(names, row.Name)
		}
	}

	ctx := &importContext{
		userId:       userId,
		existingName: map[string]bool{},
		usedName:     map[string]bool{},
		usedIp:       map[string]bool{},
		groups:       map[string]int{},
		keys:         map[string]int{},
	}

	if len(names) > 0 {
		var existing []string
		if err := models.DB.Model(&models.OpsInstance{}).
			Where("name IN ?", names).
			Pluck("name", &existing).Error; err != nil {
			log.Println("查询已存在主机失败：", err)
			return nil, errors.New("查询主机数据失败")
		}
		for _, name := range existing {
			ctx.existingName[name] = true
		}
	}

	var groups []models.OpsGroup
	if err := models.DB.Where("del_flag = ?", "0").Find(&groups).Error; err != nil {
		log.Println("查询分组失败：", err)
		return nil, errors.New("查询分组数据失败")
	}
	for _, group := range groups {
		ctx.groups[group.Name] = group.ID
	}

	var keys []models.OpsKey
	if err := models.DB.Where("del_flag = ?", "0").Find(&keys).Error; err != nil {
		log.Println("查询凭证失败：", err)
		return nil, errors.New("查询凭证数据失败")
	}
	for _, key := range keys {
		ctx.keys[key.Name] = key.ID
	}

	return ctx, nil
}

// importRow 校验并写入单行，返回空字符串表示成功，否则返回失败原因
func (c *importContext) importRow(row importRow) string {
	instance, groupIds, keyIds, reason := c.validateRow(row)
	if reason != "" {
		return reason
	}

	err := models.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&instance).Error; err != nil {
			return err
		}
		for _, groupId := range groupIds {
			if err := tx.Create(&models.OpsInstanceGroup{InstanceId: instance.ID, GroupId: groupId}).Error; err != nil {
				return err
			}
		}
		for _, keyId := range keyIds {
			if err := tx.Create(&models.OpsInstanceKey{InstanceId: instance.ID, KeyId: keyId}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("导入主机失败 name=%s: %v", row.Name, err)
		return "入库失败，请重试"
	}

	c.usedName[row.Name] = true
	c.usedIp[row.Ip] = true
	return ""
}

// validateRow 校验单行数据并组装待写入的主机记录，返回空原因表示校验通过
func (c *importContext) validateRow(row importRow) (models.OpsInstance, []int, []int, string) {
	var instance models.OpsInstance
	if row.Name == "" {
		return instance, nil, nil, "主机名称不能为空"
	}
	if len([]rune(row.Name)) > 32 {
		return instance, nil, nil, "主机名称超过 32 个字符"
	}
	if c.existingName[row.Name] {
		return instance, nil, nil, "主机名称已存在"
	}
	if c.usedName[row.Name] {
		return instance, nil, nil, "文件内主机名称重复"
	}
	if row.Ip == "" {
		return instance, nil, nil, "主机IP不能为空"
	}
	if ip := net.ParseIP(row.Ip); ip == nil || ip.To4() == nil {
		return instance, nil, nil, "主机IP不是合法的 IPv4 地址"
	}
	if c.usedIp[row.Ip] {
		return instance, nil, nil, "文件内主机IP重复"
	}
	if len([]rune(row.Remark)) > 500 {
		return instance, nil, nil, "主机描述超过 500 个字符"
	}

	cpu := 1
	if row.Cpu != "" {
		value, err := strconv.Atoi(row.Cpu)
		if err != nil || value < 0 {
			return instance, nil, nil, "CPU核数需填写非负整数"
		}
		cpu = value
	}

	memGb := 1.0
	if row.MemGb != "" {
		value, err := strconv.ParseFloat(row.MemGb, 64)
		if err != nil || value < 0 {
			return instance, nil, nil, "内存(GB)需填写数字"
		}
		memGb = value
	}
	memMb := int(math.Round(memGb * 1024))

	diskGb := 10
	if row.DiskGb != "" {
		value, err := strconv.Atoi(row.DiskGb)
		if err != nil || value < 0 {
			return instance, nil, nil, "磁盘(GB)需填写非负整数"
		}
		diskGb = value
	}

	os := "Linux"
	if row.Os != "" {
		matched := false
		for _, candidate := range []string{"Linux", "Windows", "MacOS"} {
			if strings.EqualFold(row.Os, candidate) {
				os = candidate
				matched = true
				break
			}
		}
		if !matched {
			return instance, nil, nil, "系统类型仅支持 Linux / Windows / MacOS"
		}
	}

	status := "1"
	switch row.Status {
	case "", "启用", "正常", "1", "在线":
		status = "1"
	case "禁用", "停用", "0", "离线":
		status = "0"
	default:
		return instance, nil, nil, "状态仅支持 启用 / 禁用"
	}

	groupIds, reason := c.resolveIds(row.Group, c.groups, "分组")
	if reason != "" {
		return instance, nil, nil, reason
	}
	keyIds, reason := c.resolveIds(row.Keys, c.keys, "登录凭证")
	if reason != "" {
		return instance, nil, nil, reason
	}

	instance.Name = row.Name
	instance.Cpu = cpu
	instance.MemMb = memMb
	instance.DiskGb = diskGb
	instance.Ip = row.Ip
	instance.Os = os
	instance.Status = status
	instance.Remark = row.Remark
	instance.CreateBy = c.userId
	instance.UpdateBy = c.userId

	return instance, groupIds, keyIds, ""
}

// resolveIds 将「分组/登录凭证」列的名称解析为主键，任一分组名称不存在即返回错误
func (c *importContext) resolveIds(column string, dict map[string]int, label string) ([]int, string) {
	if column == "" {
		return nil, ""
	}
	replacer := strings.NewReplacer("；", ";", "，", ";", ",", ";", "、", ";", "|", ";", "/", ";")
	var ids []int
	for _, name := range strings.Split(replacer.Replace(column), ";") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		id, ok := dict[name]
		if !ok {
			return nil, fmt.Sprintf("%s「%s」不存在", label, name)
		}
		ids = append(ids, id)
	}
	return ids, ""
}
