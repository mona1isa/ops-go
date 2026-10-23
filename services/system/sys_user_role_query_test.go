package system

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
	controllers "github.com/zhany/ops-go/controllers"
	api "github.com/zhany/ops-go/controllers/system/api"
	"github.com/zhany/ops-go/models"
)

// TestGetUserRoleQueryPopulatesFields 守住用户角色查询的列名大小写坑：
// PostgreSQL 会把未加引号的标识符折叠成小写，SELECT 里若用 camelCase 别名
// （AS roleName），驱动返回的列名是 rolename，GORM 匹配不到字段，Scan 出的每一行全为零值，
// 「关联角色」等依赖该结果的界面就会空白。MySQL 保留别名大小写，所以此问题只在 PG 侧暴露。
func TestGetUserRoleQueryPopulatesFields(t *testing.T) {
	if os.Getenv("DB_AUTOINIT") == "off" {
		t.Skip("DB_AUTOINIT=off，跳过需要数据库的测试")
	}
	// go test 的工作目录是本包目录，.env 位于模块根目录
	for _, p := range []string{".env", "../../.env"} {
		if _, err := os.Stat(p); err == nil {
			_ = godotenv.Load(p)
			break
		}
	}
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		t.Skip("未配置 DB_DSN，跳过")
	}
	// 直接建连而不走 models.InitDB，避免测试触发 AutoMigrate
	db, err := models.Open(dsn)
	if err != nil {
		t.Skipf("数据库不可达，跳过：%v", err)
	}

	var rows []models.SysUserRoleResult
	err = db.Model(&models.SysUser{}).
		Joins("JOIN sys_user_role ON sys_user_role.user_id = sys_user.id").
		Joins("JOIN sys_role ON sys_role.id = sys_user_role.role_id").
		Select("sys_user.id AS user_id, sys_role.name AS role_name, sys_role.id AS role_id").
		Scan(&rows).Error
	if err != nil {
		t.Fatalf("查询用户角色失败：%v", err)
	}
	if len(rows) == 0 {
		t.Skip("sys_user_role 无数据，无法验证字段填充")
	}
	for i, r := range rows {
		if r.UserId == 0 || r.RoleId == 0 || r.RoleName == "" {
			t.Fatalf("第 %d 行扫描结果为零值（列名与 gorm column 标签不匹配）：%+v", i, r)
		}
	}
	t.Logf("读到 %d 条用户角色关联，字段填充正常", len(rows))

	// 端到端：直接复用同一连接调用 UserService.Page，确认接口返回的 roleNames 非空。
	// models.DB 是导出变量，这里赋值可跳过 models.InitDB 的 AutoMigrate。
	models.DB = db
	page, err := (&UserService{}).Page(&api.PageUserRequest{PageRequest: controllers.PageRequest{PageNum: 1, PageSize: 10}})
	if err != nil {
		t.Fatalf("查询用户列表失败：%v", err)
	}
	matched := 0
	for _, u := range page.Data {
		if u.RoleNames != "" {
			matched++
		}
	}
	if matched == 0 {
		t.Fatal("用户列表的 roleNames 全部为空，前端「关联角色」列会不显示")
	}
	t.Logf("用户列表 %d 行中 %d 行带角色：%+v", len(page.Data), matched, page.Data[0].RoleNames)
}
