package controller

import (
	"context"
	"net/url"

	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
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
	v, err := url.QueryUnescape(v)
	if err != nil {
		log.WithFields(log.Fields{
			"name":  name,
			"value": v,
		}).WithError(err).Error("failed to URL decode value")
	}
	return v
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
