package model

type Category struct {
	ID    int    `json:"id" validate:"numeric,gte=0"`
	CType string `json:"cType"` // todo check if not something else?
	Name  string `json:"name" validate:"required,min=3,max=32"`
	//TODO add personal categories // check if exist in the main category list
}
