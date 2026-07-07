package dbinit

import (
	"einvoice-access-point/internal/data/database/postgresql"
	red "einvoice-access-point/internal/data/database/redis"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

var (
	DB  *postgresql.Postgresql
	RDB *red.Redis
)

func InitDB(db *gorm.DB, useSingleton bool) *postgresql.Postgresql {
	if useSingleton {
		if DB == nil {
			DB = postgresql.NewPostgresqlConnection(db)
		}
		return DB
	}

	return postgresql.NewPostgresqlConnection(db)
}

func InitRed(rdb *redis.Client, useSingleton bool) *red.Redis {
	if useSingleton {
		if RDB == nil {
			RDB = red.NewRedisConnection(rdb)
		}
		return RDB
	}

	return red.NewRedisConnection(rdb)
}
