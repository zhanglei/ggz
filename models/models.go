package models

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/go-ggz/ggz/config"

	// Needed for the MySQL driver
	_ "github.com/go-sql-driver/mysql"

	// Needed for the Postgresql driver
	_ "github.com/lib/pq"

	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"github.com/go-xorm/xorm/migrate"
)

var (
	x      *xorm.Engine
	tables []interface{}
	// EnableSQLite3 for enable sqlite 3
	EnableSQLite3 bool

	// Migrations for db migrate
	migrations = []*migrate.Migration{
		{
			ID: "201709201400",
			Migrate: func(tx *xorm.Engine) error {
				return tx.Sync2(&User{})
			},
			Rollback: func(tx *xorm.Engine) error {
				return tx.DropTables(&User{})
			},
		},
	}
)

func init() {
	tables = append(tables,
		new(User),
		new(Redirect),
	)

	gonicNames := []string{"SSL", "UID"}
	for _, name := range gonicNames {
		core.LintGonicMapper[name] = true
	}
}

func getEngine() (*xorm.Engine, error) {
	connStr := ""
	var Param = "?"
	if strings.Contains(config.Database.Name, Param) {
		Param = "&"
	}
	switch config.Database.Driver {
	case "mysql":
		if config.Database.Host[0] == '/' { // looks like a unix socket
			connStr = fmt.Sprintf("%s:%s@unix(%s)/%s%scharset=utf8&parseTime=true",
				config.Database.Username, config.Database.Password, config.Database.Host, config.Database.Name, Param)
		} else {
			connStr = fmt.Sprintf("%s:%s@tcp(%s)/%s%scharset=utf8&parseTime=true",
				config.Database.Username, config.Database.Password, config.Database.Host, config.Database.Name, Param)
		}
	case "postgres":
		host, port := parsePostgreSQLHostPort(config.Database.Host)
		if host[0] == '/' { // looks like a unix socket
			connStr = fmt.Sprintf("postgres://%s:%s@:%s/%s%ssslmode=%s&host=%s",
				url.QueryEscape(config.Database.Username), url.QueryEscape(config.Database.Password), port, config.Database.Name, Param, config.Database.SSLMode, host)
		} else {
			connStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s%ssslmode=%s",
				url.QueryEscape(config.Database.Username), url.QueryEscape(config.Database.Password), host, port, config.Database.Name, Param, config.Database.SSLMode)
		}
	case "sqlite3":
		if !EnableSQLite3 {
			return nil, errors.New("this binary version does not build support for SQLite3")
		}
		if err := os.MkdirAll(path.Dir(config.Database.Path), os.ModePerm); err != nil {
			return nil, fmt.Errorf("Failed to create directories: %v", err)
		}
		connStr = fmt.Sprintf("file:%s?cache=shared&mode=rwc&_busy_timeout=%d", config.Database.Path, config.Database.TimeOut)
	default:
		return nil, fmt.Errorf("Unknown database type: %s", config.Database.Driver)
	}

	return xorm.NewEngine(config.Database.Driver, connStr)
}

// parsePostgreSQLHostPort parses given input in various forms defined in
// https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING
// and returns proper host and port number.
func parsePostgreSQLHostPort(info string) (string, string) {
	host, port := "127.0.0.1", "5432"
	if strings.Contains(info, ":") && !strings.HasSuffix(info, "]") {
		idx := strings.LastIndex(info, ":")
		host = info[:idx]
		port = info[idx+1:]
	} else if len(info) > 0 {
		host = info
	}
	return host, port
}

// SetEngine sets the xorm.Engine
func SetEngine() (err error) {
	x, err = getEngine()
	if err != nil {
		return fmt.Errorf("Failed to connect to database: %v", err)
	}

	x.SetMapper(core.GonicMapper{})
	// WARNING: for serv command, MUST remove the output to os.stdout,
	// so use log file to instead print to stdout.
	// x.SetLogger(log.XORMLogger)
	x.ShowSQL(true)
	return nil
}

// NewEngine initializes a new xorm.Engine
func NewEngine() (err error) {
	if err = SetEngine(); err != nil {
		return err
	}

	if err = x.Ping(); err != nil {
		return err
	}

	m := migrate.New(x, migrate.DefaultOptions, migrations)
	if err = m.Migrate(); err != nil {
		return fmt.Errorf("migrate: %v", err)
	}

	if err = x.StoreEngine("InnoDB").Sync2(tables...); err != nil {
		return fmt.Errorf("sync database struct error: %v", err)
	}

	return nil
}
