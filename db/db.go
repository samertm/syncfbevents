package db

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/samertm/syncfbevents/conf"
)

var DB *sqlx.DB = sqlx.MustConnect("postgres", fmt.Sprintf("sslmode=disable dbname=%s user=%s", conf.Config.PGDATABASE, conf.Config.PGUSER))

type Binder struct {
	Len   int
	Items []interface{}
}

// Returns "$b.Len".
func (b *Binder) Bind(i interface{}) string {
	b.Items = append(b.Items, i)
	b.Len++
	return fmt.Sprintf("$%d", b.Len)
}
