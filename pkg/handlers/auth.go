package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/myysophia/OpsAgent/pkg/middleware"
	"github.com/myysophia/OpsAgent/pkg/utils"
	"go.uber.org/zap"
	"net/http"
	"time"
)

const (
	DEFAULT_USERNAME = "admin"
	DEFAULT_PASSWORD = "AqA-dqR2@GGJ,8fjz%=6%Ehrv"
)

// LoginRequest 登录请求结构
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 处理登录请求
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error("登录请求参数无效", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 使用默认账户验证
	if req.Username != DEFAULT_USERNAME || req.Password != DEFAULT_PASSWORD {
		utils.Warn("登录失败：用户名或密码错误",
			zap.String("username", req.Username))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// 创建 JWT token
	claims := &middleware.Claims{
		Username: req.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 从全局变量中获取JWT密钥
	jwtKey, ok := utils.GetGlobalVar("jwtKey")
	if !ok {
		utils.Error("JWT 密钥未找到")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	tokenString, err := token.SignedString(jwtKey.([]byte))
	if err != nil {
		utils.Error("生成令牌失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	utils.Info("登录成功", zap.String("username", req.Username))
	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"note":  "Default credentials: admin/novastar",
	})
}
