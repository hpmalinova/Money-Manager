package model

type Category struct {
	ID    int    `json:"id" validate:"numeric,gte=0"`
	CType string `json:"cType"`
	Name  string `json:"name" validate:"required,min=3,max=32"`
}
