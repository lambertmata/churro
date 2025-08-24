package churro

import (
	"encoding/json"
	"errors"
	validator2 "github.com/lambertmata/churro/validator"
	"testing"
)

type CreateUserProfile struct {
	Email       *string `json:"email" validate:"required|email"`
	Name        string  `json:"name" validate:"required"`
	LastName    string  `json:"last_name" validate:""`
	BirthDate   string  `json:"birth_date" validate:"required|date_format:2006-01-02 00:00:00"`
	StartDate   string  `json:"start_date" validate:"date"`
	Description string  `json:"description" validate:"required|min:3"`
	IsTest      bool    `json:"is_test" validate:"required"`
}

type Customer struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required|email"`
}

type Item struct {
	ProductId    string  `json:"product_id"`
	ProductName  string  `json:"product_name"`
	Quantity     int     `json:"quantity"`
	Price        float64 `json:"price"`
	ShipSeparate bool    `json:"ship_separate"`
}

type OrderRequest struct {
	Customer   Customer `json:"customer" validate:"required"`
	Tags       []string `json:"tags" validate:"required"`
	Items      []Item   `json:"items" validate:"required"`
	Currency   string   `json:"currency" validate:"required|in:eur,usd"`
	CouponCode string   `json:"coupon_code" validate:"uuid"`
}

func TestStructValidation(t *testing.T) {

	validator := validator2.NewValidator()

	email := "lambert@email.com"
	structUser := CreateUserProfile{
		Name:        "Lambert",
		Email:       &email,
		LastName:    "Mata",
		BirthDate:   "1999-01-01 00:00:00",
		StartDate:   "2023-01-15",
		Description: "Software",
		IsTest:      true,
	}

	err := validator.Validate(structUser)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

}

func TestJSONValidation(t *testing.T) {

	validator := validator2.NewValidator()

	jsonUser := `
		{
			"name": "Lambert",
			"last_name": "Mata",			
			"email": "lambert@example",
			"birth_date": "1999-01-01 00:00:00",
			"start_date": "2023-01-15",
			"description": "Software",
			"is_test": true
		}
	`

	var user CreateUserProfile

	err := json.Unmarshal([]byte(jsonUser), &user)

	err = errors.Join(err, validator.Validate(user))

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestComplexJSONValidation(t *testing.T) {

	validator := validator2.NewValidator()

	jsonReq := `
		{
          "currency": "eur",
          "coupon_code": "a5382fa9-ca56-4b3f-a77f-37174bd83c9d",
		  "customer": {
			"name": "John Smith",
			"email": "johnsmith@example.com"
		  },
          "tags": ["s"],
		  "items": [
			{
			  "product_id": "101",
			  "product_name": "Smartphone",
			  "quantity": 1,
			  "price": 699.99,
              "ship_separate": true
			},
			{
			  "product_id": "202",
			  "product_name": "Headphones",
			  "quantity": 2,
			  "price": 99.99
			}
		  ]
		}
	`

	var orderRequest OrderRequest

	err := json.Unmarshal([]byte(jsonReq), &orderRequest)

	err = errors.Join(err, validator.Validate(orderRequest))

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
