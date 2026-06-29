package helpers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ResponseHandler menstandarkan bentuk respons JSON API (padanan ResponseHandler
// di core NodeAdmin): { status, message, data }.
type apiEnvelope struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

// JSONSuccess mengirim respons sukses standar.
func JSONSuccess(c *gin.Context, status int, message string, data interface{}) {
	c.JSON(status, apiEnvelope{Status: true, Message: message, Data: data})
}

// JSONError mengirim respons error standar (dipakai middleware ErrorHandler).
func JSONError(c *gin.Context, status int, message string, errs interface{}) {
	c.JSON(status, apiEnvelope{Status: false, Message: message, Errors: errs})
}

// OK = 200 sukses.
func OK(c *gin.Context, message string, data interface{}) {
	JSONSuccess(c, http.StatusOK, message, data)
}

// Created = 201 sukses.
func Created(c *gin.Context, message string, data interface{}) {
	JSONSuccess(c, http.StatusCreated, message, data)
}
