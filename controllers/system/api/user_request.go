package api

import (
	"github.com/zhany/ops-go/controllers"
	"time"
)

type UserRequest struct {
	Id        int       `json:"id"`
	DeptId    int       `json:"deptId" binding:"required"`
	UserName  string    `json:"userName" binding:"required"`
	Nickname  string    `json:"nickname"`
	Email     string    `json:"email" binding:"email"`
	Phone     string    `json:"phone" binding:"required"`
	RoleIds   []int     `json:"roleIds" binding:"required"`
	Sex       int       `json:"sex"`
	Avatar    string    `json:"avatar"`
	Password  string    `json:"password"`
	Status    string    `json:"status"`
	LoginIP   string    `json:"loginIp"`
	LoginDate time.Time `json:"loginDate"`
}

type PageUserRequest struct {
	controllers.PageRequest
	Id        int       `json:"id"`
	DeptId    int       `json:"deptId"`
	UserName  string    `json:"userName" `
	Nickname  string    `json:"nickname"`
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
	Id        int       `json:"id" binding:"required"`
	DeptId    int       `json:"deptId"`
	RoleIds   []int     `json:"roleIds"`
	UserName  string    `json:"userName"`
	Nickname  string    `json:"nickname"`
	Email     string    `json:"email" binding:"email"`
	Phone     string    `json:"phone" binding:"required"`
	Sex       int       `json:"sex"`
	Avatar    string    `json:"avatar"`
	Password  string    `json:"password"`
	Status    string    `json:"status"`
	LoginIP   string    `json:"loginIp"`
	LoginDate time.Time `json:"loginDate"`
}

type UserStatusRequest struct {
	Id     int    `json:"id" binding:"required"`
	Status string `json:"status" binding:"required"`
}

type UserInfo struct {
	Id        int      `json:"id"`
	DeptId    int      `json:"deptId"`
	UserName  string   `json:"username"`
	Nickname  string   `json:"nickname"`
	Email     string   `json:"email"`
	Phone     string   `json:"phone"`
	Sex       int      `json:"sex"`
	Avatar    string   `json:"avatar"`
	Status    string   `json:"status"`
	IpAddr    string   `json:"ipAddr"`
	LoginDate string   `json:"loginDate"`
	RoleIds   []int    `json:"roleIds"`
	RoleNames string   `json:"roleNames"`
	Perms     []string `json:"perms"`
}
