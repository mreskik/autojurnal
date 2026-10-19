package migration

import (
	"embed"

	"github.com/uptrace/bun/migrate"
)

// Migrations berisi semua file SQL di folder ini.
// Penamaan: <timestamp>_<nama>.tx.up.sql / .tx.down.sql
// (".tx." = dijalankan di dalam transaction, kalau gagal otomatis rollback).
var Migrations = migrate.NewMigrations()

//go:embed *.sql
var sqlMigrations embed.FS

func init() {
	if err := Migrations.Discover(sqlMigrations); err != nil {
		panic(err)
	}
}
