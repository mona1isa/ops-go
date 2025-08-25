package system

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/google/uuid"
	"github.com/steambap/captcha"
	"github.com/zhany/ops-go/utils"
	"log"
	"strings"
	"time"
)

type Captcha struct {
	Uuid string // 验证码uuid
	Text string // 验证码内容
	Img  string // 验证码图片
}

type CaptchaService struct {
}

// GenerateCaptcha 生成验证码
func (*CaptchaService) GenerateCaptcha() *Captcha {
	// 生成uuid
	uuidV4 := uuid.New().String()
	uuidV4 = strings.ReplaceAll(uuidV4, "-", "")
	// 生成验证码
	data, err := captcha.New(280, 150)
	if err != nil {
		log.Println("生成图形验证码失败", err)
	}
	buf := new(bytes.Buffer)
	err = data.WriteImage(buf)
	if err != nil {
		log.Println("验证码写入缓存异常：", err)
		return nil
	}
	// 生成base64 格式的图片
	base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())
	capInstance := Captcha{
		Uuid: uuidV4,
		Text: data.Text,
		Img:  base64Str,
	}
	// 使用redis 缓存验证码信息
	key := fmt.Sprintf("captcha:%s", uuidV4)
	err = utils.SetCache(key, data.Text, 2*time.Minute)
	if err != nil {
		log.Println("缓存验证码信息异常：", err)
		return nil
	}
	return &capInstance
}

// VerifyCaptcha 校验验证码
func (*CaptchaService) VerifyCaptcha(instance *Captcha) bool {
	key := fmt.Sprintf("captcha:%s", instance.Uuid)
	uuidValue, err := utils.GetCache(key)
	if err != nil {
		log.Println(err)
		return false
	}

	rs := strings.EqualFold(instance.Text, uuidValue)
	if rs == true {
		// 清除验证码缓存
		_ = utils.DelCache(key)
	}
	return rs
}
