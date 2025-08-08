package api

import (
	"github.com/beganov/L0/internal/storage"
	"github.com/gin-gonic/gin"
)

func RouteRegister(router *gin.Engine) {
	server := NewServer()
	//router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler)) // Маршрут для сваггера
	router.GET("/order/:order_uid", server.getOrder)
}

func SetupRouter() *gin.Engine {
	router := gin.Default()
	return router
}

func NewServer() *httpServer {
	store := storage.NewStorage()
	return &httpServer{store: store}
}

func (hs *httpServer) getOrder(c *gin.Context) {
}

type httpServer struct {
	store *storage.Storage // хранилище для управления комнатами
}
