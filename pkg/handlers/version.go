package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

const VERSION = "v1.0.18"

// Version 处理版本信息请求
func Version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": VERSION})
}
