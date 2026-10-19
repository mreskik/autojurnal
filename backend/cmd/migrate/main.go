package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"autojurnal/internal/config"
	"autojurnal/migration"

	"github.com/uptrace/bun/migrate"
)

const usage = `usage: go run ./cmd/migrate <command>

commands:
  up                 jalankan semua migration yang belum dijalankan
  down -confirm      rollback 1 group migration terakhir
  status             lihat status migration
  create <nama>      buat file .tx.up.sql & .tx.down.sql baru
  unlock             lepas lock kalau proses sebelumnya crash`

func main() {
	if len(os.Args) < 2 {
		fmt.Println(usage)
		os.Exit(1)
	}
	cmd, args := os.Args[1], os.Args[2:]

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	db, err := config.NewDatabase(cfg.DB)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	m := migrate.NewMigrator(db, migration.Migrations)

	// bikin tabel bun_migrations & bun_migration_locks kalau belum ada
	if err := m.Init(ctx); err != nil {
		log.Fatalf("init: %v", err)
	}

	switch cmd {
	case "up":
		withLock(ctx, m, func() error {
			group, err := m.Migrate(ctx)
			if err != nil {
				return err
			}
			if group.IsZero() {
				fmt.Println("nothing to migrate, database up to date")
				return nil
			}
			fmt.Printf("migrated to %s\n", group)
			return nil
		})

	case "down":
		fs := flag.NewFlagSet("down", flag.ExitOnError)
		confirm := fs.Bool("confirm", false, "konfirmasi rollback")
		_ = fs.Parse(args)
		if !*confirm {
			log.Fatal("rollback bisa menghapus data, jalankan ulang dengan: down -confirm")
		}
		withLock(ctx, m, func() error {
			group, err := m.Rollback(ctx)
			if err != nil {
				return err
			}
			if group.IsZero() {
				fmt.Println("nothing to rollback")
				return nil
			}
			fmt.Printf("rolled back %s\n", group)
			return nil
		})

	case "status":
		ms, err := m.MigrationsWithStatus(ctx)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("migrations: %s\n", ms)
		fmt.Printf("unapplied:  %s\n", ms.Unapplied())
		fmt.Printf("last group: %s\n", ms.LastGroup())

	case "create":
		if len(args) == 0 {
			log.Fatal("nama migration wajib diisi, contoh: create add_trades_table")
		}
		name := strings.Join(args, "_")
		files, err := m.CreateTxSQLMigrations(ctx, name)
		if err != nil {
			log.Fatal(err)
		}
		for _, f := range files {
			fmt.Printf("created %s\n", f.Path)
		}

	case "unlock":
		if err := m.Unlock(ctx); err != nil {
			log.Fatal(err)
		}
		fmt.Println("unlocked")

	default:
		fmt.Println(usage)
		os.Exit(1)
	}
}

// withLock mencegah dua proses migrate jalan bersamaan.
func withLock(ctx context.Context, m *migrate.Migrator, fn func() error) {
	if err := m.Lock(ctx); err != nil {
		log.Fatalf("lock: %v (jalankan 'unlock' kalau yakin tidak ada proses lain)", err)
	}
	err := fn()
	if uerr := m.Unlock(ctx); uerr != nil {
		log.Printf("unlock: %v", uerr)
	}
	if err != nil {
		log.Fatal(err)
	}
}
