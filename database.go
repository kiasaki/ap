package ap

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Database struct {
	url   string
	debug bool
	conn  *sqlx.DB
}

func NewDatabase() *Database {
	return &Database{}
}

func (db *Database) SetURL(url string) {
	db.url = url
}

func (db *Database) SetDebug(debug bool) {
	db.debug = debug
}

func (db *Database) Connect() error {
	sourceName, err := pq.ParseURL(db.url)
	if err != nil {
		panic(fmt.Sprintf("database postgres: %v", err))
	}
	db.conn, err = sqlx.Connect("postgres", sourceName)
	if err != nil {
		return err
	}
	db.conn = db.conn.Unsafe()
	return nil
}

func (db *Database) Save(e interface{}) error {
	value := reflect.ValueOf(e)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return fmt.Errorf("Database.Save expects a non-nil pointer")
	}
	structValue := value.Elem()
	if structValue.Kind() != reflect.Struct {
		return fmt.Errorf("Database.Save expects a pointer to struct")
	}
	tableName := ToSnakeCase(value.Type().Elem().Name()) + "s"
	if strings.HasSuffix(tableName, "ys") {
		tableName = tableName[:len(tableName)-2] + "ies"
	}

	fields := []string{}
	placeholders := []string{}
	values := []interface{}{}
	for i := 0; i < structValue.NumField(); i++ {
		field := structValue.Type().Field(i)
		fieldValue := structValue.Field(i)
		if field.Name == "ID" {
			if fieldValue.Kind() == reflect.String && fieldValue.String() == "" && fieldValue.CanSet() {
				fieldValue.SetString(UUID())
			}
		}
		if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
			timeValue := fieldValue.Interface().(time.Time)
			if timeValue.IsZero() && fieldValue.CanSet() {
				fieldValue.Set(reflect.ValueOf(time.Now()))
			}
		}
		fields = append(fields, `"`+ToSnakeCase(field.Name)+`"`)
		placeholders = append(placeholders, "$"+strconv.Itoa(i+1))
		values = append(values, fieldValue.Interface())
	}
	updates := make([]string, 0, len(fields))
	for _, field := range fields {
		if field == `"id"` {
			continue
		}
		updates = append(updates, fmt.Sprintf(`%s = excluded.%s`, field, field))
	}
	updateSet := strings.Join(updates, ", ")
	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(id) DO UPDATE SET %s",
		tableName,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
		updateSet,
	)
	return db.Exec(sql, values...)
}

func (db *Database) Where(e interface{}, whereSQL string, args ...interface{}) error {
	value := reflect.ValueOf(e)
	slice := false
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return fmt.Errorf("Database.Where expects a non-nil pointer")
	}
	entityType := value.Type().Elem()
	for entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}
	if entityType.Kind() == reflect.Slice || entityType.Kind() == reflect.Array {
		slice = true
		entityType = entityType.Elem()
		for entityType.Kind() == reflect.Ptr {
			entityType = entityType.Elem()
		}
	}
	if entityType.Kind() != reflect.Struct {
		return fmt.Errorf("Database.Where expects a pointer to struct or slice of structs")
	}

	tableName := ToSnakeCase(entityType.Name()) + "s"
	query := fmt.Sprintf("select * from %s", tableName)
	if whereSQL != "" {
		query = query + " where " + whereSQL
	}

	if slice {
		return db.Query(e, query, args...)
	}
	return db.First(e, query, args...)
}

func (db *Database) Exec(sql string, args ...interface{}) error {
	if db.debug {
		log.Printf("database exec: %s: %v\n", sql, args)
	}
	_, err := db.conn.Exec(sql, args...)
	return err
}

func (db *Database) First(dest interface{}, query string, args ...interface{}) error {
	if db.debug {
		log.Printf("database first: %s: %v\n", query, args)
	}
	err := db.conn.Get(dest, query, args...)
	if err == sql.ErrNoRows {
		return nil
	}
	return err
}

func (db *Database) Query(dest interface{}, query string, args ...interface{}) error {
	if db.debug {
		log.Printf("database query: %s: %v\n", query, args)
	}
	return db.conn.Select(dest, query, args...)
}
