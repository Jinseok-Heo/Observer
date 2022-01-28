package dbrepo

import (
	"context"
	"database/sql"
	"golang.org/x/crypto/bcrypt"
	"log"
	"server_monitor/internal/models"
	"time"
)

// GetUserById returns a user by id
func (repo *mysqlDBRepo) GetUserById(id int) (models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `SELECT id, name, user_active, access_level, email, created_at, updated_at FROM users where id = $1`

	row := repo.DB.QueryRowContext(ctx, stmt, id)

	var u models.User

	err := row.Scan(
		&u.ID,
		&u.Name,
		&u.UserActive,
		&u.AccessLevel,
		&u.Email,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		log.Println(err)
		return u, err
	}

	return u, nil
}

// InsertUser adds a new record to the users table
func (repo *mysqlDBRepo) InsertUser(u models.User) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	hashedPassword, err := bcrypt.GenerateFromPassword(u.Password, 12)
	if err != nil {
		return 0, err
	}

	stmt := `INSERT INTO users (name, email, password, access_level, user_active)
				VALUES($1, $2, $3, $4, $5, $6) RETURNING id`

	var newId int
	err = repo.DB.QueryRowContext(ctx, stmt,
		u.Name, u.Email, hashedPassword, u.AccessLevel, &u.UserActive).Scan(&newId)

	if err != nil {
		return 0, err
	}

	return newId, nil
}

// UpdateUser updates a user by id
func (repo *mysqlDBRepo) UpdateUser(u models.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `UPDATE users SET name = $1, user_active = $2, email = $3, access_level = $4, updated_at = $5 WHERE id = $7`

	_, err := repo.DB.ExecContext(ctx, stmt, u.Name, u.UserActive, u.Email, u.AccessLevel, u.UpdatedAt, u.ID)

	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// DeleteUser sets a user to deleted by populating deleted_at value
func (repo *mysqlDBRepo) DeleteUser(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `UPDATE users set deleted_at = $1, user_active = 0 WHERE id = $2`

	_, err := repo.DB.ExecContext(ctx, stmt, time.Now(), id)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// UpdatePassword resets a password
func (repo *mysqlDBRepo) UpdatePassword(id int, newPassword string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		log.Println(err)
		return err
	}

	stmt := `UPDATE users SET password = $1 WHERE id = $2`

	_, err = repo.DB.ExecContext(ctx, stmt, hashedPassword, id)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// Authenticate authenticates
func (repo *mysqlDBRepo) Authenticate(email, testPassword string) (int, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var id int
	var hashedPassword string
	var userActive int

	query := `SELECT id, password, user_active FROM users WHERE email = $1 AND deleted_at IS NULL`

	row := repo.DB.QueryRowContext(ctx, query, email)
	err := row.Scan(&id, &hashedPassword, &userActive)

	if err == sql.ErrNoRows {
		return 0, "", models.ErrInvalidCredentials
	} else if err != nil {
		log.Println(err)
		return 0, "", err
	}

	if userActive == 0 {
		return 0, "", models.ErrInactiveAccount
	}

	return id, hashedPassword, nil
}

// AllUsers returns all user
func (repo *mysqlDBRepo) AllUsers() ([]*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `SELECT id, name, email, user_active, created_at, updated_at FROM users WHERE deleted_at IS NULL`

	rows, err := repo.DB.QueryContext(ctx, stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User

	for rows.Next() {
		s := &models.User{}
		err = rows.Scan(&s.ID, &s.Name, &s.Email, &s.UserActive, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			return nil, err
		}

		users = append(users, s)
	}

	if err = rows.Err(); err != nil {
		log.Println(err)
		return nil, err
	}

	return users, nil
}

// InsertRememberMeToken inserts a remember me token into remember_tokens for a user
func (repo *mysqlDBRepo) InsertRememberMeToken(id int, token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := "INSERT INTO remember_tokens (user_id, remember_token) VALUES ($1, $2)"

	_, err := repo.DB.ExecContext(ctx, stmt, id, token)

	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// DeleteToken deletes a remember me token
func (repo *mysqlDBRepo) DeleteToken(token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `DELETE FROM remember_tokens WHERE remember_token = $1`
	_, err := repo.DB.ExecContext(ctx, stmt, token)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// CheckForToken checks for a valid remember me token
func (repo *mysqlDBRepo) CheckForToken(id int, token string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `SELECT id FROM remember_tokens WHERE user_id = $1 AND remember_token = $2`

	row := repo.DB.QueryRowContext(ctx, stmt, id, token)
	err := row.Scan(&id)

	return err == nil
}
