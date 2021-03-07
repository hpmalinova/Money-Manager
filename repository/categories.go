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

func NewCategoryRepoMysql(user, password, dbname string) *CategoryRepoMysql {
	connectionString := fmt.Sprintf("%s:%s@/%s", user, password, dbname)
	repo := &CategoryRepoMysql{}
	var err error
	repo.db, err = sql.Open("mysql", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	//repo.db.SetConnMaxLifetime(time.Minute * 5) // todo delete?
	//repo.db.SetMaxOpenConns(10)
	//repo.db.SetMaxIdleConns(10)
	//repo.db.SetConnMaxIdleTime(time.Minute * 3)

	return repo
}

func (g *CategoryRepoMysql) Find() ([]model.Category, error) {
	statement := `SELECT id, name FROM categories`

	rows, err := g.db.Query(statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []model.Category{}
	for rows.Next() {
		var category model.Category
		err := rows.Scan(&category.ID, &category.Name)
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
