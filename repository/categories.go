package repository

import (
	"database/sql"
	"fmt"
	"github.com/hpmalinova/Money-Manager/model"
	"log"
)

type CategoryRepoMysql struct {
	db *sql.DB
}

const (
	expenses = "expenses"
	incomes = "incomes"
)

func NewCategoryRepoMysql(user, password, dbname string) *CategoryRepoMysql {
	connectionString := fmt.Sprintf("%s:%s@/%s", user, password, dbname)
	repo := &CategoryRepoMysql{}
	var err error
	repo.db, err = sql.Open("mysql", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	return repo
}

func (c *CategoryRepoMysql) FindByName(categoryName string) (*model.Category, error) {
	category := &model.Category{}
	statement := `SELECT id, c_type, name FROM categories WHERE name = ?`
	err := c.db.QueryRow(statement, categoryName).Scan(&category.ID, &category.CType, &category.Name)
	if err != nil {
		return nil, err
	}
	return category, nil
}

func (c *CategoryRepoMysql) FindExpenses() ([]model.Category, error) {
	statement := `SELECT id, c_type, name FROM categories WHERE c_type = ?`
	return c.findByType(statement, expenses)
	//rows, err := c.db.Query(statement)
	//if err != nil {
	//	return nil, err
	//}
	//defer rows.Close()
	//
	//categories := []model.Category{}
	//for rows.Next() {
	//	var category model.Category
	//	err := rows.Scan(&category.ID, &category.CType, &category.Name)
	//	if err != nil {
	//		return nil, err
	//	}
	//	categories = append(categories, category)
	//}
	//rows.Close()
	//if err = rows.Err(); err != nil {
	//	return nil, err
	//}
	//return categories, nil
}

func (c *CategoryRepoMysql) FindIncomes() ([]model.Category, error) {
	statement := `SELECT id, c_type, name FROM categories WHERE c_type = ?`
	return c.findByType(statement, incomes)
}

func (c *CategoryRepoMysql) findByType(statement string, cType string) ([]model.Category, error) {
	rows, err := c.db.Query(statement, cType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []model.Category{}
	for rows.Next() {
		var category model.Category
		err := rows.Scan(&category.ID, &category.CType, &category.Name)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}

func (c *CategoryRepoMysql) FindAll() ([]model.Category, error) {
	statement := `SELECT id, c_type, name FROM categories`

	rows, err := c.db.Query(statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []model.Category{}
	for rows.Next() {
		var category model.Category
		err := rows.Scan(&category.ID, &category.CType, &category.Name)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}
