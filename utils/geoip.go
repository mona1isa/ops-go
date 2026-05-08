package utils

import (
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

var (
	searcher *xdb.Searcher
	once     sync.Once
	dbLoaded bool
)

// loadDB loads the ip2region database into memory (once).
func loadDB() {
	once.Do(func() {
		dbPath := filepath.Join("data", "ip2region.xdb")

		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			log.Printf("[GeoIP] 数据文件不存在: %s, 跳过IP地域检查", dbPath)
			return
		}

		cBuff, err := xdb.LoadContentFromFile(dbPath)
		if err != nil {
			log.Printf("[GeoIP] 加载数据文件失败: %v, 跳过IP地域检查", err)
			return
		}

		s, err := xdb.NewWithBuffer(xdb.IPv4, cBuff)
		if err != nil {
			log.Printf("[GeoIP] 初始化搜索器失败: %v, 跳过IP地域检查", err)
			return
		}

		searcher = s
		dbLoaded = true
		log.Println("[GeoIP] ip2region 数据库加载成功")
	})
}

// isPrivateIP checks if the IP is a private/internal network address.
func isPrivateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	privateBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}
	for _, cidr := range privateBlocks {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(parsedIP) {
			return true
		}
	}
	return false
}

// GetRegion returns the region string for the given IP (e.g. "中国|0|广东省|深圳市|电信").
// Returns empty string if the IP cannot be resolved.
func GetRegion(ip string) (string, error) {
	loadDB()
	if !dbLoaded {
		return "", nil
	}

	// Skip private IPs
	if isPrivateIP(ip) {
		return "内网", nil
	}

	// Skip IPv6
	if strings.Contains(ip, ":") {
		log.Printf("[GeoIP] IPv6 地址暂不支持地域查询: %s", ip)
		return "", nil
	}

	region, err := searcher.Search(ip)
	if err != nil {
		return "", err
	}
	return region, nil
}

// IsChinaIP checks if the given IP address is from China.
// Private/internal IPs are treated as China (assumed to be within the internal network).
// If the GeoIP database is not available, returns true (fail-open).
func IsChinaIP(ip string) bool {
	loadDB()
	if !dbLoaded {
		return true
	}

	// Private IPs are treated as China
	if isPrivateIP(ip) {
		return true
	}

	// IPv6: allow (not supported by ip2region)
	if strings.Contains(ip, ":") {
		log.Printf("[GeoIP] IPv6 地址暂不支持地域判断, 默认放行: %s", ip)
		return true
	}

	region, err := searcher.Search(ip)
	if err != nil {
		log.Printf("[GeoIP] IP地域查询失败: %s, 错误: %v, 默认放行", ip, err)
		return true
	}

	// Region format: 国家|区域|省份|城市|ISP
	country := strings.Split(region, "|")
	if len(country) > 0 && country[0] == "中国" {
		return true
	}

	return false
}
