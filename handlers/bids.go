package handlers

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"testAvito/models"
	"testAvito/utils"

	_ "github.com/swaggo/http-swagger"
	_ "testAvito/docs"
)

// CreateBidHandler создает новое предложение (Bid).
// @Summary Создание нового предложения
// @Description Создает новое предложение для тендера, проверяет условия и права автора предложения.
// @Tags Bids
// @Accept  json
// @Produce  json
// @Param bid body models.Bid true "Информация о предложении"
// @Success 200 {object} models.Bid "Успешное создание предложения"
// @Failure 400 {string} string "Неверно введенное предложение"
// @Failure 404 {string} string "Тендер или пользователь не найдены"
// @Failure 409 {string} string "Организация не может отправить предложение на свои тендеры"
// @Router /bids/new [post]
func CreateBidHandler(w http.ResponseWriter, r *http.Request) {
	var bid models.Bid
	// Декодируем входящий json
	if err := json.NewDecoder(r.Body).Decode(&bid); err != nil {
		http.Error(w, "Неверно введенное предложение.", http.StatusBadRequest)
		return
	}

	// Проверка на существование тендера
	var tender models.Tender
	if err := utils.DB.First(&tender, bid.TenderID).Error; err != nil {
		http.Error(w, "Тендер не найден.", http.StatusNotFound)
		return
	}
	if tender.Status != models.PUBLISHED {
		http.Error(w, "Тендер не опубликован.", http.StatusNotFound)
		return
	}
	switch bid.AuthorType {
	case models.USER:
		var employee models.Employee
		if err := utils.DB.First(&employee, bid.AuthorID).Error; err != nil {
			http.Error(w, "Пользователь не найден.", http.StatusNotFound)
			return
		}
		var orgResp models.OrganizationResponsible
		if err := utils.DB.Where("organization_id = ? AND user_id = ?", tender.OrganizationID, employee.ID).First(&orgResp).Error; err == nil {
			http.Error(w, "Пользователь не может подать предложение на тендер в своей организации.", http.StatusBadRequest)
			return
		}

	case models.ORGANIZATION:
		var org models.Organization
		if err := utils.DB.First(&org, bid.AuthorID).Error; err != nil {
			http.Error(w, "Организация не найдена.", http.StatusBadRequest)
			return
		}
		if tender.OrganizationID == bid.AuthorID {
			http.Error(w, "Организация не может отправлять себе же предложения на свои тендеры.", http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "Неверно введенный тип автора. Тип автора должен быть USER или ORGANIZATION.", http.StatusBadRequest)
		return

	}

	// Установление статуса создания предложения
	bid.Status = models.CREATEDBid

	// Создание в бд предложения
	if err := utils.DB.Create(&bid).Error; err != nil {
		http.Error(w, "Ошибка создания предложения.", http.StatusNotFound)
		return
	}
	saveBidsVersion(bid)
	// Возвращаем все в нормальный вид (unmarshal)
	utils.JSONFormat(w, r, bid)

}

// GetBidUserHandler получает список предложений пользователя.
// @Summary Получение предложений пользователя
// @Description Возвращает список предложений, созданных пользователем с указанным именем (username).
// @Tags Bids
// @Accept  json
// @Produce  json
// @Param username query string true "Имя пользователя для поиска предложений"
// @Success 200 {array} models.Bid "Список предложений пользователя"
// @Failure 400 {string} string "Имя пользователя пустое"
// @Failure 404 {string} string "Пользователь не найден"
// @Failure 500 {string} string "Ошибка нахождения предложений"
// @Router /bids/my [get]
func GetBidUserHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Имя пользователя пустое", http.StatusBadRequest)
		return
	}
	var employee models.Employee
	if err := utils.DB.Where("username = ?", username).First(&employee).Error; err != nil {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	var bids []models.Bid
	if err := utils.DB.Where("author_id = ? AND author_type = ?", employee.ID, models.USER).Find(&bids).Error; err != nil {
		http.Error(w, "Ошибка нахождения предложения", http.StatusInternalServerError)
		return
	}

	utils.JSONFormat(w, r, bids)

}

// GetBidByTenderIdHandler получает список предложений для конкретного тендера.
// @Summary Получение предложений по TenderID
// @Description Возвращает список предложений для указанного тендера, если пользователь имеет право на просмотр.
// @Tags Bids
// @Accept  json
// @Produce  json
// @Param tenderId path int true "ID тендера"
// @Param username query string true "Имя пользователя для проверки прав доступа"
// @Success 200 {array} models.Bid "Список предложений"
// @Failure 400 {string} string "Неверный тендер ID или пустое имя пользователя"
// @Failure 403 {string} string "Нет прав на получение списка предложений"
// @Failure 404 {string} string "Тендер или пользователь не найдены"
// @Failure 500 {string} string "Ошибка получения предложений"
// @Router /bids/{tenderId}/list [get]
func GetBidByTenderIdHandler(w http.ResponseWriter, r *http.Request) {
	// Ищем в URL тендер_айди
	params := mux.Vars(r)
	tenderID, err := strconv.Atoi(params["tenderId"])
	if err != nil {
		http.Error(w, "Неверный тендер ID", http.StatusBadRequest)
		return
	}
	var tender models.Tender
	if err := utils.DB.First(&tender, tenderID).Error; err != nil {
		http.Error(w, "Ошибка нахождения тендера", http.StatusInternalServerError)
		return
	}
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Имя пользователя пустое", http.StatusBadRequest)
		return
	}
	var employee models.Employee
	if err := utils.DB.Where("username = ?", username).First(&employee).Error; err != nil {
		http.Error(w, "Ошибка нахождения пользователя", http.StatusInternalServerError)
		return
	}

	var organizationResponsible models.OrganizationResponsible
	if err := utils.DB.Where("organization_id = ? AND user_id = ?", tender.OrganizationID, employee.ID).First(&organizationResponsible).Error; err != nil {
		http.Error(w, "У вас нет прав получить список предложений для тендера.", http.StatusForbidden)
		return
	}

	// Создаем массив из предложений
	var bids []models.Bid
	if err := utils.DB.Where("tender_id = ?", tenderID).Find(&bids).Error; err != nil {
		http.Error(w, "Ошибка нахождения предложения", http.StatusInternalServerError)
		return
	}

	// Возвращаем все в нормальный вид (unmarshal) и выводим массив предложений
	utils.JSONFormat(w, r, bids)
}

// GetStatusBidHandler возвращает статус предложения (bid) по его ID.
// @Summary Получение статуса предложения
// @Description Возвращает статус предложения, если пользователь имеет права на просмотр статуса.
// @Tags Bids
// @Accept  json
// @Produce  plain
// @Param bidId path int true "ID предложения"
// @Param username query string true "Имя пользователя, запрашивающего статус"
// @Success 200 {string} string "Статус предложения"
// @Failure 400 {string} string "Неверный ID предложения или никнейм не введен"
// @Failure 403 {string} string "Нет прав для просмотра статуса предложения"
// @Failure 404 {string} string "Предложение или пользователь не найдены"
// @Failure 500 {string} string "Ошибка сервера"
// @Router /bids/{bidId}/status [get]
func GetStatusBidHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	bidId, err := strconv.Atoi(params["bidId"])
	if err != nil {
		http.Error(w, "Неверный ID предложения", http.StatusBadRequest)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Никнейм пользователя, кто хочет проверить статус предложения не введено", http.StatusBadRequest)
	}
	var employee models.Employee
	if err := utils.DB.Where("username = ?", username).First(&employee).Error; err != nil {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	var bid models.Bid
	if err = utils.DB.First(&bid, bidId).Error; err != nil {
		http.Error(w, "Предложение не найдено", http.StatusNotFound)
		return
	}

	switch bid.AuthorType {
	case models.USER:
		if bid.AuthorID != employee.ID {
			http.Error(w, "Только автор предложения может смотреть на статус предложения", http.StatusForbidden)
			return
		}

	case models.ORGANIZATION:
		var orgResp models.OrganizationResponsible
		if err := utils.DB.Where("organization_id = ? AND user_id = ?", bid.AuthorID, employee.ID).First(&orgResp).Error; err != nil {
			http.Error(w, "Только члены организации могут смотреть на статус предложения", http.StatusForbidden)
			return
		}
	default:
		http.Error(w, "Неверный тип автора предложения", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(bid.Status))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// EditBidHandler редактирует предложение (Bid) по его ID.
// @Summary Редактирование предложения
// @Description Изменяет предложение по его ID, если автором является пользователь или член организации.
// @Tags Bids
// @Accept  json
// @Produce  json
// @Param bidId path int true "ID предложения"
// @Param username query string true "Имя пользователя"
// @Param bid body object true "Данные для обновления предложения (name, description)"
// @Success 200 {object} models.Bid "Обновленное предложение"
// @Failure 400 {string} string "Неверный ID предложения, имя пользователя или данные предложения"
// @Failure 403 {string} string "Нет прав для редактирования предложения"
// @Failure 404 {string} string "Предложение или пользователь не найдены"
// @Failure 500 {string} string "Ошибка сохранения предложения"
// @Router /bids/{bidId}/edit [patch]
func EditBidHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем bidId и username из URL
	params := mux.Vars(r)
	bidId, err := strconv.Atoi(params["bidId"])
	if err != nil {
		http.Error(w, "Неверный ID предложения", http.StatusBadRequest)
		return
	}

	// Получаем username из строки запроса
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Имя пользователя пустое", http.StatusBadRequest)
		return
	}

	// Проверяем, существует ли пользователь с переданным username
	var employee models.Employee
	if err := utils.DB.Where("username = ?", username).First(&employee).Error; err != nil {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	// Находим предложение по bidId
	var bid models.Bid
	if err := utils.DB.First(&bid, bidId).Error; err != nil {
		http.Error(w, "Предложение не найдено", http.StatusNotFound)
		return
	}

	switch bid.AuthorType {
	case models.USER:
		if bid.AuthorID != employee.ID {
			http.Error(w, "Только автор предложения может изменять его", http.StatusForbidden)
			return
		}

	case models.ORGANIZATION:
		var orgResp models.OrganizationResponsible
		if err := utils.DB.Where("organization_id = ? AND user_id = ?", bid.AuthorID, employee.ID).First(&orgResp).Error; err != nil {
			http.Error(w, "Только члены организации могут изменять предложения", http.StatusForbidden)
			return
		}
	default:
		http.Error(w, "Неверный тип автора предложения", http.StatusBadRequest)
		return
	}

	// Проверяем текущий статус предложения
	if bid.Status == models.CANCELED {
		http.Error(w, "Предложение отменено", http.StatusBadRequest)
		return
	}
	if bid.Status == models.PUBLISHEDBid {
		http.Error(w, "Предложение уже утверждено, изменения невозможны", http.StatusBadRequest)
		return
	}

	// Декодируем обновленные данные из тела запроса
	var updatedBids map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updatedBids); err != nil {
		http.Error(w, "Неверно введенное предложение", http.StatusBadRequest)
		return
	}

	// Обновляем данные предложения, если они были переданы
	if name, ok := updatedBids["name"].(string); ok {
		bid.Name = name
	}
	if description, ok := updatedBids["description"].(string); ok {
		bid.Description = description
	}

	// Увеличиваем версию предложения
	bid.Version++

	// Сохраняем изменения в базе данных
	if err := utils.DB.Save(&bid).Error; err != nil {
		http.Error(w, "Ошибка сохранения предложения", http.StatusInternalServerError)
		return
	}

	// Сохраняем версию предложения для истории
	saveBidsVersion(bid)

	// Возвращаем обновленное предложение в формате JSON
	utils.JSONFormat(w, r, bid)
}

// RollbackBidHandler откатывает предложение (Bid) к указанной версии.
// @Summary Откат предложения к версии
// @Description Откатывает предложение к указанной версии, если автором является пользователь или член организации, и предложение не утверждено или отменено.
// @Tags Bids
// @Accept  json
// @Produce  json
// @Param bidId path int true "ID предложения"
// @Param version path int true "Версия, к которой откатывается предложение"
// @Param username query string true "Имя пользователя, инициирующего откат"
// @Success 200 {object} models.Bid "Успешное откатывание предложения"
// @Failure 400 {string} string "Неверный ID предложения, версия или имя пользователя"
// @Failure 403 {string} string "Нет прав для откатывания версии предложения"
// @Failure 404 {string} string "Предложение, пользователь или версия не найдены"
// @Failure 500 {string} string "Ошибка обновления предложения"
// @Router /bids/{bidId}/rollback/{version} [put]
func RollbackBidHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем bidId и version из URL
	params := mux.Vars(r)
	bidID, err := strconv.Atoi(params["bidId"])
	if err != nil {
		http.Error(w, "Неверное ID предложения", http.StatusBadRequest)
		return
	}

	version, err := strconv.Atoi(params["version"])
	if err != nil {
		http.Error(w, "Неверная версия", http.StatusBadRequest)
		return
	}

	// Получаем username из строки запроса
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Имя пользователя пустое", http.StatusBadRequest)
		return
	}

	// Проверяем, существует ли пользователь с переданным username
	var employee models.Employee
	if err := utils.DB.Where("username = ?", username).First(&employee).Error; err != nil {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	// Находим предложение по bidId
	var bid models.Bid
	if err := utils.DB.First(&bid, bidID).Error; err != nil {
		http.Error(w, "Предложение не найдено", http.StatusNotFound)
		return
	}

	switch bid.AuthorType {
	case models.USER:
		if bid.AuthorID != employee.ID {
			http.Error(w, "Только автор предложения может изменять его версию", http.StatusForbidden)
			return
		}

	case models.ORGANIZATION:
		var orgResp models.OrganizationResponsible
		if err := utils.DB.Where("organization_id = ? AND user_id = ?", bid.AuthorID, employee.ID).First(&orgResp).Error; err != nil {
			http.Error(w, "Только члены организации могут изменять версию предложения", http.StatusForbidden)
			return
		}

	default:
		http.Error(w, "Неверный тип автора предложения", http.StatusBadRequest)
		return
	}

	// Проверяем статус предложения
	if bid.Status == models.CANCELED {
		http.Error(w, "Предложение отменено, дальнейшее взаимодействие с ним невозможно.", http.StatusBadRequest)
		return
	}
	if bid.Status == models.PUBLISHEDBid {
		http.Error(w, "Предложение уже утверждено, дальнейшее взаимодействие с ним невозможно.", http.StatusBadRequest)
		return
	}

	// Находим указанную версию предложения
	var bidVersion models.BidVersion
	if err := utils.DB.Where("bid_id = ? AND version = ?", bidID, version).First(&bidVersion).Error; err != nil {
		http.Error(w, "Введенная версия предложения не найдена", http.StatusNotFound)
		return
	}

	// Откат предложения к указанной версии
	bid.Name = bidVersion.Name
	bid.Description = bidVersion.Description
	bid.Status = bidVersion.Status
	bid.Version++ // Увеличиваем версию предложения

	// Сохраняем изменения в базе данных
	if err := utils.DB.Save(&bid).Error; err != nil {
		http.Error(w, "Ошибка обновления предложения", http.StatusInternalServerError)
		return
	}
	saveBidsVersion(bid)
	// Возвращаем обновленное предложение в формате JSON
	utils.JSONFormat(w, r, bid)
}

// SubmitReviewBidByTenderIdHandler добавляет отзыв по предложению (Bid) по его ID.
// @Summary Добавление отзыва по предложению
// @Description Добавляет отзыв по предложению, если пользователь имеет право принимать решение. Проверяет, было ли уже добавлено решение по данному предложению.
// @Tags Bids
// @Accept  json
// @Produce  json
// @Param bidId path int true "ID предложения"
// @Param bidFeedback query string true "Решение по предложению (одобрено или отклонено)"
// @Param username query string true "Имя пользователя, принимающего решение"
// @Success 200 {object} models.BidFeedback "Отзыв успешно сохранен"
// @Failure 400 {string} string "Неверный ID предложения или пустое имя пользователя или отзыв"
// @Failure 403 {string} string "Нет прав для принятия решения по предложению"
// @Failure 404 {string} string "Пользователь, предложение или тендер не найдены"
// @Failure 409 {string} string "Решение по данному предложению уже было принято"
// @Failure 500 {string} string "Ошибка сохранения решения"
// @Router /bids/{bidId}/feedback [put]
func SubmitReviewBidByTenderIdHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	bidId, err := strconv.Atoi(params["bidId"])
	if err != nil {
		http.Error(w, "Неверный ID предложения", http.StatusBadRequest)
		return
	}

	bidFeedback := r.URL.Query().Get("bidFeedback")
	username := r.URL.Query().Get("username")

	if bidFeedback == "" {
		http.Error(w, "Необходимо ввести решение по предложению", http.StatusBadRequest)
		return
	}

	if username == "" {
		http.Error(w, "Имя пользователя пустое", http.StatusBadRequest)
		return
	}

	var employee models.Employee
	if err := utils.DB.Where("username = ?", username).First(&employee).Error; err != nil {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	var bid models.Bid
	if err := utils.DB.First(&bid, bidId).Error; err != nil {
		http.Error(w, "Предложение не найдено", http.StatusNotFound)
		return
	}

	var tender models.Tender
	if err := utils.DB.First(&tender, bid.TenderID).Error; err != nil {
		http.Error(w, "Тендер не найден", http.StatusNotFound)
		return
	}

	var organizationResponsible models.OrganizationResponsible
	if err := utils.DB.Where("organization_id = ? AND user_id = ?", tender.OrganizationID, employee.ID).First(&organizationResponsible).Error; err != nil {
		http.Error(w, "Вы не можете принимать решение по данному предложению", http.StatusForbidden)
		return
	}

	var existingFeedback models.BidFeedback
	if err := utils.DB.Where("bid_id = ? AND username = ?", bid.ID, username).First(&existingFeedback).Error; err == nil {
		http.Error(w, "Вы уже приняли решение по данному предложению", http.StatusConflict)
		return
	}
	newFeedback := models.BidFeedback{
		BidID:    bid.ID,
		Username: username,
		Feedback: bidFeedback,
	}

	if err := utils.DB.Create(&newFeedback).Error; err != nil {
		http.Error(w, "Ошибка сохранения решения", http.StatusInternalServerError)
		return
	}
	utils.JSONFormat(w, r, newFeedback)
}

// SubmitBidDecisionHandler добавляет решение по предложению (Bid) по его ID.
// @Summary Добавление решения по предложению
// @Description Добавляет решение ("Approved" или "Rejected") по предложению на основании прав пользователя. Проверяет наличие кворума для публикации предложения.
// @Tags Bids
// @Accept  json
// @Produce  json
// @Param bidId path int true "ID предложения"
// @Param decision query string true "Решение по предложению ('Approved' или 'Rejected')"
// @Param username query string true "Имя пользователя, принимающего решение"
// @Success 200 {object} models.Bid "Обновленное предложение"
// @Failure 400 {string} string "Неверное решение, ID предложения или имя пользователя"
// @Failure 403 {string} string "Нет прав для принятия решения по предложению"
// @Failure 404 {string} string "Пользователь, предложение или тендер не найдены"
// @Failure 409 {string} string "Решение по данному предложению уже было принято"
// @Failure 500 {string} string "Ошибка сохранения решения или публикации предложения"
// @Router /bids/{bidId}/submit_decision [put]
func SubmitBidDecisionHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем bidId из URL
	params := mux.Vars(r)
	bidId, err := strconv.Atoi(params["bidId"])
	if err != nil {
		http.Error(w, "Неверный ID предложения", http.StatusBadRequest)
		return
	}

	// Получаем решение и username из строки запроса
	decision := r.URL.Query().Get("decision")
	username := r.URL.Query().Get("username")

	if decision == "" || (decision != "Approved" && decision != "Rejected") {
		http.Error(w, "Неверное решение. Решение должно быть 'Approved' или 'Rejected'.", http.StatusBadRequest)
		return
	}

	if username == "" {
		http.Error(w, "Имя пользователя пустое", http.StatusBadRequest)
		return
	}

	// Проверяем, существует ли пользователь с переданным username
	var employee models.Employee
	if err := utils.DB.Where("username = ?", username).First(&employee).Error; err != nil {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	// Находим предложение по bidId
	var bid models.Bid
	if err := utils.DB.First(&bid, bidId).Error; err != nil {
		http.Error(w, "Предложение не найдено", http.StatusNotFound)
		return
	}
	if bid.Status == models.CANCELED {
		http.Error(w, "Предложение отменено", http.StatusBadRequest)
		return
	}
	if bid.Status == models.PUBLISHEDBid {
		http.Error(w, "Предложение уже утверждено, изменения невозможны", http.StatusBadRequest)
		return
	}

	// Находим тендер, связанный с предложением
	var tender models.Tender
	if err := utils.DB.First(&tender, bid.TenderID).Error; err != nil {
		http.Error(w, "Тендер не найден", http.StatusNotFound)
		return
	}
	if tender.Status == models.CLOSED {
		http.Error(w, "Тендер был закрыт, нельзя добавить предложение.", http.StatusBadRequest)
		bid.Status = models.CANCELED
		if err := utils.DB.Save(&bid).Error; err != nil {
			http.Error(w, "Ошибка обновления предложения", http.StatusInternalServerError)
			return
		}
		return
	}

	// Проверяем, что пользователь является ответственным за организацию тендера
	var organizationResponsible models.OrganizationResponsible
	if err := utils.DB.Where("organization_id = ? AND user_id = ?", tender.OrganizationID, employee.ID).First(&organizationResponsible).Error; err != nil {
		http.Error(w, "Вы не можете принимать решение по данному предложению", http.StatusForbidden)
		return
	}

	// Проверяем, что ответственный уже не голосовал за это предложение
	var existingDecision models.BidDecision
	if err := utils.DB.Where("bid_id = ? AND responsible_id = ?", bid.ID, employee.ID).First(&existingDecision).Error; err == nil {
		http.Error(w, "Вы уже приняли решение по данному предложению", http.StatusConflict)
		return
	}

	// Добавляем решение ответственного
	newDecision := models.BidDecision{
		BidID:         bid.ID,
		ResponsibleID: employee.ID,
		Decision:      decision,
	}
	if err := utils.DB.Create(&newDecision).Error; err != nil {
		http.Error(w, "Ошибка сохранения решения", http.StatusInternalServerError)
		return
	}

	// Проверка на наличие отклонений
	if decision == "Rejected" {
		bid.Status = models.CANCELED
		bid.Version++
		if err := utils.DB.Save(&bid).Error; err != nil {
			http.Error(w, "Ошибка отмены предложения", http.StatusInternalServerError)
			return
		}
		saveBidsVersion(bid)
		// Возвращаем обновленное предложение в формате JSON
		utils.JSONFormat(w, r, bid)
		return
	}

	// Подсчитываем количество ответственных за организацию
	var responsibleCount int64
	utils.DB.Model(&models.OrganizationResponsible{}).Where("organization_id = ?", tender.OrganizationID).Count(&responsibleCount)

	// Кворум = min(3, количество ответственных за организацию)
	quorum := int64(3)
	if responsibleCount < 3 {
		quorum = responsibleCount
	}

	// Подсчитываем количество утверждений
	var approvedCount int64
	utils.DB.Model(&models.BidDecision{}).Where("bid_id = ? AND decision = 'Approved'", bid.ID).Count(&approvedCount)

	// Если утверждений больше или равно кворуму, предложение публикуется
	if approvedCount >= quorum {
		bid.Status = models.PUBLISHEDBid
		bid.Version++
		if err := utils.DB.Save(&bid).Error; err != nil {
			http.Error(w, "Ошибка публикации предложения", http.StatusInternalServerError)
			return
		}
		saveBidsVersion(bid)

		tender.Status = models.CLOSED
		tender.Version++
		if err := utils.DB.Save(&tender).Error; err != nil {
			http.Error(w, "Ошибка закрытия тендера", http.StatusInternalServerError)
			return
		}
		saveTenderVersion(tender)

		var bids []models.Bid
		if err := utils.DB.Where("tender_id = ?", tender.ID).Find(&bids).Error; err != nil {
			http.Error(w, "Ошибка c поиском предложения по тендер айди", http.StatusInternalServerError)
			return
		}
		if len(bids) == 0 {
			http.Error(w, "Предложения не найдены для этого тендера.", http.StatusNotFound)
			return
		}
		for _, bid := range bids {
			if bid.Status != models.CANCELED && bid.Status != models.PUBLISHEDBid {
				bid.Status = models.CANCELED
				bid.Version++
				saveBidsVersion(bid)
				if err := utils.DB.Save(&bid).Error; err != nil {
					http.Error(w, "Ошибка закрытия предложения", http.StatusInternalServerError)
					return
				}
			}
		}
	}

	// Возвращаем обновленное предложение в формате JSON
	utils.JSONFormat(w, r, bid)
}

// SetStatusBidHandler устанавливает статус предложения (Bid) по его ID.
// @Summary Установка статуса предложения
// @Description Изменяет статус предложения на основании прав автора. Статус может быть изменен на 'CANCELED', но не на 'PUBLISHED' или 'CREATED', так как эти статусы устанавливаются автоматически.
// @Tags Bids
// @Accept  json
// @Produce  json
// @Param bidId path int true "ID предложения"
// @Param username query string true "Имя пользователя, изменяющего статус"
// @Param status query string true "Новый статус предложения ('CANCELED')"
// @Success 200 {object} models.Bid "Обновленное предложение"
// @Failure 400 {string} string "Неверный статус, ID предложения или имя пользователя"
// @Failure 403 {string} string "Нет прав для изменения статуса предложения"
// @Failure 404 {string} string "Предложение, тендер или пользователь не найдены"
// @Failure 500 {string} string "Ошибка обновления статуса"
// @Router /bids/{bidId}/status [put]
func SetStatusBidHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	bidId := params["bidId"]
	username := r.URL.Query().Get("username")
	status := r.URL.Query().Get("status")

	var bid models.Bid
	if err := utils.DB.First(&bid, bidId).Error; err != nil {
		http.Error(w, "Предложение не найдено", http.StatusNotFound)
		return
	}

	var tender models.Tender
	if err := utils.DB.First(&tender, bid.TenderID).Error; err != nil {
		http.Error(w, "Тендер не найден", http.StatusNotFound)
		return
	}
	var employee models.Employee
	if err := utils.DB.Where("username = ?", username).First(&employee).Error; err != nil {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	switch bid.AuthorType {
	case models.USER:
		if bid.AuthorID != employee.ID {
			http.Error(w, "Только автор предложения может изменять его статус", http.StatusForbidden)
			return
		}

	case models.ORGANIZATION:
		var orgResp models.OrganizationResponsible
		if err := utils.DB.Where("organization_id = ? AND user_id = ?", bid.AuthorID, employee.ID).First(&orgResp).Error; err != nil {
			http.Error(w, "Только члены организации могут изменять статус предложения", http.StatusForbidden)
			return
		}

	default:
		http.Error(w, "Неверный тип автора предложения", http.StatusBadRequest)
		return
	}

	if bid.Status == models.CANCELED {
		http.Error(w, "Предложение отменено, дальнейшее взаимодействие с ним невозможно.", http.StatusBadRequest)
		return
	}

	if bid.Status == models.PUBLISHEDBid {
		http.Error(w, "Предложение было принято, дальнейшее взаимодействие с ним невозможно.", http.StatusBadRequest)
		return
	}

	tmpBidStatus := bid.Status

	switch models.BidStatus(status) {
	case models.CANCELED:
		bid.Status = models.BidStatus(status)
	case models.PUBLISHEDBid:
		http.Error(w, "Статус PUBLISHED достигается решением Кворума, выбери другой статус (CANCELED)", http.StatusBadRequest)
		return
	case models.CREATEDBid:
		http.Error(w, "Статус CREATED достигается при инициализации предложения, выбери другой статус (CANCELED)", http.StatusBadRequest)
		return
	default:
		http.Error(w, "Неверно введенный статус. Статус должен быть CANCELED, PUBLISHED, CREATED", http.StatusBadRequest)
		return
	}

	if tmpBidStatus != bid.Status {
		bid.Version++
	}

	if err := utils.DB.Save(&bid).Error; err != nil {
		http.Error(w, "Ошибка обновления статуса предложения", http.StatusInternalServerError)
		return
	}
	saveBidsVersion(bid)

	utils.JSONFormat(w, r, bid)
}

// GetBidReviewsHandler получает отзывы по предложениям пользователя для конкретного тендера.
// @Summary Получение отзывов по предложениям пользователя
// @Description Возвращает отзывы по предложениям автора (authorUsername), связанным с тендером, если пользователь-запросчик (requesterUsername) имеет права доступа.
// @Tags Bids
// @Accept  json
// @Produce  json
// @Param tenderId path int true "ID тендера"
// @Param authorUsername query string true "Имя пользователя, автора предложений"
// @Param requesterUsername query string true "Имя пользователя, запрашивающего данные"
// @Success 200 {array} models.BidFeedback "Список отзывов по предложениям"
// @Failure 400 {string} string "Неверный ID тендера или отсутствует authorUsername/requesterUsername"
// @Failure 403 {string} string "Нет доступа к просмотру обратной связи"
// @Failure 404 {string} string "Тендер или пользователь не найден, или нет предложений"
// @Failure 500 {string} string "Ошибка загрузки данных"
// @Router /bids/{tenderId}/reviews [get]
func GetBidReviewsHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем tenderId из URL
	params := mux.Vars(r)
	tenderId, err := strconv.Atoi(params["tenderId"])
	if err != nil {
		http.Error(w, "Неверный ID тендера", http.StatusBadRequest)
		return
	}

	// Получаем authorUsername и requesterUsername из строки запроса
	authorUsername := r.URL.Query().Get("authorUsername")
	requesterUsername := r.URL.Query().Get("requesterUsername")

	if authorUsername == "" || requesterUsername == "" {
		http.Error(w, "Необходимы authorUsername и requesterUsername", http.StatusBadRequest)
		return
	}

	// Проверяем, существует ли пользователь-запросчик (requesterUsername)
	var requester models.Employee
	if err := utils.DB.Where("username = ?", requesterUsername).First(&requester).Error; err != nil {
		http.Error(w, "Пользователь-запросчик не найден", http.StatusNotFound)
		return
	}

	// Проверяем, что requester является ответственным за организацию, связанную с тендером
	var tender models.Tender
	if err := utils.DB.First(&tender, tenderId).Error; err != nil {
		http.Error(w, "Тендер не найден", http.StatusNotFound)
		return
	}

	var organizationResponsible models.OrganizationResponsible
	if err := utils.DB.Where("organization_id = ? AND user_id = ?", tender.OrganizationID, requester.ID).First(&organizationResponsible).Error; err != nil {
		http.Error(w, "У вас нет доступа к просмотру обратной связи по данному предложению", http.StatusForbidden)
		return
	}

	var employee models.Employee
	if err := utils.DB.Where("username = ?", authorUsername).First(&employee).Error; err != nil {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	// Находим все предложения автора (authorUsername), связанные с тендером
	var bids []models.Bid
	if err := utils.DB.Where("tender_id = ? AND author_id = ? AND author_type = ?", tenderId, employee.ID, models.USER).Find(&bids).Error; err != nil {
		http.Error(w, "Ошибка загрузки предложений данного автора", http.StatusInternalServerError)
		return
	}

	if len(bids) == 0 {
		http.Error(w, "У автора нет предложений к данному тендеру", http.StatusNotFound)
		return
	}

	// Получаем все отзывы на эти предложения
	var bidIds []uint
	for _, bid := range bids {
		bidIds = append(bidIds, bid.ID)
	}

	var reviews []models.BidFeedback
	if err := utils.DB.Where("bid_id IN (?)", bidIds).Find(&reviews).Error; err != nil {
		http.Error(w, "Ошибка загрузки обратной связи к предложением", http.StatusInternalServerError)
		return
	}

	// Возвращаем список отзывов в формате JSON
	utils.JSONFormat(w, r, reviews)
}

// Функция для хранения версий предложений
func saveBidsVersion(bid models.Bid) {
	version := models.BidVersion{
		BidID:       bid.ID,
		Name:        bid.Name,
		Description: bid.Description,
		Status:      bid.Status,
		TenderID:    bid.TenderID,
		AuthorID:    bid.AuthorID,
		AuthorType:  bid.AuthorType,
		Version:     bid.Version,
		CreatedAt:   bid.CreatedAt,
	}

	utils.DB.Create(&version)
}
