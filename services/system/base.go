package system

import (
	"context"
	"errors"
	"github.com/zhany/ops-go/config"
	"gorm.io/gorm"
)

var ctx = context.Background()

// Create 创建记录
func Create[T any](entity *T) error {
	return gorm.G[T](config.DB).Create(ctx, entity)
}

// FindById 查询所有记录
func FindById[T any](id int) (T, error) {
	first, err := gorm.G[T](config.DB).Where("id=?", id).First(ctx)
	if err != nil {
		return first, err
	}

	return first, nil
}

// Update 更新记录
func Update[T any](id int, updates map[string]interface{}) error {
	return config.DB.Model(new(T)).Where("id = ?", id).Updates(updates).Error
}

// Delete 删除记录
func Delete[T any](id int) error {
	affected, err := gorm.G[T](config.DB).Where("id = ?", id).Delete(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("删除失败")
	}
	return nil
}

// FindAll 批量查询
func FindAll[T any]() ([]T, error) {
	var entities []T
	if err := config.DB.Find(&entities).Error; err != nil {
		return nil, err
	}
	return entities, nil
}
