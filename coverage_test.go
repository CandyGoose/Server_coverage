package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestFindUsersLimitOffset(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	client := &SearchClient{
		AccessToken: "test_token",
		URL:         ts.URL,
	}

	req := SearchRequest{
		Limit:  5,
		Offset: 0,
	}

	resp, err := client.FindUsers(req)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(resp.Users) != 5 {
		t.Errorf("expected 5 users, got %d", len(resp.Users))
	}
}

func SlowSearchServer(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Second * 2)
	SearchServer(w, r)
}

func TestTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SlowSearchServer))
	defer ts.Close()

	client := &SearchClient{
		AccessToken: "test_token",
		URL:         ts.URL,
	}

	req := SearchRequest{
		Limit:      1,
		Offset:     0,
		OrderField: "Name",
		OrderBy:    OrderByAsc,
	}

	_, err := client.FindUsers(req)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}

	expectedError := fmt.Sprintf("timeout for %s", url.Values{
		"limit":       []string{strconv.Itoa(req.Limit + 1)},
		"offset":      []string{strconv.Itoa(req.Offset)},
		"query":       []string{req.Query},
		"order_field": []string{req.OrderField},
		"order_by":    []string{strconv.Itoa(req.OrderBy)},
	}.Encode())

	if err.Error() != expectedError {
		t.Errorf("expected error %q, got %q", expectedError, err.Error())
	}
}

func BadJSONErrorSearchServer(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	if _, err := w.Write([]byte("invalid json")); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}

func TestInvalidJSONErrorResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(BadJSONErrorSearchServer))
	defer ts.Close()

	client := &SearchClient{
		AccessToken: "test_token",
		URL:         ts.URL,
	}

	req := SearchRequest{
		Limit:  1,
		Offset: 0,
	}

	_, err := client.FindUsers(req)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !strings.HasPrefix(err.Error(), "cant unpack error json:") {
		t.Errorf("expected error json unpack error, got %q", err.Error())
	}
}

func TestSuccessfulResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	client := &SearchClient{
		AccessToken: "test_token",
		URL:         ts.URL,
	}

	req := SearchRequest{
		Limit:      2,
		Offset:     0,
		Query:      "Boyd",
		OrderField: "Name",
		OrderBy:    OrderByAsc,
	}

	resp, err := client.FindUsers(req)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(resp.Users) != 1 {
		t.Errorf("expected 1 user, got %d", len(resp.Users))
	}

	if resp.Users[0].Name != "Boyd Wolf" {
		t.Errorf("expected user Boyd Wolf, got %s", resp.Users[0].Name)
	}
}

func TestEncoderError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if err := json.NewEncoder(w).Encode(make(chan int)); err == nil {
			t.Errorf("expected json encoding error, got nil")
		}
	}))
	defer ts.Close()

	client := &SearchClient{
		AccessToken: "test_token",
		URL:         ts.URL,
	}

	req := SearchRequest{
		Limit:  1,
		Offset: 0,
	}

	_, err := client.FindUsers(req)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestMaxLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	client := &SearchClient{
		AccessToken: "test_token",
		URL:         ts.URL,
	}

	req := SearchRequest{
		Limit:  100,
		Offset: 0,
	}

	resp, err := client.FindUsers(req)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(resp.Users) > 25 {
		t.Errorf("expected no more than 25 users, got %d", len(resp.Users))
	}
}

func TestOffsetBeyondRange(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	client := &SearchClient{
		AccessToken: "test_token",
		URL:         ts.URL,
	}

	req := SearchRequest{
		Limit:  1,
		Offset: 1000,
	}

	resp, err := client.FindUsers(req)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(resp.Users) != 0 {
		t.Errorf("expected 0 users, got %d", len(resp.Users))
	}
}

func TestInvalidOffset(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	client := &SearchClient{
		AccessToken: "test_token",
		URL:         ts.URL,
	}

	req := SearchRequest{
		Limit:  1,
		Offset: -1,
	}

	_, err := client.FindUsers(req)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	expectedError := "offset must be > 0"
	if err.Error() != expectedError {
		t.Errorf("expected error %q, got %q", expectedError, err.Error())
	}
}

func TestLoadDataFileCloseSuccess(t *testing.T) {
	originalDatasetFilePath := datasetFilePath
	datasetFilePath = "dataset.xml"

	err := loadData()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	datasetFilePath = originalDatasetFilePath
}

func TestLoadDataSuccess(t *testing.T) {
	originalDatasetFilePath := datasetFilePath
	datasetFilePath = "dataset.xml"

	err := loadData()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(users) == 0 {
		t.Errorf("expected users to be loaded, got none")
	}

	datasetFilePath = originalDatasetFilePath
}

func TestLoadData_FileOpenError(t *testing.T) {
	originalPath := datasetFilePath
	defer func() { datasetFilePath = originalPath }()

	datasetFilePath = "non_existent_file.xml"

	err := loadData()
	if err == nil {
		t.Errorf("Expected error when opening non-existent file, but got nil")
	}
}

func TestLoadData_XMLDecodeError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "invalid_xml*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err = tmpFile.WriteString("<invalid><xml></xml>"); err != nil {
		t.Fatalf("Failed to write invalid XML to temp file: %v", err)
	}
	tmpFile.Close()

	originalPath := datasetFilePath
	defer func() { datasetFilePath = originalPath }()
	datasetFilePath = tmpFile.Name()

	err = loadData()
	if err == nil {
		t.Errorf("Expected XML decode error, but got nil")
	}
}

func TestHandleError(t *testing.T) {
	rr := httptest.NewRecorder()

	handleError(rr, fmt.Errorf("test error"), http.StatusBadRequest)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status code %v, but got %v", http.StatusBadRequest, status)
	}

	expected := "test error\n"
	if rr.Body.String() != expected {
		t.Errorf("Expected body %v, but got %v", expected, rr.Body.String())
	}
}

func TestValidateIntParam_EmptyValue(t *testing.T) {
	result, err := validateIntParam("", 5)
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}
	if result != 5 {
		t.Errorf("Expected default value 5, but got %v", result)
	}
}

func TestValidateIntParam_ValidValue(t *testing.T) {
	result, err := validateIntParam("10", 5)
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}
	if result != 10 {
		t.Errorf("Expected value 10, but got %v", result)
	}
}

func TestValidateIntParam_InvalidValue(t *testing.T) {
	_, err := validateIntParam("invalid", 5)
	if err == nil {
		t.Errorf("Expected error, but got nil")
	}
}

func TestValidateParams_InvalidLimit(t *testing.T) {
	req := httptest.NewRequest("GET", "/?limit=invalid", nil)
	_, _, _, _, err := validateParams(req)
	if err == nil {
		t.Errorf("Expected error for invalid limit, but got nil")
	}
}

func TestValidateParams_InvalidOffset(t *testing.T) {
	req := httptest.NewRequest("GET", "/?offset=invalid", nil)
	_, _, _, _, err := validateParams(req)
	if err == nil {
		t.Errorf("Expected error for invalid offset, but got nil")
	}
}

func TestValidateParams_InvalidOrderField(t *testing.T) {
	req := httptest.NewRequest("GET", "/?order_field=InvalidField", nil)
	_, _, _, _, err := validateParams(req)
	if err == nil {
		t.Errorf("Expected error for invalid order_field, but got nil")
	}
}

func TestValidateParams_ValidParams(t *testing.T) {
	req := httptest.NewRequest("GET", "/?limit=10&offset=0&order_field=Name&order_by=1", nil)
	limit, offset, orderField, orderBy, err := validateParams(req)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if limit != 10 {
		t.Errorf("Expected limit 10, but got %v", limit)
	}

	if offset != 0 {
		t.Errorf("Expected offset 0, but got %v", offset)
	}

	if orderField != "Name" {
		t.Errorf("Expected order_field 'Name', but got %v", orderField)
	}

	if orderBy != 1 {
		t.Errorf("Expected order_by 1, but got %v", orderBy)
	}
}

func TestFilterAndSortUsers_NameFilter(t *testing.T) {
	users := []UserServer{
		{ID: 1, Name: "Alice", Age: 30, About: "Engineer"},
		{ID: 2, Name: "Bob", Age: 25, About: "Doctor"},
		{ID: 3, Name: "Charlie", Age: 35, About: "Teacher"},
	}

	filtered := filterAndSortUsers(users, "Bob", "Name", 0)

	if len(filtered) != 1 || filtered[0].Name != "Bob" {
		t.Errorf("Expected 1 user named 'Bob', but got %v", filtered)
	}
}

func TestFilterAndSortUsers_AboutFilter(t *testing.T) {
	users := []UserServer{
		{ID: 1, Name: "Alice", Age: 30, About: "Engineer"},
		{ID: 2, Name: "Bob", Age: 25, About: "Doctor"},
		{ID: 3, Name: "Charlie", Age: 35, About: "Engineer"},
	}

	filtered := filterAndSortUsers(users, "Engineer", "Name", 0)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 users with 'Engineer' in About, but got %v", len(filtered))
	}
}

func TestFilterAndSortUsers_SortByID(t *testing.T) {
	users := []UserServer{
		{ID: 2, Name: "Bob", Age: 25},
		{ID: 1, Name: "Alice", Age: 30},
		{ID: 3, Name: "Charlie", Age: 35},
	}

	sorted := filterAndSortUsers(users, "", "Id", 1)

	if sorted[0].ID != 1 || sorted[1].ID != 2 || sorted[2].ID != 3 {
		t.Errorf("Expected users sorted by ID ascending, but got %v", sorted)
	}

	sorted = filterAndSortUsers(users, "", "Id", -1)

	if sorted[0].ID != 3 || sorted[1].ID != 2 || sorted[2].ID != 1 {
		t.Errorf("Expected users sorted by ID descending, but got %v", sorted)
	}
}

func TestFilterAndSortUsers_SortByAge(t *testing.T) {
	users := []UserServer{
		{ID: 1, Name: "Alice", Age: 30},
		{ID: 2, Name: "Bob", Age: 25},
		{ID: 3, Name: "Charlie", Age: 35},
	}

	sorted := filterAndSortUsers(users, "", "Age", 1)

	if sorted[0].Age != 25 || sorted[1].Age != 30 || sorted[2].Age != 35 {
		t.Errorf("Expected users sorted by Age ascending, but got %v", sorted)
	}

	sorted = filterAndSortUsers(users, "", "Age", -1)

	if sorted[0].Age != 35 || sorted[1].Age != 30 || sorted[2].Age != 25 {
		t.Errorf("Expected users sorted by Age descending, but got %v", sorted)
	}
}

func TestSearchServer_MissingAccessToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/?limit=10", nil)
	rr := httptest.NewRecorder()

	SearchServer(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %v, got %v", http.StatusUnauthorized, rr.Code)
	}
}

func TestSearchServer_InvalidLimit(t *testing.T) {
	req := httptest.NewRequest("GET", "/?limit=invalid", nil)
	req.Header.Set("AccessToken", "valid_token")
	rr := httptest.NewRecorder()

	SearchServer(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %v, got %v", http.StatusBadRequest, rr.Code)
	}
}
func TestSearchServer_LoadDataError(t *testing.T) {
	originalPath := datasetFilePath
	defer func() { datasetFilePath = originalPath }()

	datasetFilePath = "non_existent_file.xml"

	req := httptest.NewRequest("GET", "/?limit=10", nil)
	req.Header.Set("AccessToken", "valid_token")
	rr := httptest.NewRecorder()

	SearchServer(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %v, but got %v", http.StatusInternalServerError, rr.Code)
	}

	expectedPattern := `Failed to load data`
	matched, err := regexp.MatchString(expectedPattern, rr.Body.String())
	if err != nil {
		t.Fatalf("Error compiling regex: %v", err)
	}
	if !matched {
		t.Errorf("Expected body to match pattern %v, but got %v", expectedPattern, rr.Body.String())
	}
}

func TestSearchServer_WriteJSONError(t *testing.T) {
	// Создаем тестовый запрос
	req := httptest.NewRequest("GET", "/?limit=1&offset=0", nil)
	req.Header.Set("AccessToken", "valid_token")
	rr := httptest.NewRecorder()

	// Симулируем ошибку сериализации JSON, передав канал
	paginatedUsers := make(chan int) // Несериализуемые данные

	// Вызов SearchServer, который должен вызвать ошибку записи JSON
	writeJSONResponse(rr, http.StatusOK, paginatedUsers)

	// Проверяем код ошибки в ответе
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %v, but got %v", http.StatusInternalServerError, rr.Code)
	}

	// Проверяем содержание тела ответа
	expectedError := "Failed to write response"
	if !strings.Contains(rr.Body.String(), expectedError) {
		t.Errorf("Expected error to contain %q, but got %q", expectedError, rr.Body.String())
	}
}

func TestFindUsers_LimitLessThanZero(t *testing.T) {
	client := &SearchClient{
		AccessToken: "valid_token",
		URL:         "http://localhost",
	}

	req := SearchRequest{Limit: -1}
	_, err := client.FindUsers(req)
	if err == nil || err.Error() != "limit must be > 0" {
		t.Errorf("expected error 'limit must be > 0', got %v", err)
	}
}

func TestFindUsers_BadAccessToken(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &SearchClient{
		AccessToken: "invalid_token",
		URL:         server.URL,
	}

	req := SearchRequest{Limit: 10}
	_, err := client.FindUsers(req)
	if err == nil || err.Error() != "bad AccessToken" {
		t.Errorf("expected error 'bad AccessToken', got %v", err)
	}
}

func TestFindUsers_OrderFieldInvalid(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error": "OrderField invalid"}`, http.StatusBadRequest)
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &SearchClient{
		AccessToken: "valid_token",
		URL:         server.URL,
	}

	req := SearchRequest{Limit: 10, OrderField: "invalid_field"}
	_, err := client.FindUsers(req)
	if err == nil || !strings.Contains(err.Error(), "OrderFeld invalid") {
		t.Errorf("expected error to contain 'OrderFeld invalid', got %v", err)
	}
}

func TestFindUsers_JSONUnmarshalError(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("invalid json")); err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &SearchClient{
		AccessToken: "valid_token",
		URL:         server.URL,
	}

	req := SearchRequest{Limit: 10}
	_, err := client.FindUsers(req)
	if err == nil || !strings.Contains(err.Error(), "cant unpack result json") {
		t.Errorf("expected error to contain 'cant unpack result json', got %v", err)
	}
}

func TestFindUsers_UnknownBadRequestError(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`{"error": "Some unknown error"}`)); err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)
			return
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	client := &SearchClient{
		AccessToken: "valid_token",
		URL:         server.URL,
	}

	req := SearchRequest{Limit: 10, OrderField: "invalid_field"}
	_, err := client.FindUsers(req)
	if err == nil || err.Error() != "unknown bad request error: Some unknown error" {
		t.Errorf("expected error 'unknown bad request error: Some unknown error', got %v", err)
	}
}
