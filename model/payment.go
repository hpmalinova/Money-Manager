package model

type Pay struct {
	UserID       int    `json:"userID" validate:"numeric,gte=0"`
	Amount       int    `json:"amount" validate:"numeric,gte=0"`
	CategoryName string `json:"categoryName" validate:"required,min=3,max=32"`
	Description  string `json:"description,omitempty"`
}
