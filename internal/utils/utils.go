package utils

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const loggerKey = "zapLogger"

func GetLogger(c *gin.Context) *zap.Logger {
    l, exists := c.Get(loggerKey)
    if !exists {
        return zap.L()
    }
    return l.(*zap.Logger)
}