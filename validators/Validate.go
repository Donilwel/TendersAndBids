package validators

import (
	"errors"
	"gorm.io/gorm"
	"log"
	"net/http"
	"testAvito/models"
	"testAvito/utils"
)

// Проверка корректности введеного имени пользователя
func CheckUsername(employee *models.Employee) (bool, error) {
	if employee.Username == "" {
		return false, errors.New("Пользователь не введен")
	}

	result := utils.DB.Where("username = ?", employee.Username).First(&employee)

	// Проверка на то, что пользователь не найден
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, errors.New("Пользователь не найден")
	}

	// Проверка на другие ошибки
	if result.Error != nil {
		return false, errors.New("Ошибка базы данныхr")
	}

	return true, nil
}

// Проверка на существования организации по айдишнику
func CheckOrganizationsExist(organization models.Organization) (bool, error) {
	result := utils.DB.Where("id = ?", organization.ID).First(&organization)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, errors.New("Организация не найдена")
	}
	if result.Error != nil {
		return false, errors.New("Ошибка базы данных")
	}
	return true, nil
}

// Проверка ввода статуса
func CheckCorrectStatusTender(status models.TenderStatus) error {
	switch status {
	case models.PUBLISHED:
		return nil
	case models.CREATED:
		return nil
	case models.CLOSED:
		return nil
	default:
		return errors.New("Неверный статус, статус должен быть: PUBLISHED, CREATED, CLOSED")
	}
}

//func CheckCorrectStatusBids(status models.BidStatus) error {
//	switch status {
//	case models.CREATEDBid:
//		return nil
//	case models.PUBLISHEDBid:
//		return nil
//	case models.APPROVED:
//		return nil
//	case models.CANCELED:
//		return nil
//	case models.REJECTED:
//		return nil
//	default:
//		return errors.New("invalid status, status must be CREATED, PUBLISHED, APPROVED, CANCELED, REJECTED")
//
//	}
//}

// Проверка организации и юзера на их совместимость
func CheckOrganizationResponsible(orgId uint, employeeId uint) (bool, error) {
	var orgResponsible models.OrganizationResponsible
	exist := utils.DB.Where("organization_id = ? AND user_id = ?", orgId, employeeId).First(&orgResponsible)
	if errors.Is(exist.Error, gorm.ErrRecordNotFound) {
		return false, errors.New("Данный пользователь не ответственен за организацию или не является автором этого тендера")
	}
	if exist.Error != nil {
		return false, errors.New("Ошибка при выполнении запроса к бд")
	}
	return true, nil
}

//func CheckDependOrganizationResponsibleAndEmployee(w http.ResponseWriter ,tender *models.Tender, employee *models.Employee, org *models.OrganizationResponsible){
//	if err := utils.DB.Where("organization_id = ? AND user_id = ?", tender.OrganizationID, employee.ID).First(&org).Error; err != nil {
//		http.Error(w, "You do not have permission to submit a decision for this bid", http.StatusForbidden)
//		return
//	}
//}

func ValidateCreateTender(w http.ResponseWriter, tender *models.Tender) error {
	employee := models.Employee{Username: tender.CreatorUsername}
	exists, err := CheckUsername(&employee)
	if err != nil {
		log.Println("Ошибка проверки пользователя:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	if !exists {
		log.Println("Пользователь не найден:", tender.CreatorUsername)
		http.Error(w, "User does not exist", http.StatusNotFound)
		return err
	}
	id := models.Organization{ID: tender.OrganizationID}
	exists, err = CheckOrganizationsExist(id)
	if err != nil {
		log.Println("Ошибка проверки айди организации:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	if !exists {
		log.Println("Пользователь не найден:", tender.CreatorUsername)
		http.Error(w, "User does not exist", http.StatusNotFound)
		return err
	}
	if err = CheckCorrectStatusTender(tender.Status); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	responsible, err := CheckOrganizationResponsible(tender.OrganizationID, employee.ID)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	if !responsible {
		http.Error(w, "User is not responsible for the organizations", http.StatusBadRequest)
		return err
	}
	return nil
}
