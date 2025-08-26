package request

type RoleRequest struct {
	Name     string `json:"name" binding:"required"`
	OrderNum int    `json:"order_num"`
	Status   string `json:"status"`
}
