package database

import "database/sql"

func queryList[T any](db *sql.DB, query string, args []any, scan func(*sql.Rows) (T, error)) ([]T, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []T
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}
