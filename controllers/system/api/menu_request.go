package api

// AddMenuRequest 添加菜单请求参数
type AddMenuRequest struct {
	Name          string `json:"name" validate:"required"` // 菜单名称
	ParentId      int    `json:"parentId"`                 // 父菜单ID
	OrderNum      int    `json:"orderNum"`                 // 排序
	Path          string `json:"path"`                     // 路由地址
	Component     string `json:"component"`                // 组件路径
	IsAffix       bool   `json:"isAffix"`                  // 是否固定 	(true 固定  false 不固定)
	IsIframe      bool   `json:"isIframe"`                 // 是否为内嵌 	(true是 false否)
	IsLink        bool   `json:"isLink"`                   // 是否为外链	(true是 false否)
	KeepAlive     bool   `json:"keepAlive"`                // 是否缓存	(true缓存 false不缓存)
	Type          string `json:"type"`                     // 菜单类型 	(M目录 C菜单 F按钮)
	IsHide        bool   `json:"isHide"`                   // 显示状态	(true显示 false隐藏)
	Status        bool   `json:"status"`                   // 菜单状态	（true正常 false停用）
	Url           string `json:"url"`                      // 外链地址
	Perms         string `json:"perms"`                    // 权限标识
	Icon          string `json:"icon"`                     // 菜单图标
	RequestUrl    string `json:"requestUrl"`               // 请求地址
	RequestMethod string `json:"requestMethod"`            // 请求方法
	CreateBy      string `json:"createBy"`
	UpdateBy      string `json:"updateBy"`
}

// MenuListRequest 菜单列表请求参数
type MenuListRequest struct {
	Name string `json:"name"` // 菜单名称
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
	IsIframe  bool     `json:"isIframe"`  // 是否为内嵌
	Roles     []string `json:"roles"`     // 角色
	Icon      string   `json:"icon"`      // 菜单图标
}

type MenuVo struct {
	Id        int    `json:"id"`        // 菜单ID
	Name      string `json:"name"`      // 菜单名称
	ParentId  int    `json:"parentId"`  // 父菜单ID
	OrderNum  int    `json:"orderNum"`  // 排序
	Path      string `json:"path"`      // 路由地址
	Component string `json:"component"` // 组件路径
	IsLink    bool   `json:"isLink"`    // 是否为外链
	IsAffix   bool   `json:"isAffix"`   // 是否固定 true 固定  false 不固定
	IsFrame   bool   `json:"isFrame"`   // 是否为外链（true是 false否）
	KeepAlive bool   `json:"keepAlive"` // 是否缓存（true缓存 false不缓存）
	Type      string `json:"type"`      // 菜单类型： M目录 C菜单 F按钮
	IsHide    string `json:"isHide"`    // 显示状态（true显示 false隐藏）
	Status    bool   `json:"status"`    // 菜单状态（true正常 false停用）
	Url       string `json:"url"`       // 外链地址
	Perms     string `json:"perms"`     // 权限标识
	Icon      string `json:"icon"`      // 菜单图标
}

type MenuTree struct {
	Id            int         `json:"id"`                 // 菜单ID
	Name          string      `json:"name"`               // 菜单名称
	ParentId      int         `json:"parentId"`           // 父菜单ID
	OrderNum      int         `json:"orderNum"`           // 排序
	Path          string      `json:"path"`               // 路由地址
	Component     string      `json:"component"`          // 组件路径
	Type          string      `json:"type"`               // 菜单类型 (M目录 C菜单 F按钮)
	Status        bool        `json:"status"`             // 菜单状态 (true 正常 false 停用)
	Perms         string      `json:"perms"`              // 权限标识
	Icon          string      `json:"icon"`               // 菜单图标
	Url           string      `json:"url"`                // 外链地址
	RequestMethod string      `json:"requestMethod"`      // 请求方法
	RequestUrl    string      `json:"requestUrl"`         // 请求地址
	Meta          Meta        `json:"meta"`               // 菜单元数据
	Children      []*MenuTree `json:"children,omitempty"` // 子菜单
}
