package mysql

import (
	"database/sql"
	"strconv"
)

type Queries struct{ DB *sql.DB }

func parseID(v string) (uint64, error) { return strconv.ParseUint(v, 10, 63) }
