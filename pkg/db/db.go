package db

import (
	"database/sql"
	"fmt"
	"os"
	"runtime"
	"sync"

	// Driver for database/sql package
	_ "github.com/go-sql-driver/mysql"
	migrate "github.com/rubenv/sql-migrate"
)

// DB is main class for MySQL API
type DB struct {
	con   *sql.DB
	mutex *sync.RWMutex
}

func dbDestructor(db *DB) {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	if db.con != nil {
		db.con.Close()
		db.con = nil
	}
}

type DSN struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

// New is function for construct a new DB connection
func New(dsn *DSN) (*DB, error) {
	connString, err := getConnectionString(dsn)
	if err != nil {
		return nil, err
	}
	con, err := sql.Open("mysql", connString)
	if err != nil {
		return nil, err
	}
	con.SetMaxIdleConns(3)
	con.SetMaxOpenConns(256)

	db := &DB{con: con, mutex: &sync.RWMutex{}}
	runtime.SetFinalizer(db, dbDestructor)

	return db, nil
}

func getConnectionString(dsn *DSN) (string, error) {
	if dsn == nil {
		var err error
		dsn, err = getDSNFromEnvVars()
		if err != nil {
			return "", fmt.Errorf("failed to get the DSN from the environment variables: %w", err)
		}
	}
	return fmt.Sprintf(
		"%s:%s@%s/%s?parseTime=true",
		dsn.User,
		dsn.Password,
		fmt.Sprintf("tcp(%s:%s)", dsn.Host, dsn.Port),
		dsn.DBName,
	), nil
}

func getDSNFromEnvVars() (*DSN, error) {
	dsn := &DSN{}
	var ok bool
	genErr := func(missingEnvVar string) error {
		return fmt.Errorf("environment variable %s is undefined", missingEnvVar)
	}
	const (
		envVarDBUser = "DB_USER"
		envVarDBPass = "DB_PASS"
		envVarDBHost = "DB_HOST"
		envVarDBPort = "DB_PORT"
		envVarDBName = "DB_NAME"
	)
	if dsn.User, ok = os.LookupEnv(envVarDBUser); !ok {
		return nil, genErr(envVarDBUser)
	}
	if dsn.Password, ok = os.LookupEnv(envVarDBPass); !ok {
		return nil, genErr(envVarDBPass)
	}
	if dsn.Host, ok = os.LookupEnv(envVarDBHost); !ok {
		return nil, genErr(envVarDBHost)
	}
	if dsn.Port, ok = os.LookupEnv(envVarDBPort); !ok {
		return nil, genErr(envVarDBPort)
	}
	if dsn.DBName, ok = os.LookupEnv(envVarDBName); !ok {
		return nil, genErr(envVarDBName)
	}
	return dsn, nil
}

var (
	errDatabaseConnectionReleased error = fmt.Errorf("database connection is already released")
)

// Query is function for execute select query type
func (db *DB) Query(query string, args ...interface{}) (result []map[string]string, err error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	if db.con == nil {
		err = errDatabaseConnectionReleased
		return
	}

	result = make([]map[string]string, 0)
	rows, err := db.con.Query(query, args...)
	if err != nil {
		return
	}

	defer rows.Close()
	columnNames, err := rows.Columns()
	if err != nil {
		return
	}

	vals := make([]interface{}, len(columnNames))
	for rows.Next() {
		for i := range columnNames {
			vals[i] = &vals[i]
		}
		err = rows.Scan(vals...)
		if err != nil {
			return
		}
		var row = make(map[string]string)
		for i := range columnNames {
			switch vals[i].(type) {
			case int, int64:
				row[columnNames[i]] = fmt.Sprintf("%d", vals[i])
			case float32, float64:
				row[columnNames[i]] = fmt.Sprintf("%f", vals[i])
			case nil:
				row[columnNames[i]] = ""
			default:
				row[columnNames[i]] = fmt.Sprintf("%s", vals[i])
			}
		}
		result = append(result, row)
	}

	return
}

// Exec is function for execute update and delete query type
func (db *DB) Exec(query string, args ...interface{}) (result sql.Result, err error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	if db.con == nil {
		err = errDatabaseConnectionReleased
		return
	}

	result, err = db.con.Exec(query, args...)

	return
}

// MigrateUp is function that receives path to migration dir and runs up ones
func (db *DB) MigrateUp(path string) (err error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	if db.con == nil {
		err = errDatabaseConnectionReleased
		return
	}

	migrations := &migrate.FileMigrationSource{
		Dir: path,
	}
	_, err = migrate.Exec(db.con, "mysql", migrations, migrate.Up)

	return
}

// MigrateDown is function that receives path to migration dir and runs down ones
func (db *DB) MigrateDown(path string) (err error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	if db.con == nil {
		err = errDatabaseConnectionReleased
		return
	}

	migrations := &migrate.FileMigrationSource{
		Dir: path,
	}
	_, err = migrate.Exec(db.con, "mysql", migrations, migrate.Down)

	return
}
