package instance

import (
	"errors"
	"fmt"
	"github.com/zhany/ops-go/controllers/instance/api"
	"github.com/zhany/ops-go/models"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type GroupService struct {
}

func (s *GroupService) AddGroup(request api.AddGroupRequest) (err error) {
	name := request.Name
	// 校验名称是否重复
	if isExist, _ := s.IsGroupExist(name, 0); isExist {
		return errors.New("分组名称已存在")
	}

	// 保存分组
	group := models.OpsGroup{
		Name: name,
	}
	parentIdStr := request.ParentId
	if parentIdStr != "" {
		parentId, _ := strconv.Atoi(parentIdStr)
		group.ParentId = parentId
	}

	group.CreateBy = request.CreateBy
	group.UpdateBy = request.UpdateBy
	if err = models.DB.Create(&group).Error; err != nil {
		log.Println("保存分组失败：", err)
		return errors.New("保存分组失败")
	}
	return nil
}

// EditGroup 编辑分组
func (s *GroupService) EditGroup(request api.UpdateGroupRequest) (err error) {
	id := request.Id
	name := request.Name

	var group models.OpsGroup
	if err = models.DB.Where("id = ?", id).First(&group).Error; err != nil {
		log.Println("查询分组失败：", err)
		return errors.New("查询分组失败")
	}

	// 校验名称是否重复
	if isExist, _ := s.IsGroupExist(name, id); isExist {
		return errors.New("分组名称已存在")
	}

	// 更新分组
	group.Name = name
	group.UpdateBy = request.UpdateBy
	if err = models.DB.Model(&models.OpsGroup{}).Where("id = ?", id).Updates(&group).Error; err != nil {
		log.Println("更新分组失败：", err)
		return errors.New("更新分组失败")
	}
	return nil
}

// ListGroup 查询分组树形结构
func (s *GroupService) ListGroup() ([]*models.OpsGroup, error) {
	var all []models.OpsGroup
	if err := models.DB.Find(&all).Error; err != nil {
		log.Println("查询分组失败：", err)
		return nil, errors.New("查询分组失败")
	}

	return buildTree(all), nil
}

// 将查询出来的分组数据组装成树形结构
func buildTree(groups []models.OpsGroup) []*models.OpsGroup {
	var root []*models.OpsGroup
	groupMap := make(map[int]*models.OpsGroup)

	// 初始化映射
	for i := range groups {
		groupMap[groups[i].ID] = &groups[i]
	}

	// 构建树形结构
	for i := range groups {
		if groups[i].ParentId == 0 {
			root = append(root, &groups[i])
		} else {
			if parent, ok := groupMap[groups[i].ParentId]; ok {
				if parent.Children == nil {
					parent.Children = []*models.OpsGroup{}
				}
				parent.Children = append(parent.Children, &groups[i])
			}
		}
	}

	return root
}

// DeleteGroup 删除分组
func (s *GroupService) DeleteGroup(id int) (err error) {
	// 校验是否存在子分组，如果有则不允许删除
	var count int64
	if err = models.DB.Model(&models.OpsGroup{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
		log.Println("查询分组是否存在失败：", err)
		return errors.New("查询分组是否存在失败")
	}
	if count > 0 {
		return errors.New("存在子分组，不允许删除")
	}

	// 校验分组下是否存在实例，如果有则不允许删除
	var instanceCount int64
	if err := models.DB.Model(&models.OpsInstanceGroup{}).Where("group_id = ?", id).Count(&instanceCount).Error; err != nil {
		log.Println("查询分组是否存在实例失败：", err)
		return errors.New("查询分组是否存在实例失败")
	}
	if instanceCount > 0 {
		return errors.New("分组下存在实例，不允许删除")
	}

	// 校验分组是否存在实例，如果有则不允许删除
	if err = models.DB.Where("id = ?", id).Delete(&models.OpsGroup{}).Error; err != nil {
		log.Println("删除分组失败：", err)
		return errors.New("删除分组失败")
	}
	return nil
}

// IsGroupExist 校验分组是否存在
func (s *GroupService) IsGroupExist(name string, id int) (isExist bool, err error) {
	var count int64
	if id > 0 {
		if err = models.DB.Model(&models.OpsGroup{}).Where("id != ? and name = ?", id, name).Count(&count).Error; err != nil {
			log.Println("查询分组是否存在失败：", err)
			return
		}
	} else {
		if err = models.DB.Model(&models.OpsGroup{}).Where("name = ?", name).Count(&count).Error; err != nil {
			log.Println("查询分组是否存在失败：", err)
			return
		}
	}

	if count > 0 {
		isExist = true
	}
	return isExist, nil
}

// GroupInstanceOps 向分组添加/移除实例
func (s *GroupService) GroupInstanceOps(request api.GroupInstanceRequest) (err error) {
	groupId := request.GroupId
	instanceIds := request.InstanceIds
	opsType := request.OpsType

	// 校验分组是否存在
	var count int64
	if err = models.DB.Model(&models.OpsGroup{}).Where("id = ?", groupId).Count(&count).Error; err != nil {
		log.Println("查询分组是否存在失败：", err)
		return errors.New("查询分组是否存在失败")
	}
	if count == 0 {
		return errors.New("分组不存在")
	}

	if len(instanceIds) == 0 {
		return errors.New("请选择实例")
	}

	// 执行分组实例操作
	if opsType == "add" {
		var instanceGroup []models.OpsInstanceGroup
		for _, instanceId := range instanceIds {
			instanceGroup = append(instanceGroup, models.OpsInstanceGroup{
				GroupId:    groupId,
				InstanceId: instanceId,
			})
		}
		if err = models.DB.Create(&instanceGroup).Error; err != nil {
			log.Println("添加实例到分组失败：", err)
			return errors.New("添加实例到分组失败")
		}
	} else if opsType == "remove" {
		if err = models.DB.Where("group_id = ? and instance_id in (?)", groupId, instanceIds).Delete(&models.OpsInstanceGroup{}).Error; err != nil {
			log.Println("移除实例到分组失败：", err)
			return errors.New("移除实例到分组失败")
		}
	} else {
		return errors.New("操作类型错误")
	}
	return nil
}

// PageGroupInstance 分页查询分组实例
func (s *GroupService) PageGroupInstance(request api.PageGroupInstanceRequest) (page api.PageGroupInstanceResponse, err error) {
	// 校验分组是否存在
	var count int64
	if err = models.DB.Model(&models.OpsGroup{}).Where("id = ?", request.GroupId).Count(&count).Error; err != nil {
		log.Println("查询分组是否存在失败：", err)
		return page, errors.New("查询分组是否存在失败")
	}
	if count == 0 {
		return page, errors.New("分组不存在")
	}
	// 分页查询分组下实例
	// SELECT * FROM ops_instance WHERE id IN (SELECT instance_id FROM ops_instance_group WHERE group_id = ?) OFFSET pageNum*pageSize LIMIT pageSize
	var instances []models.OpsInstance
	if err = models.DB.Table("ops_instance").Select("ops_instance.*").Joins("JOIN ops_instance_group ON ops_instance.id = ops_instance_group.instance_id").Where("ops_instance_group.group_id = ?", request.GroupId).Offset((request.PageNum - 1) * request.PageSize).Limit(request.PageSize).Find(&instances).Error; err != nil {
		log.Println("查询分组实例失败：", err)
		return page, errors.New("查询分组实例失败")
	}
	// 查询分组实例总数
	if err = models.DB.Table("ops_instance").Joins("JOIN ops_instance_group ON ops_instance.id = ops_instance_group.instance_id").Where("ops_instance_group.group_id = ?", request.GroupId).Count(&page.Total).Error; err != nil {
		log.Println("查询分组实例总数失败：", err)
		return page, errors.New("查询分组实例总数失败")
	}
	page.Data = instances
	page.PageNum = request.PageNum
	page.PageSize = request.PageSize
	return page, nil
}

// ScanHosts 扫描网段内的主机，检测22和3389端口
func (s *GroupService) ScanHosts(ipRange string) ([]api.ScannedHost, error) {
	var hosts []api.ScannedHost
	var ips []string

	// 解析IP网段
	if strings.Contains(ipRange, "/") {
		// CIDR格式，如 192.168.1.0/24
		ips = parseCIDR(ipRange)
	} else if strings.Contains(ipRange, "-") {
		// 范围格式，如 192.168.1.1-100
		ips = parseRange(ipRange)
	} else {
		// 单个IP
		ips = []string{ipRange}
	}

	if len(ips) == 0 {
		return hosts, errors.New("无效的IP网段格式")
	}

	// 限制扫描的IP数量，避免过多
	if len(ips) > 256 {
		ips = ips[:256]
	}

	// 并发扫描
	var wg sync.WaitGroup
	hostChan := make(chan api.ScannedHost, len(ips))
	semaphore := make(chan struct{}, 50) // 限制并发数量

	for _, ip := range ips {
		wg.Add(1)
		go func(ipAddr string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 检测SSH端口(22)
			if isOpen, _ := checkPort(ipAddr, 22, 2*time.Second); isOpen {
				hostChan <- api.ScannedHost{
					Ip:     ipAddr,
					Port:   22,
					OsType: "Linux",
				}
				return
			}

			// 检测RDP端口(3389)
			if isOpen, _ := checkPort(ipAddr, 3389, 2*time.Second); isOpen {
				hostChan <- api.ScannedHost{
					Ip:     ipAddr,
					Port:   3389,
					OsType: "Windows",
				}
				return
			}
		}(ip)
	}

	// 等待所有扫描完成
	go func() {
		wg.Wait()
		close(hostChan)
	}()

	// 收集结果
	for host := range hostChan {
		hosts = append(hosts, host)
	}

	return hosts, nil
}

// 解析CIDR格式的IP网段
func parseCIDR(cidr string) []string {
	var ips []string
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return ips
	}

	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
		ips = append(ips, ip.String())
	}

	// 移除网络地址和广播地址
	if len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
	}
	return ips
}

// 解析范围格式的IP
func parseRange(ipRange string) []string {
	var ips []string
	parts := strings.Split(ipRange, "-")
	if len(parts) != 2 {
		return ips
	}

	baseIP := parts[0]
	endNum, err := strconv.Atoi(parts[1])
	if err != nil {
		return ips
	}

	// 提取基础IP的前三段
	ipParts := strings.Split(baseIP, ".")
	if len(ipParts) != 4 {
		return ips
	}

	startNum, _ := strconv.Atoi(ipParts[3])
	prefix := strings.Join(ipParts[:3], ".")

	for i := startNum; i <= endNum && i <= 255; i++ {
		ips = append(ips, fmt.Sprintf("%s.%d", prefix, i))
	}

	return ips
}

// IP增量
func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// 检测端口是否开放
func checkPort(ip string, port int, timeout time.Duration) (bool, error) {
	address := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false, err
	}
	conn.Close()
	return true, nil
}

// SaveScannedHosts 保存扫描到的主机
func (s *GroupService) SaveScannedHosts(request api.SaveScannedHostsRequest, userId string) error {
	groupId := request.GroupId
	hosts := request.Hosts

	if len(hosts) == 0 {
		return errors.New("没有要保存的主机")
	}

	// 校验分组是否存在
	var count int64
	if err := models.DB.Model(&models.OpsGroup{}).Where("id = ?", groupId).Count(&count).Error; err != nil {
		log.Println("查询分组是否存在失败：", err)
		return errors.New("查询分组是否存在失败")
	}
	if count == 0 {
		return errors.New("分组不存在")
	}

	// 查询已存在的IP，避免重复添加
	var existingIPs []string
	for _, host := range hosts {
		existingIPs = append(existingIPs, host.Ip)
	}

	var existingInstances []models.OpsInstance
	models.DB.Where("ip IN ?", existingIPs).Find(&existingInstances)

	existingIPMap := make(map[string]int)
	for _, inst := range existingInstances {
		existingIPMap[inst.Ip] = inst.ID
	}

	var newInstances []models.OpsInstance
	var instanceGroups []models.OpsInstanceGroup

	for _, host := range hosts {
		if instanceId, exists := existingIPMap[host.Ip]; exists {
			// IP已存在，只添加分组关联（如果还没有关联）
			var count int64
			models.DB.Model(&models.OpsInstanceGroup{}).Where("group_id = ? AND instance_id = ?", groupId, instanceId).Count(&count)
			if count == 0 {
				instanceGroups = append(instanceGroups, models.OpsInstanceGroup{
					GroupId:    groupId,
					InstanceId: instanceId,
				})
			}
		} else {
			// 新主机 - 使用扫描结果设置字段
			var osName string
			var cpu, memMb, diskGb int
			if host.OsType == "Linux" {
				osName = "Linux"
				cpu = 1
				memMb = 1024
				diskGb = 20
			} else {
				osName = "Windows"
				cpu = 2
				memMb = 4096
				diskGb = 50
			}
			inst := models.OpsInstance{
				Name:   host.Ip,
				Ip:     host.Ip,
				Cpu:    cpu,
				MemMb:  memMb,
				DiskGb: diskGb,
				Os:     osName,
				Status: "1",
			}
			inst.CreateBy = userId
			inst.UpdateBy = userId
			newInstances = append(newInstances, inst)
		}
	}

	// 开启事务
	tx := models.DB.Begin()

	// 批量插入新主机
	if len(newInstances) > 0 {
		if err := tx.Create(&newInstances).Error; err != nil {
			tx.Rollback()
			log.Println("批量插入主机失败：", err)
			return errors.New("保存主机失败")
		}

		// 为新插入的主机添加分组关联
		for _, inst := range newInstances {
			instanceGroups = append(instanceGroups, models.OpsInstanceGroup{
				GroupId:    groupId,
				InstanceId: inst.ID,
			})
		}
	}

	// 批量插入分组关联
	if len(instanceGroups) > 0 {
		if err := tx.Create(&instanceGroups).Error; err != nil {
			tx.Rollback()
			log.Println("添加主机到分组失败：", err)
			return errors.New("添加主机到分组失败")
		}
	}

	tx.Commit()
	return nil
}

// AvailableInstance 查询不在分组内的实例用于添加到分组中
func (s *GroupService) AvailableInstance(request api.PageGroupInstanceRequest) (page api.PageGroupInstanceResponse, err error) {
	// 校验分组是否存在
	var count int64
	if err = models.DB.Model(&models.OpsGroup{}).Where("id = ?", request.GroupId).Count(&count).Error; err != nil {
		log.Println("查询分组是否存在失败：", err)
		return page, errors.New("查询分组是否存在失败")
	}
	if count == 0 {
		return page, errors.New("分组不存在")
	}

	// 查询分组内实例ID列表
	var instanceIds []string
	if err = models.DB.Table("ops_instance_group").Select("instance_id").Where("group_id = ?", request.GroupId).Find(&instanceIds).Error; err != nil {
		log.Println("查询分组实例列表失败：", err)
		return page, errors.New("查询分组实例列表失败")
	}

	// 分页查询不在分组内的实例
	var instances []models.OpsInstance
	if len(instanceIds) == 0 {
		if err := models.DB.Table("ops_instance").Offset((request.PageNum - 1) * request.PageSize).Limit(request.PageSize).Find(&instances).Error; err != nil {
			log.Println("查询分组实例失败：", err)
			return page, errors.New("查询分组实例失败")
		}
		// 查询分组实例总数
		if err := models.DB.Table("ops_instance").Count(&page.Total).Error; err != nil {
			log.Println("查询分组实例总数失败：", err)
			return page, errors.New("查询分组实例总数失败")
		}
	} else {
		if err := models.DB.Table("ops_instance").Where("id NOT IN (?)", instanceIds).Offset((request.PageNum - 1) * request.PageSize).Limit(request.PageSize).Find(&instances).Error; err != nil {
			log.Println("查询分组实例失败：", err)
			return page, errors.New("查询分组实例失败")
		}
		// 查询分组实例总数
		if err := models.DB.Table("ops_instance").Where("id NOT IN (?)", instanceIds).Count(&page.Total).Error; err != nil {
			log.Println("查询分组实例总数失败：", err)
			return page, errors.New("查询分组实例总数失败")
		}
	}

	page.Data = instances
	page.PageNum = request.PageNum
	page.PageSize = request.PageSize
	return page, nil
}
