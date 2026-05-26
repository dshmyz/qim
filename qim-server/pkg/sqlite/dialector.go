package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"

	_ "modernc.org/sqlite"
)

type Dialector struct {
	DSN string
}

func Open(dsn string) Dialector {
	return Dialector{DSN: dsn}
}

func (d Dialector) Name() string {
	return "sqlite"
}

func (d Dialector) Initialize(db *gorm.DB) error {
	sqlDB, err := sql.Open("sqlite", d.DSN)
	if err != nil {
		return err
	}

	_, err = sqlDB.Exec("PRAGMA synchronous = FULL")
	if err != nil {
		return err
	}

	db.ConnPool = sqlDB
	db.SkipDefaultTransaction = true
	db.FullSaveAssociations = false

	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{
		CreateClauses:        []string{"INSERT", "VALUES", "ON CONFLICT"},
		UpdateClauses:        []string{"UPDATE", "SET", "WHERE", "ORDER BY"},
		DeleteClauses:        []string{"DELETE", "FROM", "WHERE"},
		LastInsertIDReversed: true,
	})
	return nil
}

func (d Dialector) DataTypeOf(field *schema.Field) string {
	switch field.DataType {
	case schema.Bool:
		return "numeric"
	case schema.Int, schema.Uint:
		return "integer"
	case schema.Float:
		if field.Precision > 0 {
			return fmt.Sprintf("real(%d,%d)", field.Precision, field.Scale)
		}
		return "real"
	case schema.String:
		if field.Size > 0 {
			return fmt.Sprintf("varchar(%d)", field.Size)
		}
		return "text"
	case schema.Time:
		return "datetime"
	case schema.Bytes:
		return "blob"
	}

	switch field.DataType {
	case "json":
		return "text"
	case "text", "varchar", "char", "tinytext", "longtext", "mediumtext":
		if field.Size > 0 {
			return fmt.Sprintf("varchar(%d)", field.Size)
		}
		return string(field.DataType)
	}
	return string(field.DataType)
}

func (d Dialector) DefaultValueOf(field *schema.Field) clause.Expression {
	if field.AutoIncrement {
		return clause.Expr{SQL: "NULL"}
	}
	return clause.Expr{SQL: "DEFAULT"}
}

func (d Dialector) Migrator(db *gorm.DB) gorm.Migrator {
	return &migrator.Migrator{
		Config: migrator.Config{
			DB:                          db,
			Dialector:                   d,
			CreateIndexAfterCreateTable: true,
		},
	}
}

func (d Dialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {
	writer.WriteByte('?')
}

func (d Dialector) QuoteTo(writer clause.Writer, str string) {
	writer.WriteByte('`')
	if strings.Contains(str, ".") {
		for idx, str := range strings.Split(str, ".") {
			if idx > 0 {
				writer.WriteString(".`")
			}
			writer.WriteString(str)
			writer.WriteByte('`')
		}
	} else {
		writer.WriteString(str)
		writer.WriteByte('`')
	}
}

func (d Dialector) Explain(sql string, vars ...interface{}) string {
	return logger.ExplainSQL(sql, nil, `"`, vars...)
}

var _ gorm.Dialector = Dialector{}
