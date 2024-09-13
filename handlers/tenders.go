package handlers

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"testAvito/models"
	"testAvito/utils"
	"testAvito/validators"
)

// Создание тендера
// CreateTenderHandler создает новый тендер.
// @Summary Создание нового тендера
// @Description Создает новый тендер, декодируя данные из тела запроса и сохраняя их в базе данных.
// @Tags Tenders
// @Accept  json
// @Produce  json
// @Param tender body models.Tender true "Данные для создания тендера"
// @Success 200 {object} models.Tender "Успешно созданный тендер"
// @Failure 400 {string} string "Неверные данные для создания тендера"
// @Failure 500 {string} string "Ошибка сохранения тендера в базе данных"
// @Router /tenders/new [post]
func CreateTenderHandler(w http.ResponseWriter, r *http.Request) {
	var tender models.Tender

	log.Println("Получен запрос на создание тендера")

	// Декодируем тело запроса
	if err := json.NewDecoder(r.Body).Decode(&tender); err != nil {
		log.Println("Ошибка декодирования JSON:", err)
		http.Error(w, "Неверные данные", http.StatusBadRequest)
		return
	}

	log.Println("Декодирование JSON прошло успешно")
	if err := validators.ValidateCreateTender(w, &tender); err != nil {
		return
	}
	// Создаем тендер в базе данных
	if err := utils.DB.Create(&tender).Error; err != nil {
		log.Println("Ошибка создания тендера в базе данных:", err)
		http.Error(w, "Ошибка создания тендера", http.StatusInternalServerError)
		return
	}

	log.Println("Тендер успешно создан в базе данных")

	// Сохраняем для контроля версий
	saveTenderVersion(tender)

	// Форматируем JSON с отступами для лучшего чтения
	utils.JSONFormat(w, r, tender)
}

// SetStatusTenderHandler изменяет статус тендера по его ID.
// @Summary Изменение статуса тендера
// @Description Позволяет изменить статус тендера на "publish" или "close", если пользователь имеет права доступа.
// @Tags Tenders
// @Accept  json
// @Produce  json
// @Param tenderId path int true "ID тендера"
// @Param username query string true "Имя пользователя, изменяющего статус"
// @Param status query string true "Новый статус тендера ('publish' или 'close')"
// @Success 200 {object} models.Tender "Успешно обновленный тендер"
// @Failure 400 {string} string "Неверный ID тендера или неправильный статус"
// @Failure 403 {string} string "Нет прав для изменения статуса"
// @Failure 404 {string} string "Тендер или пользователь не найдены"
// @Failure 500 {string} string "Ошибка обновления тендера"
// @Router /tenders/{tenderId}/status [put]
func SetStatusTenderHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	// Конвертируем url с припиской tenderId в значение integer
	tenderId, err := strconv.Atoi(params["tenderId"])
	if err != nil {
		http.Error(w, "Неверный тендер ID", http.StatusBadRequest)
		return
	}
	username := r.URL.Query().Get("username")
	status := r.URL.Query().Get("status")
	var employee models.Employee
	if err := utils.DB.Where("username = ?", username).First(&employee).Error; err != nil {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	var tender models.Tender
	// Далее среди тендеров ищу тот же тендер что и с этим же айдишником
	if err := utils.DB.First(&tender, tenderId).Error; err != nil {
		http.Error(w, "Тендер не найден", http.StatusNotFound)
		return
	}
	if tender.Status == models.CLOSED {
		http.Error(w, "Тендер был закрыт, изменения невозможны.", http.StatusBadRequest)
		return
	}

	var organizationResponsible models.OrganizationResponsible
	if err := utils.DB.Where("organization_id = ? AND user_id = ?", tender.OrganizationID, employee.ID).First(&organizationResponsible).Error; err != nil {
		http.Error(w, "У вас нет прав изменять статус этого тендера", http.StatusForbidden)
		return
	}

	// Проверка на лог в статусе
	switch status {
	case "publish":
		if tender.Status != models.CREATED {
			http.Error(w, "Тендер должен быть в статусе CREATED", http.StatusBadRequest)
			return
		}
		tender.Status = models.PUBLISHED
		tender.Version++
		log.Println("Тендер был опубликован")
	case "close":
		if tender.Status != models.PUBLISHED {
			http.Error(w, "Тендер должен быть в статусе PUBLISHED", http.StatusBadRequest)
			return
		}
		tender.Status = models.CLOSED
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
		tender.Version++
		log.Println("Тендер закрыт")
	default:
		http.Error(w, "Неправильное действие. Используй 'publish' или 'close'.", http.StatusBadRequest)
		return
	}

	// Сохранение в базу данных
	if err := utils.DB.Save(&tender).Error; err != nil {
		http.Error(w, "Ошибка обновления статуса тендера", http.StatusInternalServerError)
		return
	}
	saveTenderVersion(tender)

	// В красивом формате
	utils.JSONFormat(w, r, tender)
}

// Тендер всех пользователей с возможностью фильтрации по типу сервиса
// TenderShowHandler возвращает список всех тендеров с возможностью фильтрации по типу услуг.
// @Summary Получение списка тендеров
// @Description Возвращает список всех тендеров с возможностью фильтрации по типу услуг.
// @Tags Tenders
// @Accept  json
// @Produce  json
// @Param serviceType query string false "Тип услуг для фильтрации тендеров"
// @Success 200 {array} models.Tender "Список тендеров"
// @Failure 500 {string} string "Ошибка загрузки тендеров"
// @Router /tenders [get]
func TenderShowHandler(w http.ResponseWriter, r *http.Request) {
	var tenders []models.Tender
	serviceType := r.URL.Query().Get("serviceType")

	if serviceType != "" {
		log.Printf("Фильтрация по типу услуг: %s", serviceType)
		if err := utils.DB.Where("service_type = ?", serviceType).Find(&tenders).Error; err != nil {
			http.Error(w, "Ошибка поимка тендера.", http.StatusInternalServerError)
			return
		}
	} else {
		if err := utils.DB.Find(&tenders).Error; err != nil {
			http.Error(w, "Ошибка поимка тендера.", http.StatusInternalServerError)
			return
		}
	}
	utils.JSONFormat(w, r, tenders)
}

// GetStatusTenderHandler возвращает статус тендера по его ID.
// @Summary Получение статуса тендера
// @Description Возвращает статус тендера, если пользователь имеет права на просмотр статуса.
// @Tags Tenders
// @Accept  json
// @Produce  plain
// @Param tenderId path int true "ID тендера"
// @Param username query string true "Имя пользователя, запрашивающего статус тендера"
// @Success 200 {string} string "Статус тендера"
// @Failure 400 {string} string "Неправильный ID тендера или никнейм не введен"
// @Failure 403 {string} string "Пользователь не является ответственным за тендер"
// @Failure 404 {string} string "Тендер или пользователь не найдены"
// @Failure 500 {string} string "Ошибка сервера"
// @Router /tenders/{tenderId}/status [get]
func GetStatusTenderHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем параметры из URL
	params := mux.Vars(r)

	// Получаем имя пользователя из query
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Никнейм пользователя, кто хочет проверить статус тендера не введено", http.StatusBadRequest)
	}
	var employee models.Employee

	// Проверяем, что пользователь существует
	if err := utils.DB.Where("username = ?", username).First(&employee).Error; err != nil {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	// Преобразуем tenderId в число
	tenderId, err := strconv.Atoi(params["tenderId"])
	if err != nil {
		http.Error(w, "Неправильный тендер ID", http.StatusBadRequest)
		return
	}

	// Проверяем, что тендер существует
	var tender models.Tender
	if err = utils.DB.First(&tender, tenderId).Error; err != nil {
		http.Error(w, "Тендер не найден", http.StatusNotFound)
		return
	}

	// Проверяем, является ли пользователь ответственным за организацию
	var orgResponsible models.OrganizationResponsible
	if err = utils.DB.Where("organization_id = ? AND user_id = ?", tender.OrganizationID, employee.ID).First(&orgResponsible).Error; err != nil {
		http.Error(w, "Пользователь не является ответственным за тендер", http.StatusForbidden)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(tender.Status))

}

// Показать тендер определенного пользователя
// ShowTenderUserHandler возвращает список тендеров, созданных пользователем.
// @Summary Получение тендеров пользователя
// @Description Возвращает список всех тендеров, созданных пользователем по его имени.
// @Tags Tenders
// @Accept  json
// @Produce  json
// @Param username query string true "Имя пользователя, создавшего тендеры"
// @Success 200 {array} models.Tender "Список тендеров пользователя"
// @Failure 500 {string} string "Ошибка поиска тендеров"
// @Router /tenders/my [get]
func ShowTenderUserHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	var tenders []models.Tender
	if err := utils.DB.Where("creator_username = ?", username).Find(&tenders).Error; err != nil {
		http.Error(w, "Ошибка поиска тендера.", http.StatusInternalServerError)
		return
	}

	utils.JSONFormat(w, r, tenders)
}

// Изменить тендер (поиск его по id)
// EditTenderHandler редактирует тендер по его ID.
// @Summary Редактирование тендера
// @Description Обновляет данные тендера (имя, описание, тип услуг) по его ID, если пользователь имеет права.
// @Tags Tenders
// @Accept  json
// @Produce  json
// @Param tenderId path int true "ID тендера"
// @Param username query string true "Имя пользователя, инициирующего изменение"
// @Param tender body object true "Данные для обновления тендера (имя, описание, тип услуг)"
// @Success 200 {object} models.Tender "Обновленный тендер"
// @Failure 400 {string} string "Неверные данные или ID тендера"
// @Failure 403 {string} string "Нет прав на редактирование тендера"
// @Failure 404 {string} string "Тендер или пользователь не найдены"
// @Failure 500 {string} string "Ошибка обновления тендера"
// @Router /tenders/{tenderId}/edit [patch]
func EditTenderHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем параметр tenderId из URL
	params := mux.Vars(r)
	tenderIDStr := params["tenderId"]
	username := r.URL.Query().Get("username")

	var employee models.Employee
	if err := utils.DB.Where("username = ?", username).First(&employee).Error; err != nil {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}
	// Преобразуем tenderId в число
	tenderID, err := strconv.Atoi(tenderIDStr)
	if err != nil {
		log.Println("Неверный tenderId:", err)
		http.Error(w, "Неверный тендер ID", http.StatusBadRequest)
		return
	}

	// Найдем тендер по tenderId
	var tender models.Tender
	if err := utils.DB.First(&tender, tenderID).Error; err != nil {
		log.Println("Тендер не найден:", err)
		http.Error(w, "Тендер не найден", http.StatusNotFound)
		return
	}

	var organizationResponsible models.OrganizationResponsible
	if err := utils.DB.Where("organization_id = ? AND user_id = ?", tender.OrganizationID, employee.ID).First(&organizationResponsible).Error; err != nil {
		http.Error(w, "У вас нет прав изменять тендер.", http.StatusForbidden)
		return
	}

	// Декодируем обновлённые данные тендера из тела запроса
	var updatedTender map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updatedTender); err != nil {
		log.Println("Ошибка при декодировании JSON:", err)
		http.Error(w, "Неправильные введенные данные", http.StatusBadRequest)
		return
	}
	// Обновляем поля тендера только те которые были переданы
	if name, ok := updatedTender["name"]; ok {
		tender.Name = name.(string)
	}
	if description, ok := updatedTender["description"]; ok {
		tender.Description = description.(string)
	}
	if serviceType, ok := updatedTender["serviceType"]; ok {
		tender.ServiceType = serviceType.(string)
	}
	//if status, ok := updatedTender["status"]; ok {
	//	tender.Status = models.TenderStatus(status.(string))
	//}

	if err = validators.CheckCorrectStatusTender(tender.Status); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Увеличиваем версию тендера с каждым изменением
	tender.Version++

	// Сохраняем изменения в базе данных
	if err := utils.DB.Save(&tender).Error; err != nil {
		log.Println("Ошибка при сохранении тендера:", err)
		http.Error(w, "Ошибка обновления тендера", http.StatusInternalServerError)
		return
	}
	// Сохраняем для контроля версий
	saveTenderVersion(tender)

	// В красивом формате
	utils.JSONFormat(w, r, tender)
}

// Откат тендера к определённой версии
// RollbackTenderHandler откатывает тендер к указанной версии.
// @Summary Откат тендера к версии
// @Description Откатывает тендер к указанной версии на основании прав пользователя и статуса тендера. Откат невозможен, если тендер уже закрыт.
// @Tags Tenders
// @Accept  json
// @Produce  json
// @Param tenderId path int true "ID тендера"
// @Param version path int true "Версия тендера, к которой необходимо откатиться"
// @Param username query string true "Имя пользователя, инициирующего откат"
// @Success 200 {object} models.Tender "Откатанный тендер"
// @Failure 400 {string} string "Неправильный ID тендера или версия"
// @Failure 403 {string} string "Нет прав на откат тендера"
// @Failure 404 {string} string "Тендер или версия не найдены"
// @Failure 500 {string} string "Ошибка сохранения откатанного тендера"
// @Router /tenders/{tenderId}/rollback/{version} [put]
func RollbackTenderHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем параметры tenderId и version из URL
	params := mux.Vars(r)

	// Конвертируем айди в инт
	tenderID, err := strconv.Atoi(params["tenderId"])
	if err != nil {
		log.Println("Неверный tenderId:", err)
		http.Error(w, "Неправильный тендер ID", http.StatusBadRequest)
		return
	}

	// Конвертируем версию в инт
	version, err := strconv.Atoi(params["version"])
	if err != nil {
		log.Println("Неверная версия:", err)
		http.Error(w, "Неверная версия", http.StatusBadRequest)
		return
	}

	// Ищем указанную версию тендера в таблице tender_versions
	var tenderVersion models.TenderVersion
	if err := utils.DB.Where("tender_id = ? AND version = ?", tenderID, version).First(&tenderVersion).Error; err != nil {
		log.Println("Версия тендера не найдена:", err)
		http.Error(w, "Версия тендера не найдена", http.StatusNotFound)
		return
	}

	// Ищем текущий тендер по его ID
	var tender models.Tender
	if err := utils.DB.First(&tender, tenderID).Error; err != nil {
		log.Println("Тендер не найден:", err)
		http.Error(w, "Тендер не найден", http.StatusNotFound)
		return
	}
	if tender.Status == models.CLOSED {
		http.Error(w, "Тендер закрыт, дальнейшее использование его невозможно", http.StatusBadRequest)
		return
	}

	username := r.URL.Query().Get("username")
	var employee models.Employee
	if err := utils.DB.Where("username = ?", username).First(&employee).Error; err != nil {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}
	var organizationResponsible models.OrganizationResponsible
	if err := utils.DB.Where("organization_id = ? AND user_id = ?", tender.OrganizationID, employee.ID).First(&organizationResponsible).Error; err != nil {
		http.Error(w, "Вы не можете обращаться к прошлым версиям тендера, у вас нет прав.", http.StatusForbidden)
		return
	}

	// Обновляем текущий тендер данными из выбранной версии
	tender.Name = tenderVersion.Name
	tender.Description = tenderVersion.Description
	tender.ServiceType = tenderVersion.ServiceType
	tender.Status = tenderVersion.Status

	// Используем ту же версию, к которой откатились
	//tender.Version = tenderVersion.Version - по тз не понял как изменять версию

	tender.Version++

	// Сохраняем откатанный тендер с обновлёнными данными, сохраняя его ID
	if err := utils.DB.Save(&tender).Error; err != nil {
		log.Println("Ошибка при сохранении откатанного тендера:", err)
		http.Error(w, "Ошибка при сохранении откатанного тендера", http.StatusInternalServerError)
		return
	}

	saveTenderVersion(tender)

	// Возвращаем все в нормальный вид (unmarshal)
	utils.JSONFormat(w, r, tender)
}

// Фукнция которая переносит в бд все версии продукта по айдишникам
func saveTenderVersion(tender models.Tender) {
	version := models.TenderVersion{
		TenderID:    tender.ID,
		Name:        tender.Name,
		Description: tender.Description,
		ServiceType: tender.ServiceType,
		Status:      tender.Status,
		Version:     tender.Version,
	}

	utils.DB.Create(&version)
}
