package system

import (
	"errors"
	"github.com/zhany/ops-go/config"
	"github.com/zhany/ops-go/controllers/system/api"
	"github.com/zhany/ops-go/models"
	"github.com/zhany/ops-go/services"
	"log"
)

type DeptService struct {
}

// Add 新增部门
func (d *DeptService) Add(request *api.AddDeptRequest) error {
	name := request.Name
	var count int64
	config.DB.Model(models.SysDept{}).Where("name = ?", name).Count(&count)
	if count > 0 {
		return errors.New("部门名称已存在")
	}

	dept := models.SysDept{
		Name:     name,
		ParentId: request.ParentId,
		OrderNum: request.OrderNum,
		Status:   request.Status,
	}

	dept.CreateBy = request.CreateBy
	dept.UpdateBy = request.UpdateBy
	dept.Remark = request.Remark

	if err := services.Create[models.SysDept](&dept); err != nil {
		log.Println("新增部门失败：", err)
		return errors.New("新增部门失败")
	}
	return nil
}

// Edit 编辑部门
func (d *DeptService) Edit(request *api.EditDeptRequest) error {
	id := request.Id
	name := request.Name
	var count int64
	config.DB.Model(models.SysDept{}).Where("id = ? and name <> ?", id, name).Count(&count)
	if count > 0 {
		return errors.New("部门名称已存在")
	}

	dept := map[string]any{
		"name":      name,
		"order_num": request.OrderNum,
		"status":    request.Status,
		"update_by": request.UpdateBy,
	}
	if err := services.Update[models.SysDept](id, dept); err != nil {
		log.Println("编辑部门失败：", err)
		return errors.New("编辑部门失败")
	}

	return nil
}

// Page 分页查询部门
func (d *DeptService) GetTree() error {
	return nil
}

// List 部门列表
func (d *DeptService) List(request *api.QueryDeptRequest) ([]models.SysDept, error) {
	var deptList []models.SysDept
	tx := config.DB.Where("del_flag = ?", "0")
	if request.Name != "" {
		tx = tx.Where("name = ?", request.Name)
	}
	tx.Find(&deptList)
	return deptList, nil
}

// Delete 删除部门
func (d *DeptService) Delete(id int) error {
	if err := services.Delete[models.SysDept](id); err != nil {
		log.Println("删除部门失败：", err)
		return errors.New("删除部门失败")
	}
	return nil
}
