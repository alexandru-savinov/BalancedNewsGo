package db

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func TestUpdateArticleScoreRetry(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	db := sqlx.NewDb(mockDB, "sqlmock")
	defer mockDB.Close()

	query := "UPDATE articles"
	mock.ExpectExec(query).WithArgs(1.23, 0.45, int64(1)).WillReturnError(fmt.Errorf("database is locked"))
	mock.ExpectExec(query).WithArgs(1.23, 0.45, int64(1)).WillReturnResult(sqlmock.NewResult(1, 1))

	err = UpdateArticleScore(db, 1, 1.23, 0.45)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateArticleStatusDBError(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	db := sqlx.NewDb(mockDB, "sqlmock")
	defer mockDB.Close()

	mock.ExpectExec("UPDATE articles SET status").WithArgs("done", int64(5)).WillReturnError(fmt.Errorf("db error"))

	err = UpdateArticleStatus(db, 5, "done")
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateArticleScoreLLMRetry(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	db := sqlx.NewDb(mockDB, "sqlmock")
	defer mockDB.Close()

	query := "UPDATE articles"
	mock.ExpectExec(query).WithArgs(1.0, 0.5, int64(7)).WillReturnError(fmt.Errorf("database is locked"))
	mock.ExpectExec(query).WithArgs(1.0, 0.5, int64(7)).WillReturnResult(sqlmock.NewResult(1, 1))

	err = UpdateArticleScoreLLM(db, 7, 1.0, 0.5)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateArticleScoreLLMNoRows(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	db := sqlx.NewDb(mockDB, "sqlmock")
	defer mockDB.Close()

	query := "UPDATE articles"
	mock.ExpectExec(query).WithArgs(1.0, 0.5, int64(8)).WillReturnResult(sqlmock.NewResult(0, 0))

	err = UpdateArticleScoreLLM(db, 8, 1.0, 0.5)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateArticleScoreLLMError(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	db := sqlx.NewDb(mockDB, "sqlmock")
	defer mockDB.Close()

	query := "UPDATE articles"
	mock.ExpectExec(query).WithArgs(1.0, 0.5, int64(9)).WillReturnError(fmt.Errorf("boom"))

	err = UpdateArticleScoreLLM(db, 9, 1.0, 0.5)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFetchSourceByIDErrors(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	db := sqlx.NewDb(mockDB, "sqlmock")
	defer mockDB.Close()

	mock.ExpectQuery(`SELECT *`).WithArgs(int64(2)).WillReturnError(sql.ErrNoRows)
	src, err := FetchSourceByID(db, 2)
	assert.Nil(t, src)
	assert.EqualError(t, err, "source not found")

	mock.ExpectQuery(`SELECT *`).WithArgs(int64(3)).WillReturnError(fmt.Errorf("fail"))
	src, err = FetchSourceByID(db, 3)
	assert.Nil(t, src)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
