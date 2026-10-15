package models

// OpsCmdbSubnet IPAM 网段
type OpsCmdbSubnet struct {
	Name    string `gorm:"column:name;type:varchar(64);not null;uniqueIndex:uk_cmdb_subnet_name;comment:网段名称" json:"name"`
	Cidr    string `gorm:"column:cidr;type:varchar(64);not null;comment:CIDR，如 10.0.1.0/24" json:"cidr"`
	Gateway string `gorm:"column:gateway;type:varchar(64);comment:网关" json:"gateway"`
	Vlan    string `gorm:"column:vlan;type:varchar(32);comment:VLAN" json:"vlan"`
	Base
}

const TableOpsCmdbSubnet = "ops_cmdb_subnet"

func (OpsCmdbSubnet) TableName() string {
	return TableOpsCmdbSubnet
}
