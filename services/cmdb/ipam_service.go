package cmdb

import (
	"errors"
	"log"
	"math/big"
	"net"
	"strings"

	"github.com/zhany/ops-go/models"
	"gorm.io/gorm"
)

// SubnetService IPAM 网段管理
type SubnetService struct{}

// SubnetUsage IP 占用明细
type SubnetUsage struct {
	Ip       string `json:"ip"`
	Kind     string `json:"kind"` // asset 资产 / instance 纳管主机
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Owner    string `json:"owner"`
	Status   string `json:"status"`
	Location string `json:"location"`
}

// UsageResult 网段使用情况
type UsageResult struct {
	Capacity int64         `json:"capacity"`
	Used     int           `json:"used"`
	Usage    []SubnetUsage `json:"usage"`
}

func parseCidr(cidr string) (*net.IPNet, error) {
	_, network, err := net.ParseCIDR(strings.TrimSpace(cidr))
	if err != nil {
		return nil, errors.New("网段格式不合法，示例：10.0.1.0/24")
	}
	return network, nil
}

// cidrCapacity 可分配主机地址数（/31 以上按地址数计）
func cidrCapacity(network *net.IPNet) int64 {
	ones, bits := network.Mask.Size()
	hostBits := bits - ones
	if hostBits >= 63 {
		return 1 << 62
	}
	total := new(big.Int).Lsh(big.NewInt(1), uint(hostBits))
	if hostBits > 1 {
		total.Sub(total, big.NewInt(2)) // 网络地址与广播地址
	}
	if !total.IsInt64() {
		return 1<<62 - 1
	}
	return total.Int64()
}

func (s *SubnetService) List(pageNum, pageSize int, keyword string) (models.PageResult[models.OpsCmdbSubnet], error) {
	result, err := models.Paginate[models.OpsCmdbSubnet](models.DB, pageNum, pageSize, func(db *gorm.DB) *gorm.DB {
		db = db.Where("del_flag = ?", "0")
		if keyword != "" {
			kw := "%" + escapeLike(keyword) + "%"
			db = db.Where("name LIKE ? OR cidr LIKE ?", kw, kw)
		}
		return db.Order("id")
	})
	if err != nil {
		log.Println("查询网段失败:", err)
		return result, errors.New("查询网段失败")
	}
	return result, nil
}

func (s *SubnetService) Add(subnet *models.OpsCmdbSubnet) error {
	if strings.TrimSpace(subnet.Name) == "" {
		return errors.New("网段名称不能为空")
	}
	network, err := parseCidr(subnet.Cidr)
	if err != nil {
		return err
	}
	subnet.Cidr = network.String()
	var count int64
	if err := models.DB.Model(&models.OpsCmdbSubnet{}).
		Where("del_flag = ? AND (name = ? OR cidr = ?)", "0", subnet.Name, subnet.Cidr).Count(&count).Error; err != nil {
		log.Println("校验网段失败:", err)
		return errors.New("校验网段失败")
	}
	if count > 0 {
		return errors.New("网段名称或 CIDR 已存在")
	}
	subnet.DelFlag = "0"
	if err := models.DB.Create(subnet).Error; err != nil {
		log.Println("新增网段失败:", err)
		return errors.New("新增网段失败")
	}
	return nil
}

func (s *SubnetService) Edit(subnet *models.OpsCmdbSubnet) error {
	var existing models.OpsCmdbSubnet
	if err := models.DB.Where("id = ? AND del_flag = ?", subnet.ID, "0").First(&existing).Error; err != nil {
		return errors.New("网段不存在")
	}
	if strings.TrimSpace(subnet.Name) == "" {
		return errors.New("网段名称不能为空")
	}
	network, err := parseCidr(subnet.Cidr)
	if err != nil {
		return err
	}
	subnet.Cidr = network.String()
	var count int64
	err = models.DB.Model(&models.OpsCmdbSubnet{}).
		Where("del_flag = ? AND id != ? AND (name = ? OR cidr = ?)", "0", subnet.ID, subnet.Name, subnet.Cidr).
		Count(&count).Error
	if err != nil {
		log.Println("校验网段失败:", err)
		return errors.New("校验网段失败")
	}
	if count > 0 {
		return errors.New("网段名称或 CIDR 已存在")
	}
	updates := map[string]any{
		"name": subnet.Name, "cidr": subnet.Cidr, "gateway": subnet.Gateway,
		"vlan": subnet.Vlan, "remark": subnet.Remark, "update_by": subnet.UpdateBy,
	}
	if err := models.DB.Model(&existing).Updates(updates).Error; err != nil {
		log.Println("编辑网段失败:", err)
		return errors.New("编辑网段失败")
	}
	return nil
}

func (s *SubnetService) Delete(id int) error {
	var existing models.OpsCmdbSubnet
	if err := models.DB.Where("id = ? AND del_flag = ?", id, "0").First(&existing).Error; err != nil {
		return errors.New("网段不存在")
	}
	if err := models.DB.Model(&existing).Update("del_flag", "1").Error; err != nil {
		log.Println("删除网段失败:", err)
		return errors.New("删除网段失败")
	}
	return nil
}

// Usage 计算某网段的容量与占用：资产表非空 IP + 纳管主机 IP
func (s *SubnetService) Usage(id int) (*UsageResult, error) {
	var subnet models.OpsCmdbSubnet
	if err := models.DB.Where("id = ? AND del_flag = ?", id, "0").First(&subnet).Error; err != nil {
		return nil, errors.New("网段不存在")
	}
	network, err := parseCidr(subnet.Cidr)
	if err != nil {
		return nil, err
	}

	result := &UsageResult{Capacity: cidrCapacity(network), Usage: []SubnetUsage{}}

	var assets []models.OpsCmdbAsset
	if err := models.DB.Where("del_flag = ? AND ip <> ''", "0").Find(&assets).Error; err != nil {
		log.Println("查询资产 IP 失败:", err)
		return nil, errors.New("查询资产失败")
	}
	for _, a := range assets {
		ip := net.ParseIP(strings.TrimSpace(a.Ip))
		if ip != nil && network.Contains(ip) {
			result.Used++
			result.Usage = append(result.Usage, SubnetUsage{
				Ip: a.Ip, Kind: "asset", Id: a.ID, Name: a.Name,
				Owner: a.Owner, Status: statusText(a.Status), Location: a.Location,
			})
		}
	}

	var instances []models.OpsInstance
	if err := models.DB.Where("del_flag = ?", "0").Find(&instances).Error; err != nil {
		log.Println("查询主机 IP 失败:", err)
		return nil, errors.New("查询主机失败")
	}
	for _, i := range instances {
		ip := net.ParseIP(strings.TrimSpace(i.Ip))
		if ip == nil || !network.Contains(ip) {
			continue
		}
		duplicated := false
		for _, u := range result.Usage {
			if u.Kind == "instance" && u.Id == i.ID {
				duplicated = true
				break
			}
		}
		if duplicated {
			continue
		}
		onlineText := "离线"
		if i.OnlineStatus == "1" {
			onlineText = "在线"
		}
		result.Used++
		result.Usage = append(result.Usage, SubnetUsage{
			Ip: i.Ip, Kind: "instance", Id: i.ID, Name: i.Name, Status: onlineText,
		})
	}
	return result, nil
}
