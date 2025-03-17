package api

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/myysophia/OpsAgent/pkg/handlers"
	"github.com/myysophia/OpsAgent/pkg/middleware"
	"github.com/myysophia/OpsAgent/pkg/utils"
	"go.uber.org/zap"
)

// Router 设置API路由
func Router() *gin.Engine {
	// 获取日志记录器
	logger := utils.GetLogger()

	// 设置gin模式
	gin.SetMode(gin.DebugMode)

	// 创建gin引擎
	r := gin.New()

	// 使用自定义中间件
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())

	// 配置CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-OpenAI-Key", "X-API-Key", "X-Requested-With", "api-key"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
		AllowWildcard:    true,
		AllowWebSockets:  true,
	}))

	// 添加请求日志中间件
	r.Use(func(c *gin.Context) {
		// 请求开始时间
		startTime := time.Now()

		// 读取请求体
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = c.GetRawData()
			// 将请求体放回，以便后续中间件使用
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		logger.Debug("收到请求",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("body", string(bodyBytes)),
		)

		// 处理请求
		c.Next()

		// 请求结束时间
		duration := time.Since(startTime)

		logger.Debug("请求处理完成",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
		)
	})

	// 全局处理OPTIONS请求
	r.OPTIONS("/*path", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	r.POST("/login", handlers.Login)

	// 注册API路由
	api := r.Group("/api")
	{
		// 版本信息
		api.GET("/version", handlers.Version)

		// 需要认证的路由
		auth := api.Group("")
		auth.Use(middleware.JWTAuth())
		{
			// 执行命令
			auth.POST("/execute", handlers.Execute)

			// 诊断
			auth.POST("/diagnose", handlers.Diagnose)

			// 分析
			auth.POST("/analyze", handlers.Analyze)

			// 性能统计
			auth.GET("/perf/stats", handlers.PerfStats)
			auth.POST("/perf/reset", handlers.ResetPerfStats)
		}
	}

	return r
}
