package controller

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// ErrorResult returned by controllers
type ErrorResult struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

func getParam(c *gin.Context, name string) string {
	v := c.Param(name)
	return strings.Replace(v, "_slash_", "/", -1)
}
