package controller

import (
	"context"
	"net/url"

	"github.com/codefresh-io/hermes/pkg/codefresh"
	"github.com/codefresh-io/hermes/pkg/model"

	"github.com/gin-gonic/gin"
	"github.com/newrelic/go-agent/_integrations/nrgin/v1"
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
	v, err := url.PathUnescape(v)
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
	// get NewRelic transaction from Gin context
	txn := nrgin.Transaction(c)
	// prepare context
	ctx := context.WithValue(context.Background(), model.ContextKeyAccount, account)
	if requestID != "" {
		ctx = context.WithValue(ctx, model.ContextRequestID, requestID)
	}
	if authEntity != "" {
		ctx = context.WithValue(ctx, model.ContextAuthEntity, authEntity)
	}
	if txn != nil {
		// add account to transaction
		if err := txn.AddAttribute("account-id", account); err != nil {
			log.WithError(err).Error("failed to add account-id to NewRelic transaction")
		}
		// add request id
		if err := txn.AddAttribute("request-id", requestID); err != nil {
			log.WithError(err).Error("failed to add request-id to NewRelic transaction")
		}
		// store transaction in context
		ctx = context.WithValue(ctx, model.ContextNewRelicTxn, txn)
	}
	return ctx
}
