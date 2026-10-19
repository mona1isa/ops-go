package instance

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// TestParseBuiltTemplate 校验模板可被自身解析逻辑识别（表头一致，无数据行）
func TestParseBuiltTemplate(t *testing.T) {
	content, err := buildTemplate()
	if err != nil {
		t.Fatalf("生成模板失败: %v", err)
	}
	rows, err := ParseImportFile("template.xlsx", bytes.NewReader(content))
	if err != nil {
		t.Fatalf("解析模板失败: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("模板不应含数据行，实际 %d 行", len(rows))
	}
}

func TestParseCsvRows(t *testing.T) {
	csv := "\ufeff" + strings.Join([]string{
		"主机名称,主机IP,CPU核数,内存(GB),磁盘(GB),系统类型,状态,分组,登录凭证,主机描述",
		"web-01,192.168.1.10,4,8,100,Linux,启用,生产,root密码;",
		"db-01,192.168.1.11,8,16,500,windows,禁用,,,",
		",,,,,,",
	}, "\n")

	rows, err := ParseImportFile("hosts.csv", strings.NewReader(csv))
	if err != nil {
		t.Fatalf("解析 CSV 失败: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("期望 2 行有效数据，实际 %d 行", len(rows))
	}
	if rows[0].Name != "web-01" || rows[0].Ip != "192.168.1.10" || rows[0].Keys != "root密码;" {
		t.Fatalf("第一行解析结果异常: %+v", rows[0])
	}
	if rows[0].RowNum != 2 || rows[1].RowNum != 3 {
		t.Fatalf("行号应从第 2 行开始: %d, %d", rows[0].RowNum, rows[1].RowNum)
	}
	if rows[1].Os != "windows" {
		t.Fatalf("系统类型应原样保留供后续归一化: %+v", rows[1])
	}
}

func TestParseRejectsBadHeader(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	if err := f.SetCellValue("Sheet1", "A1", "主机名"); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}

	_, err := ParseImportFile("hosts.xlsx", bytes.NewReader(buf.Bytes()))
	if err == nil || !strings.Contains(err.Error(), "表头") {
		t.Fatalf("表头不匹配时应报错，实际: %v", err)
	}
}

func TestParseRejectsUnknownExtension(t *testing.T) {
	if _, err := ParseImportFile("hosts.txt", strings.NewReader("a")); err == nil {
		t.Fatal("非 xlsx/csv 文件应报错")
	}
}

// newTestContext 构造不依赖数据库的导入上下文
func newTestContext() *importContext {
	return &importContext{
		userId:       "1",
		existingName: map[string]bool{"old-host": true},
		usedName:     map[string]bool{},
		usedIp:       map[string]bool{},
		groups:       map[string]int{"生产": 7},
		keys:         map[string]int{"root密码": 3},
	}
}

func TestValidateRowDefaults(t *testing.T) {
	ctx := newTestContext()
	instance, groupIds, keyIds, reason := ctx.validateRow(importRow{RowNum: 2, Name: "web-01", Ip: "10.0.0.1"})
	if reason != "" {
		t.Fatalf("最小填写应通过校验: %s", reason)
	}
	if instance.Cpu != 1 || instance.MemMb != 1024 || instance.DiskGb != 10 {
		t.Fatalf("默认规格不符合预期: %+v", instance)
	}
	if instance.Os != "Linux" || instance.Status != "1" {
		t.Fatalf("默认系统/状态不符合预期: %+v", instance)
	}
	if instance.InstanceId != "" || instance.ID != 0 {
		t.Fatalf("校验阶段不应生成主键")
	}
	if len(groupIds) != 0 || len(keyIds) != 0 {
		t.Fatalf("空分组/凭证应解析为空: %v %v", groupIds, keyIds)
	}
}

func TestValidateRowConvertsMemToMb(t *testing.T) {
	ctx := newTestContext()
	instance, _, _, reason := ctx.validateRow(importRow{
		RowNum: 2, Name: "web-02", Ip: "10.0.0.2", Cpu: "4", MemGb: "7.5", DiskGb: "200",
		Os: "windows", Status: "禁用", Group: "生产", Keys: "root密码;", Remark: "备注",
	})
	if reason != "" {
		t.Fatalf("完整填写应通过校验: %s", reason)
	}
	if instance.MemMb != 7680 {
		t.Fatalf("内存应按 GB 转 MB 入库，实际 %d", instance.MemMb)
	}
	if instance.Os != "Windows" || instance.Status != "0" {
		t.Fatalf("系统/状态归一化失败: %+v", instance)
	}
	if instance.Remark != "备注" || instance.CreateBy != "1" {
		t.Fatalf("备注与创建人应写入: %+v", instance)
	}
}

func TestValidateRowResolvesGroupAndKeys(t *testing.T) {
	ctx := newTestContext()
	_, groupIds, keyIds, reason := ctx.validateRow(importRow{
		RowNum: 2, Name: "web-03", Ip: "10.0.0.3", Group: "生产", Keys: "root密码；root密码",
	})
	if reason != "" {
		t.Fatalf("分组与凭证应可解析: %s", reason)
	}
	if len(groupIds) != 1 || groupIds[0] != 7 {
		t.Fatalf("分组解析结果异常: %v", groupIds)
	}
	if len(keyIds) != 2 || keyIds[0] != 3 {
		t.Fatalf("中文分号应作为分隔符: %v", keyIds)
	}
}

func TestValidateRowRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name string
		row  importRow
		want string
	}{
		{"缺少名称", importRow{Ip: "10.0.0.1"}, "主机名称不能为空"},
		{"名称过长", importRow{Name: strings.Repeat("a", 33), Ip: "10.0.0.1"}, "超过 32 个字符"},
		{"名称已存在", importRow{Name: "old-host", Ip: "10.0.0.1"}, "主机名称已存在"},
		{"缺少IP", importRow{Name: "web-10"}, "主机IP不能为空"},
		{"非法IP", importRow{Name: "web-11", Ip: "10.0.0.256"}, "合法的 IPv4"},
		{"非法系统", importRow{Name: "web-12", Ip: "10.0.0.3", Os: "CentOS"}, "系统类型"},
		{"非法状态", importRow{Name: "web-13", Ip: "10.0.0.4", Status: "正常态"}, "状态"},
		{"CPU非数字", importRow{Name: "web-14", Ip: "10.0.0.5", Cpu: "两核"}, "CPU核数"},
		{"分组不存在", importRow{Name: "web-15", Ip: "10.0.0.6", Group: "测试"}, "分组「测试」不存在"},
		{"凭证不存在", importRow{Name: "web-16", Ip: "10.0.0.7", Keys: "admin"}, "登录凭证「admin」不存在"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := newTestContext()
			_, _, _, reason := ctx.validateRow(tc.row)
			if !strings.Contains(reason, tc.want) {
				t.Fatalf("期望错误包含 %q，实际 %q", tc.want, reason)
			}
		})
	}
}

func TestValidateRowRejectsInFileDuplicates(t *testing.T) {
	ctx := newTestContext()
	ctx.usedName["web-20"] = true
	if _, _, _, reason := ctx.validateRow(importRow{Name: "web-20", Ip: "10.0.1.1"}); !strings.Contains(reason, "名称重复") {
		t.Fatalf("文件内重名应被拒绝: %s", reason)
	}
	ctx.usedIp["10.0.1.2"] = true
	if _, _, _, reason := ctx.validateRow(importRow{Name: "web-21", Ip: "10.0.1.2"}); !strings.Contains(reason, "IP重复") {
		t.Fatalf("文件内重复IP应被拒绝: %s", reason)
	}
}
