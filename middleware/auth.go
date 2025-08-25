package middleware

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type CustomClaims struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	DeptId   string `json:"deptId"`
	Scope    string `json:"scope"`
	jwt.StandardClaims
}

var issuer = os.Getenv("JWT_ISSUER")
var secret = os.Getenv("JWT_SECRET")

// URL 白名单
var excludePaths = [...]string{
	"/api/captcha/generate",
	"/api/user/login",
}

func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		currentPath := ctx.Request.URL.Path
		for _, path := range excludePaths {
			if match, _ := filepath.Match(path, currentPath); match {
				ctx.Next()
				return
			}
		}
		authorization := ctx.GetHeader("Authorization")
		if authorization == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": http.StatusUnauthorized,
				"msg":  "Authorization header is missing",
			})
			return
		}

		token, err := ValidateToken(authorization)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": http.StatusUnauthorized,
				"msg":  "Invalid Token",
			})
			return
		}
		// 设置用户信息
		ctx.Set("userId", token.UserID)
		ctx.Set("deptId", token.DeptId)
		ctx.Set("userName", token.Username)
		ctx.Next()
	}
}

// GenerateJWT 生成JWT
func GenerateJWT(userId, username, deptId string) (string, error) {
	// 设置过期时间
	expirationTime := time.Now().Add(1 * time.Hour).Unix()

	claims := CustomClaims{
		UserID:   userId,
		Username: username,
		DeptId:   deptId,
		Scope:    "read write",
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime,
			IssuedAt:  time.Now().Unix(),
			Issuer:    issuer,
			Subject:   secret,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		errInfo := fmt.Sprintf("生成Token失败：%v", err.Error())
		return "", errors.New(errInfo)
	}
	return tokenString, nil
}

// ValidateToken 验证JWT
func ValidateToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			errInfo := fmt.Sprintf("unexpected signing method: %v", token.Header["alg"])
			return nil, errors.New(errInfo)
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		if claims.Issuer != issuer {
			return nil, errors.New("无效的签发者")
		}
		return claims, nil
	}
	return nil, errors.New("无效的token")
}
