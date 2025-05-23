package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateAndGetUser(t *testing.T) {
	dbConn := setupTestDB(t)

	u := &User{
		Email:        "test@example.com",
		PasswordHash: "hash",
		CreatedAt:    time.Now(),
	}

	id, err := CreateUser(dbConn, u)
	assert.NoError(t, err)
	assert.Greater(t, id, int64(0))

	fetched, err := GetUserByEmail(dbConn, "test@example.com")
	assert.NoError(t, err)
	assert.Equal(t, id, fetched.ID)
	assert.Equal(t, u.Email, fetched.Email)
}

func TestGetUserByID(t *testing.T) {
	dbConn := setupTestDB(t)

	u := &User{
		Email:        "byid@example.com",
		PasswordHash: "hash",
		CreatedAt:    time.Now(),
	}

	id, err := CreateUser(dbConn, u)
	assert.NoError(t, err)

	fetched, err := GetUserByID(dbConn, id)
	assert.NoError(t, err)
	assert.Equal(t, u.Email, fetched.Email)
	assert.Equal(t, id, fetched.ID)
}
