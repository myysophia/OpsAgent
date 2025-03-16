package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/myysophia/OpsAgent/pkg/utils"
	"go.uber.org/zap"
	"net/http"
)

// Claims JWT 声明结构
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// JWTAuth JWT 认证中间件
func JWTAuth() gin.HandlerFunc {
	logger := utils.GetLogger().Named("jwt")
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			utils.Error("缺少授权令牌")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization token"})
			return
		}

		// 移除 "Bearer " 前缀
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		claims := &Claims{}

		// 从全局变量中获取JWT密钥
		jwtKey, ok := utils.GetGlobalVar("jwtKey")
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			utils.Error("JWT 密钥未找到")
			return
		}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey.([]byte), nil
		})

		if err != nil {
			utils.Error("令牌解析失败", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			logger.Error("令牌解析失败", zap.Error(err))
			return
		}

		if !token.Valid {
			utils.Error("令牌无效")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token is not valid"})
			return
		}

		utils.Debug("令牌验证成功", zap.String("username", claims.Username))
		c.Set("username", claims.Username)
		c.Next()
	}
}
