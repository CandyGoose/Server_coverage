package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

const OrderFieldName = "Name"

var datasetFilePath = "dataset.xml"

type UserXML struct {
	ID        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type UsersXML struct {
	Users []UserXML `xml:"row"`
}

type UserServer struct {
	ID     int
	Name   string
	Age    int
	About  string
	Gender string
}

var users []UserServer

// Централизованная обработка ошибок
func handleError(w http.ResponseWriter, err error, statusCode int) {
	http.Error(w, err.Error(), statusCode)
}

// Загрузка данных из XML
func loadData() error {
	xmlFile, err := os.Open(datasetFilePath)
	if err != nil {
		return fmt.Errorf("failed to open dataset file: %w", err)
	}
	defer xmlFile.Close()

	var data UsersXML
	if err := xml.NewDecoder(xmlFile).Decode(&data); err != nil {
		return fmt.Errorf("failed to decode XML: %w", err)
	}

	users = make([]UserServer, len(data.Users))
	for i, u := range data.Users {
		users[i] = UserServer{
			ID:     u.ID,
			Name:   u.FirstName + " " + u.LastName,
			Age:    u.Age,
			About:  u.About,
			Gender: u.Gender,
		}
	}
	return nil
}

// Универсальная функция для валидации параметров
func validateIntParam(value string, defaultVal int) (int, error) {
	if value == "" {
		return defaultVal, nil
	}
	return strconv.Atoi(value)
}

// Валидация и обработка параметров
func validateParams(r *http.Request) (limit, offset int, orderField string, orderBy int, err error) {
	if limit, err = validateIntParam(r.FormValue("limit"), 10); err != nil {
		return
	}
	if offset, err = validateIntParam(r.FormValue("offset"), 0); err != nil {
		return
	}
	orderField = r.FormValue("order_field")
	if orderField == "" {
		orderField = OrderFieldName
	} else if orderField != "Id" && orderField != "Age" && orderField != OrderFieldName {
		err = fmt.Errorf("invalid order_field: %s", orderField)
		return
	}
	orderBy, err = validateIntParam(r.FormValue("order_by"), 0)
	return
}

// Фильтрация и сортировка пользователей
func filterAndSortUsers(users []UserServer, query, orderField string, orderBy int) []UserServer {
	filtered := make([]UserServer, 0)
	for _, user := range users {
		if query == "" || strings.Contains(user.Name, query) || strings.Contains(user.About, query) {
			filtered = append(filtered, user)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		switch orderField {
		case "Id":
			if orderBy == 1 {
				return filtered[i].ID < filtered[j].ID
			}
			return filtered[i].ID > filtered[j].ID
		case "Age":
			if orderBy == 1 {
				return filtered[i].Age < filtered[j].Age
			}
			return filtered[i].Age > filtered[j].Age
		default:
			if orderBy == 1 {
				return filtered[i].Name < filtered[j].Name
			}
			return filtered[i].Name > filtered[j].Name
		}
	})

	return filtered
}

// Пагинация пользователей
func paginate(users []UserServer, limit, offset int) []UserServer {
	if offset >= len(users) {
		return []UserServer{}
	}
	if limit > len(users)-offset {
		limit = len(users) - offset
	}
	return users[offset : offset+limit]
}

// Централизованная отправка JSON-ответа
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}

// Вспомогательная функция для выполнения действий с проверкой на ошибки
func executeWithErrorCheck(w http.ResponseWriter, fn func() error, errorMsg string, statusCode int) bool {
	if err := fn(); err != nil {
		handleError(w, fmt.Errorf("%s: %w", errorMsg, err), statusCode)
		return false
	}
	return true
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	// Обертка для проверки access token
	if !executeWithErrorCheck(w, func() error {
		if r.Header.Get("AccessToken") == "" {
			return fmt.Errorf("unauthorized")
		}
		return nil
	}, "Unauthorized", http.StatusUnauthorized) {
		return
	}

	// Валидация параметров запроса
	limit, offset, orderField, orderBy, err := validateParams(r)
	if err != nil {
		handleError(w, err, http.StatusBadRequest)
		return
	}

	// Загрузка данных
	if !executeWithErrorCheck(w, loadData, "Failed to load data", http.StatusInternalServerError) {
		return
	}

	// Основная логика фильтрации, сортировки и ответа
	filteredUsers := filterAndSortUsers(users, r.FormValue("query"), orderField, orderBy)
	paginatedUsers := paginate(filteredUsers, limit, offset)

	writeJSONResponse(w, http.StatusOK, paginatedUsers)
}
