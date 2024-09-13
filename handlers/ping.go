package handlers

import (
	"log"
	"net/http"
)

// Базовый Пинговый хэндлер
// PingHandler проверяет работоспособность сервера.
// @Summary Проверка состояния сервера
// @Description Возвращает "ok", если сервер работает.
// @Tags Health
// @Produce  plain
// @Success 200 {string} string "ok"
// @Failure 500 {string} string "Ошибка сервера"
// @Router /ping [get]
func PingHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("ok")); err != nil {
		log.Fatal(err.Error())
	}
}
