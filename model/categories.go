package model

type Category struct {
	ID   int    `json:"id" validate:"numeric,gte=0"`
	Name string `json:"name" validate:"required,min=3,max=32"`
	//todo add personal categories // check if exist in the main category list
}
