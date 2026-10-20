// Package mail 实现告警通知的邮件投递。
// SMTP 参数以 JSON 存储在通知渠道的 headers 列，收件人存 url 列（逗号分隔），
// 因此不新增数据库字段；发送使用标准库 net/smtp，支持 SSL(465)/STARTTLS/无加密。
package mail

import (
	"encoding/json"
	"fmt"
	"net"
	"net/mail"
	"strconv"
	"strings"
)

// Config SMTP 账号配置（对应渠道 headers 列的 JSON）
type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	Auth     string `json:"auth"`     // plain | login
	Security string `json:"security"` // ssl | starttls | none
}

// ParseConfig 解析并校验 SMTP 配置；未填的加密方式/认证方式按端口推断
func ParseConfig(headersJSON string) (*Config, error) {
	if strings.TrimSpace(headersJSON) == "" {
		return nil, fmt.Errorf("邮件渠道缺少 SMTP 配置")
	}
	var cfg Config
	if err := json.Unmarshal([]byte(headersJSON), &cfg); err != nil {
		return nil, fmt.Errorf("SMTP 配置需为 JSON 对象")
	}
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.From = strings.TrimSpace(cfg.From)
	cfg.Username = strings.TrimSpace(cfg.Username)
	if cfg.Host == "" {
		return nil, fmt.Errorf("SMTP 主机不能为空")
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return nil, fmt.Errorf("SMTP 端口不合法")
	}
	if _, err := mail.ParseAddress(cfg.From); err != nil {
		return nil, fmt.Errorf("发件人地址不合法: %s", cfg.From)
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Security)) {
	case "ssl", "starttls", "none":
		cfg.Security = strings.ToLower(strings.TrimSpace(cfg.Security))
	case "":
		switch cfg.Port {
		case 465:
			cfg.Security = "ssl"
		default:
			cfg.Security = "starttls"
		}
	default:
		return nil, fmt.Errorf("SMTP 加密方式仅支持 ssl/starttls/none")
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Auth)) {
	case "plain", "login":
		cfg.Auth = strings.ToLower(strings.TrimSpace(cfg.Auth))
	case "":
		cfg.Auth = "plain"
	default:
		return nil, fmt.Errorf("SMTP 认证方式仅支持 plain/login")
	}
	if cfg.Username != "" && strings.TrimSpace(cfg.Password) == "" {
		return nil, fmt.Errorf("SMTP 密码不能为空")
	}
	return &cfg, nil
}

// SplitRecipients 解析收件人列表（逗号/分号/空白分隔），逐个校验邮箱并返回规范地址
func SplitRecipients(raw string) ([]string, error) {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '，' || r == '；' || r == '\n' || r == '\r' || r == ' ' || r == '\t'
	})
	var addrs []string
	for _, field := range fields {
		addr, err := mail.ParseAddress(field)
		if err != nil {
			return nil, fmt.Errorf("收件人地址不合法: %s", field)
		}
		addrs = append(addrs, addr.Address)
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("收件人不能为空")
	}
	return addrs, nil
}

// MaskPassword 将配置 JSON 中的密码替换为掩码，供接口回显
func MaskPassword(headersJSON, mask string) string {
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(headersJSON), &parsed); err != nil {
		return headersJSON
	}
	if _, ok := parsed["password"]; !ok {
		return headersJSON
	}
	masked, _ := json.Marshal(mask)
	parsed["password"] = masked
	out, err := json.Marshal(parsed)
	if err != nil {
		return headersJSON
	}
	return string(out)
}

// RestorePassword 编辑提交时，若密码仍为掩码则沿用库中原值
func RestorePassword(submitted, existing, mask string) string {
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(submitted), &parsed); err != nil {
		return submitted
	}
	quotedMask, _ := json.Marshal(mask)
	value, ok := parsed["password"]
	if !ok || (string(value) != `""` && string(value) != string(quotedMask)) {
		return submitted
	}
	var old map[string]json.RawMessage
	if err := json.Unmarshal([]byte(existing), &old); err != nil {
		return submitted
	}
	if oldPw, ok := old["password"]; ok {
		parsed["password"] = oldPw
	}
	out, err := json.Marshal(parsed)
	if err != nil {
		return submitted
	}
	return string(out)
}

// Addr 返回 host:port 形式的服务器地址
func (c *Config) Addr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}
