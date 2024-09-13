package main

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"testAvito/config"
	"testAvito/handlers"
	"testAvito/middleware"
	"testAvito/utils"

	httpSwagger "github.com/swaggo/http-swagger"
	_ "testAvito/docs" // Обратите внимание, что это подключение сгенерированной документации
)

// @title Tender API
// @version 1.0
// @description API для управления тендерами и предложениями
// @host localhost:8080
// @BasePath /api
func main() {
	config.LoadEnv()
	utils.InitDB()
	r := mux.NewRouter()

	// Добавление Swagger UI по пути /swagger/
	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Пути для тендеров
	apiRouter := r.PathPrefix("/api").Subrouter()
	middleware.JSONMiddleware(apiRouter)
	tenderRouter := apiRouter.PathPrefix("/tenders").Subrouter()
	bidsRouter := apiRouter.PathPrefix("/bids").Subrouter()

	// Проверяющая функция что связь с сервером установлена (возвращает 200 и ок)
	apiRouter.HandleFunc("/ping", handlers.PingHandler).Methods("GET")

	// Все ручки связанные с тендером
	tenderRouter.HandleFunc("", handlers.TenderShowHandler).Methods("GET")
	tenderRouter.HandleFunc("/new", handlers.CreateTenderHandler).Methods("POST")
	tenderRouter.HandleFunc("/{tenderId}/status", handlers.SetStatusTenderHandler).Methods("PUT")
	tenderRouter.HandleFunc("/{tenderId}/status", handlers.GetStatusTenderHandler).Methods("GET")
	tenderRouter.HandleFunc("/my", handlers.ShowTenderUserHandler).Methods("GET")
	tenderRouter.HandleFunc("/{tenderId}/edit", handlers.EditTenderHandler).Methods("PATCH")
	tenderRouter.HandleFunc("/{tenderId}/rollback/{version}", handlers.RollbackTenderHandler).Methods("PUT")

	// Все ручки связанные с предложениями
	bidsRouter.HandleFunc("/new", handlers.CreateBidHandler).Methods("POST")
	bidsRouter.HandleFunc("/{bidId}/submit_decision", handlers.SubmitBidDecisionHandler).Methods("PUT")
	bidsRouter.HandleFunc("/my", handlers.GetBidUserHandler).Methods("GET")
	bidsRouter.HandleFunc("/{tenderId}/list", handlers.GetBidByTenderIdHandler).Methods("GET")
	bidsRouter.HandleFunc("/{bidId}/status", handlers.SetStatusBidHandler).Methods("PUT")
	bidsRouter.HandleFunc("/{bidId}/status", handlers.GetStatusBidHandler).Methods("GET")
	bidsRouter.HandleFunc("/{bidId}/edit", handlers.EditBidHandler).Methods("PATCH")
	bidsRouter.HandleFunc("/{bidId}/rollback/{version}", handlers.RollbackBidHandler).Methods("PUT")
	bidsRouter.HandleFunc("/{tenderId}/reviews", handlers.GetBidReviewsHandler).Methods("GET")
	bidsRouter.HandleFunc("/{bidId}/feedback", handlers.SubmitReviewBidByTenderIdHandler).Methods("PUT")

	// Запуск сервера
	add := os.Getenv("SERVER_ADDRESS")
	log.Printf("Server listen and serve on port %s", add)
	log.Fatal(http.ListenAndServe(add, r))
}
