package database

import (
	"github.com/knadh/koanf/v2"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stephenafamo/bob"
)

func NewDatabase(conf *koanf.Koanf) (bob.DB, error) {
	return bob.Open(conf.MustString("database.driver"), conf.MustString("database.url"))
}
