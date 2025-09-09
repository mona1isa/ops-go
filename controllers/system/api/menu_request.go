package api

// AddMenuRequest 添加菜单请求参数
type AddMenuRequest struct {
	Name      string `json:"name" validate:"required"` // 菜单名称
	ParentId  int    `json:"parentId"`                 // 父菜单ID
	OrderNum  int    `json:"orderNum"`                 // 排序
	Path      string `json:"path"`                     // 路由地址
	Component string `json:"component"`                // 组件路径
	IsAffix   bool   `json:"isAffix"`                  // 是否固定 1 固定  0 不固定
	IsFrame   int    `json:"isFrame"`                  // 是否为外链（1是 0否）
	IsCache   int    `json:"isCache"`                  // 是否缓存（1缓存 0不缓存）
	Type      string `json:"type"`                     // 菜单类型： M目录 C菜单 F按钮
	Visible   string `json:"visible"`                  // 显示状态（1显示 0隐藏）
	Status    string `json:"status"`                   // 菜单状态（1正常 0停用）
	Perms     string `json:"perms"`                    // 权限标识
	Icon      string `json:"icon"`                     // 菜单图标
	CreateBy  string `json:"createBy"`
	UpdateBy  string `json:"updateBy"`
}

// MenuListRequest 菜单列表请求参数
type MenuListRequest struct {
	Name   string `json:"name"`   // 菜单名称
	Status string `json:"status"` // 菜单状态
}

// EditMenuRequest 编辑菜单请求参数
type EditMenuRequest struct {
	Id int `json:"id" validate:"required"` // 菜单ID
	AddMenuRequest
}

type Meta struct {
	KeepAlive bool     `json:"keepAlive"` // 是否缓存
	Title     string   `json:"title"`     // 菜单标题
	IsLink    bool     `json:"isLink"`    // 是否为外链
	IsHide    bool     `json:"isHide"`    // 是否隐藏
	IsAffix   bool     `json:"isAffix"`   // 是否固定
	IsIframe  bool     `json:"isFrame"`   // 是否为内嵌
	Roles     []string `json:"roles"`     // 角色
	Icon      string   `json:"icon"`      // 菜单图标
}

type MenuVo struct {
	Id        int    `json:"id"`        // 菜单ID
	Name      string `json:"name"`      // 菜单名称
	ParentId  int    `json:"parent_id"` // 父菜单ID
	OrderNum  int    `json:"orderNum"`  // 排序
	Path      string `json:"path"`      // 路由地址
	Component string `json:"component"` // 组件路径
	IsAffix   bool   `json:"isAffix"`   // 是否固定 true 固定  false 不固定
	IsFrame   int    `json:"isFrame"`   // 是否为外链（1是 0否）
	IsCache   int    `json:"isCache"`   // 是否缓存（1缓存 0不缓存）
	Type      string `json:"type"`      // 菜单类型： M目录 C菜单 F按钮
	Visible   string `json:"visible"`   // 显示状态（1显示 0隐藏）
	Status    string `json:"status"`    // 菜单状态（1正常 0停用）
	Perms     string `json:"perms"`     // 权限标识
	Icon      string `json:"icon"`      // 菜单图标
}

type MenuTree struct {
	Id        int         `json:"id"`                 // 菜单ID
	Name      string      `json:"name"`               // 菜单名称
	ParentId  int         `json:"parent_id"`          // 父菜单ID
	OrderNum  int         `json:"orderNum"`           // 排序
	Path      string      `json:"path"`               // 路由地址
	Component string      `json:"component"`          // 组件路径
	IsAffix   bool        `json:"isAffix"`            // 是否固定 true 固定  false 不固定
	IsFrame   int         `json:"isFrame"`            // 是否为外链（1是 0否）
	IsCache   int         `json:"isCache"`            // 是否缓存（1缓存 0不缓存）
	Type      string      `json:"type"`               // 菜单类型： M目录 C菜单 F按钮
	Visible   string      `json:"visible"`            // 显示状态（1显示 0隐藏）
	Status    string      `json:"status"`             // 菜单状态（1正常 0停用）
	Perms     string      `json:"perms"`              // 权限标识
	Icon      string      `json:"icon"`               // 菜单图标
	Meta      Meta        `json:"meta"`               // 菜单元数据
	Children  []*MenuTree `json:"children,omitempty"` // 子菜单
}
