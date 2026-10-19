package main

import (
	"context"
	"log"
	"os"

	"autojurnal/internal/config"
	"autojurnal/internal/module/master/users"

	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

// Seeder idempotent: data yang sudah ada tidak ditimpa.
func main() {
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
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return seedAdminUser(ctx, tx)
	})
	if err != nil {
		log.Fatalf("seed: %v", err)
	}
	log.Println("seed done")
}

func seedAdminUser(ctx context.Context, tx bun.Tx) error {
	username := os.Getenv("SEED_ADMIN_USERNAME")
	password := os.Getenv("SEED_ADMIN_PASSWORD")
	if username == "" || password == "" {
		log.Fatal("SEED_ADMIN_USERNAME and SEED_ADMIN_PASSWORD are required")
	}

	exists, err := tx.NewSelect().
		Model((*users.User)(nil)).
		Where("lower(username) = lower(?)", username).
		Exists(ctx)
	if err != nil {
		return err
	}
	if exists {
		log.Printf("user %q already exists, skipped", username)
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &users.User{
		Username:     username,
		Name:         "Administrator",
		PasswordHash: string(hash),
	}
	if _, err := tx.NewInsert().Model(user).Returning("id").Exec(ctx); err != nil {
		return err
	}
	log.Printf("user %q created (id=%s)", username, user.ID)
	return nil
}
