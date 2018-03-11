package controller

import (
	"context"
	"strings"

	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"

	"github.com/gin-gonic/gin"
)

// ErrorResult returned by controllers
type ErrorResult struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

type contextKey string

func getParam(c *gin.Context, name string) string {
	v := c.Param(name)
	return strings.Replace(v, "_slash_", "/", -1)
}

func getContext(c *gin.Context) context.Context {
	// get account ID from request URL parameter
	account := c.Param("account")
	// get request ID for logging form header
	requestID := c.GetHeader(codefresh.RequestID)
	// get authenticated entity from header
	authEntity := c.GetHeader(codefresh.AuthEntity)
	// prepare context
	ctx := context.WithValue(context.Background(), model.ContextKeyAccount, account)
	if requestID != "" {
		ctx = context.WithValue(ctx, model.ContextRequestID, requestID)
	}
	if authEntity != "" {
		ctx = context.WithValue(ctx, model.ContextAuthEntity, authEntity)
	}
	return ctx
}
