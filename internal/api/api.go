package api

import (
<<<<<<< HEAD
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
=======
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/beganov/L0/internal/cache"
	"github.com/beganov/L0/internal/config"
	"github.com/beganov/L0/internal/database"
	"github.com/beganov/L0/internal/logger"
	"github.com/beganov/L0/internal/metrics"

	_ "github.com/beganov/L0/docs"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
)

func SetupRouter(cache *cache.OrderCache, db *pgxpool.Pool) http.Handler {
	r := mux.NewRouter()
	handler := NewOrderHandler(cache, db)

	r.HandleFunc("/order/{id}", handler.GetOrder).Methods("GET")
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)
	r.Handle("/metrics", promhttp.Handler())

	return r
}

type OrderHandler struct {
	cache       *cache.OrderCache
	db          *pgxpool.Pool
	httpTimeOut time.Duration
}

func NewOrderHandler(cache *cache.OrderCache, db *pgxpool.Pool) *OrderHandler {
	return &OrderHandler{
		cache:       cache,
		db:          db,
		httpTimeOut: config.HttpTimeOut,
	}
}

// GetOrder return order by id
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	timer := prometheus.NewTimer(metrics.HttpDuration)
	defer timer.ObserveDuration()

	metrics.HttpRequestsTotal.Inc()

	orderID := mux.Vars(r)["id"]

	// check cache
	if order, ok := h.cache.Get(orderID); ok {
		writeJSON(w, order)
		return
	}

	// get from db with timeout
	dbCtx, cancel := context.WithTimeout(r.Context(), h.httpTimeOut)
	defer cancel()

	order, err := database.GetOrderFromDB(dbCtx, h.db, orderID, config.SelectTimeOut)
	if err != nil {
		metrics.HttpErrorsTotal.Inc()
		logger.Error(err, "order not found")
		w.Header().Set("Access-Control-Allow-Origin", "*") // allow browser requests
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	// save in cache
	h.cache.Set(orderID, order)

	writeJSON(w, order)
}

// writeJSON send json to client
func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Error(err, "cannot encode json")
	}
>>>>>>> 6968df1 (Add all commit)
}
