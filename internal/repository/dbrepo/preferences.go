package dbrepo

import (
	"context"
	"log"
	"server_monitor/internal/models"
	"time"
)

func (repo *mysqlDBRepo) AllPreferences() ([]models.Preference, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := "SELECT id, name, preference FROM preferences"

	rows, err := repo.DB.QueryContext(ctx, stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var preferences []models.Preference

	for rows.Next() {
		s := &models.Preference{}
		err = rows.Scan(&s.ID, &s.Name, &s.Preference)
		if err != nil {
			return nil, err
		}
		preferences = append(preferences, *s)
	}

	if err = rows.Err(); err != nil {
		log.Println(err)
		return nil, err
	}
	return preferences, nil
}

func (repo *mysqlDBRepo) SetSystemPref(name, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `delete from preferences where name = $1`
	_, _ = repo.DB.ExecContext(ctx, stmt, name)

	query := `INSERT INTO preferences (
				name, preference, created_at, updated_at) 
				VALUES ($1, $2, $3, $4)`
	_, err := repo.DB.ExecContext(ctx, query, name, value, time.Now(), time.Now())

	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (repo *mysqlDBRepo) InsertOrUpdateSitePreferences(pm map[string]string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	for k, v := range pm {
		query := `DELETE FROM preferences WHERE name = $1`
		_, err := repo.DB.ExecContext(ctx, query, k)
		if err != nil {
			return err
		}

		query = `INSERT INTO preferences (name, preference, created_at, updated_at)
					VALUES ($1, $2, $3, $4)`

		_, err = repo.DB.ExecContext(ctx, query, k, v, time.Now(), time.Now())
		if err != nil {
			return err
		}
	}
	return nil
}
