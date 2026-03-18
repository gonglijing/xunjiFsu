package database

import "database/sql"

func queryList[T any](db *sql.DB, query string, args []any, scan func(*sql.Rows) (T, error)) ([]T, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]T, 0, estimateQueryListCap(args))
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

func estimateQueryListCap(args []any) int {
	if len(args) == 0 {
		return 0
	}
	switch v := args[len(args)-1].(type) {
	case int:
		if v > 0 && v <= 4096 {
			return v
		}
	case int32:
		if v > 0 && v <= 4096 {
			return int(v)
		}
	case int64:
		if v > 0 && v <= 4096 {
			return int(v)
		}
	}
	return 0
}
