package request

import (
	"github.com/zhany/ops-go/controllers"
	"time"
)

type UserRequest struct {
	Id        uint      `json:"id"`
	DeptId    uint      `json:"deptId" binding:"required"`
	UserName  string    `json:"userName" binding:"required"`
	Nickname  string    `json:"nickName"`
	Email     string    `json:"email" binding:"email"`
	Phone     string    `json:"phone" binding:"required"`
	Sex       int       `json:"sex"`
	Avatar    string    `json:"avatar"`
	Password  string    `json:"password"`
	Status    string    `json:"status"`
	LoginIP   string    `json:"loginIp"`
	LoginDate time.Time `json:"loginDate"`
}

type PageUserRequest struct {
	controllers.PageRequest
	Id        uint      `json:"id"`
	DeptId    uint      `json:"deptId"`
	UserName  string    `json:"userName" `
	Nickname  string    `json:"nickName"`
	Email     string    `json:"email" `
	Phone     string    `json:"phone"`
	Sex       int       `json:"sex"`
	Avatar    string    `json:"avatar"`
	Password  string    `json:"password"`
	Status    string    `json:"status"`
	LoginIP   string    `json:"loginIp"`
	LoginDate time.Time `json:"loginDate"`
}

type EditUserRequest struct {
	Id        uint      `json:"id" binding:"required"`
	DeptId    uint      `json:"deptId"`
	UserName  string    `json:"userName"`
	Nickname  string    `json:"nickName"`
	Email     string    `json:"email" binding:"email"`
	Phone     string    `json:"phone" binding:"required"`
	Sex       int       `json:"sex"`
	Avatar    string    `json:"avatar"`
	Password  string    `json:"password"`
	Status    string    `json:"status"`
	LoginIP   string    `json:"loginIp"`
	LoginDate time.Time `json:"loginDate"`
}
