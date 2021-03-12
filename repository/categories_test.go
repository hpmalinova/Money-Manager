package repository

import (
	"database/sql"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"github.com/DATA-DOG/go-sqlmock"

)

func NewMock() (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	return db, mock
}

func TestCategoryRepoMysql_FindByName(t *testing.T) {
	t.Run("category exists", func(t *testing.T) {
		db, mock := NewMock()
		repo := &CategoryRepoMysql{db}
		defer func() {
			repo.Close()
		}()

		query := "SELECT id, c_type, name FROM categories WHERE name = ?"

		categoryName := "food"
		rows := sqlmock.NewRows([]string{"id", "c_type","name"}).
			AddRow(1, "expense", categoryName)

		mock.ExpectQuery(query).WithArgs(categoryName).WillReturnRows(rows)

		category, err := repo.FindByName(categoryName)
		assert.NotNil(t, category)
		assert.NoError(t, err)
	})
	t.Run("category does not exist", func(t *testing.T) {
		db, mock := NewMock()
		repo := &CategoryRepoMysql{db}
		defer func() {
			repo.Close()
		}()

		query := "SELECT id, c_type, name FROM categories WHERE name = ?"

		rows := sqlmock.NewRows([]string{"id", "c_type","name"})

		categoryName := "food"
		mock.ExpectQuery(query).WithArgs(categoryName).WillReturnRows(rows)

		category, err := repo.FindByName(categoryName)
		assert.Empty(t, category)
		assert.Error(t, err)
	})
}

func TestCategoryRepoMysql_FindIncomes(t *testing.T) {
	t.Run("have incomes", func(t *testing.T) {
		db, mock := NewMock()
		repo := &CategoryRepoMysql{db}
		defer func() {
			repo.Close()
		}()

		query := `SELECT id, c_type, name FROM categories WHERE c_type = ?`

		cType := "income"
		rows := sqlmock.NewRows([]string{"id", "c_type","name"}).
			AddRow(1, "income", "salary").AddRow(2,"income", "savings")

		mock.ExpectQuery(query).WithArgs(cType).WillReturnRows(rows)

		incomes, err := repo.FindIncomes()
		assert.NotNil(t, incomes)
		assert.NoError(t, err)
	})
	t.Run("no categories", func(t *testing.T) {
		db, mock := NewMock()
		repo := &CategoryRepoMysql{db}
		defer func() {
			repo.Close()
		}()

		query := `SELECT id, c_type, name FROM categories WHERE c_type = ?`

		cType := "income"
		rows := sqlmock.NewRows([]string{"id", "c_type","name"})

		mock.ExpectQuery(query).WithArgs(cType).WillReturnRows(rows)

		categories, _ := repo.FindIncomes()
		assert.Empty(t, categories)
	})
	t.Run("no income categories", func(t *testing.T) {
		db, mock := NewMock()
		repo := &CategoryRepoMysql{db}
		defer func() {
			repo.Close()
		}()

		query := `SELECT id, c_type, name FROM categories WHERE c_type = ?`

		cType := "income"
		rows := sqlmock.NewRows([]string{"id", "name", "email", "phone"})

		mock.ExpectQuery(query).WithArgs(cType).WillReturnRows(rows)

		categories, _ := repo.FindIncomes()
		assert.Empty(t, categories)
	})
}

func TestCategoryRepoMysql_FindExpenses(t *testing.T) {
	cType := "expense"
	t.Run("have expenses", func(t *testing.T) {
		db, mock := NewMock()
		repo := &CategoryRepoMysql{db}
		defer func() {
			repo.Close()
		}()

		query := `SELECT id, c_type, name FROM categories WHERE c_type = ?`

		rows := sqlmock.NewRows([]string{"id", "c_type","name"}).
			AddRow(1, "expense", "food").AddRow(2,"expense", "home")

		mock.ExpectQuery(query).WithArgs(cType).WillReturnRows(rows)

		incomes, err := repo.FindExpenses()
		assert.NotNil(t, incomes)
		assert.NoError(t, err)
	})
	t.Run("no categories", func(t *testing.T) {
		db, mock := NewMock()
		repo := &CategoryRepoMysql{db}
		defer func() {
			repo.Close()
		}()

		query := `SELECT id, c_type, name FROM categories WHERE c_type = ?`

		rows := sqlmock.NewRows([]string{"id", "c_type","name"})

		mock.ExpectQuery(query).WithArgs(cType).WillReturnRows(rows)

		categories, _ := repo.FindExpenses()
		assert.Empty(t, categories)
	})
	t.Run("no expense categories", func(t *testing.T) {
		db, mock := NewMock()
		repo := &CategoryRepoMysql{db}
		defer func() {
			repo.Close()
		}()

		query := `SELECT id, c_type, name FROM categories WHERE c_type = ?`

		rows := sqlmock.NewRows([]string{"id", "name", "email", "phone"})

		mock.ExpectQuery(query).WithArgs(cType).WillReturnRows(rows)

		categories, _ := repo.FindExpenses()
		assert.Empty(t, categories)
	})
}

func TestCategoryRepoMysql_FindAll(t *testing.T) {
	t.Run("have categories", func(t *testing.T) {
		db, mock := NewMock()
		repo := &CategoryRepoMysql{db}
		defer func() {
			repo.Close()
		}()

		query := `SELECT id, c_type, name FROM categories`

		rows := sqlmock.NewRows([]string{"id", "c_type","name"}).
			AddRow(1, "income", "salary").AddRow(2,"expense", "food")

		mock.ExpectQuery(query).WillReturnRows(rows)

		incomes, err := repo.FindAll()
		assert.NotNil(t, incomes)
		assert.NoError(t, err)
	})
	t.Run("no categories", func(t *testing.T) {
		db, mock := NewMock()
		repo := &CategoryRepoMysql{db}
		defer func() {
			repo.Close()
		}()

		query := `SELECT id, c_type, name FROM categories`

		rows := sqlmock.NewRows([]string{"id", "c_type","name"})

		mock.ExpectQuery(query).WillReturnRows(rows)

		categories, _ := repo.FindAll()
		assert.Empty(t, categories)
	})
}
