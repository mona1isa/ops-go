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
func (d *DeptService) GetTree() ([]*api.DeptTree, error) {
	deptList := make([]models.SysDept, 0)

	tx := config.DB.Model(models.SysDept{}).Where(" del_flag = ?", "0")
	if tx.Find(&deptList); tx.Error != nil {
		log.Println("查询部门失败：", tx.Error)
		return nil, errors.New("查询部门失败")
	}

	result := BuildDeptTree(deptList, 0)

	return result, nil
}

func BuildDeptTree(deptList []models.SysDept, parentId int) []*api.DeptTree {
	var tree []*api.DeptTree
	for _, dept := range deptList {
		if dept.ParentId == parentId {
			node := convertToTree(dept)
			children := BuildDeptTree(deptList, dept.ID)
			if len(children) > 0 {
				node.Children = children
			}
			tree = append(tree, node)
		}
	}
	return tree
}

func convertToTree(dept models.SysDept) *api.DeptTree {
	tree := api.DeptTree{
		Name:     dept.Name,
		ParentId: dept.ParentId,
		Status:   dept.Status,
		Remark:   dept.Remark,
	}
	tree.Id = dept.ID
	return &tree
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
	var count int64
	config.DB.Model(models.SysDept{}).Where("parent_id = ?", id).Count(&count)
	if count > 0 {
		return errors.New("该部门下有子部门，无法删除")
	}

	var deptUserCount int64
	config.DB.Model(models.SysUser{}).Where("dept_id = ?", id).Count(&deptUserCount)
	if deptUserCount > 0 {
		return errors.New("该部门下有用户，无法删除")
	}

	if err := services.Delete[models.SysDept](id); err != nil {
		log.Println("删除部门失败：", err)
		return errors.New("删除部门失败")
	}
	return nil
}

func (d *DeptService) UpdateStatus(request *api.DeptStatusRequest) error {
	id := request.Id
	if err := config.DB.Model(models.SysDept{}).Where("id = ?", id).Update("status", request.Status).Error; err != nil {
		log.Println("更新部门状态失败：", err)
		return errors.New("更新部门状态失败")
	}
	return nil
}
