package api

type AddDeptRequest struct {
	Name     string `json:"name" binding:"required"`
	ParentId int    `json:"parentId"`
	OrderNum int    `json:"orderNum"`
	Status   bool   `json:"status"`
	Remark   string `json:"remark"`
	CreateBy string `json:"createBy"`
	UpdateBy string `json:"updateBy"`
}

type EditDeptRequest struct {
	Id int `json:"id" binding:"required"`
	AddDeptRequest
}

type QueryDeptRequest struct {
	Name string `json:"name"`
}

type DeptTree struct {
	Id       int         `json:"id"`
	Name     string      `json:"name"`
	Status   bool        `json:"status"`
	ParentId int         `json:"parentId"`
	Remark   string      `json:"remark"`
	Children []*DeptTree `json:"children,omitempty"`
}

type DeptStatusRequest struct {
	Id     int  `json:"id"`
	Status bool `json:"status"`
}
