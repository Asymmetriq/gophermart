package main

import (
	"context"
	"net/http"

	"github.com/Asymmetriq/gophermart/internal/app/gophermart"
	"github.com/Asymmetriq/gophermart/internal/config"
	"github.com/Asymmetriq/gophermart/internal/pkg/accrual"
	"github.com/Asymmetriq/gophermart/internal/pkg/database"
	"github.com/Asymmetriq/gophermart/internal/pkg/repository"
	_ "github.com/jackc/pgx/v4/stdlib"
)

const driverName = "pgx"

func main() {
	cfg := config.InitConfig()
	ctx := context.Background()
	db := database.ConnectToDatabase(driverName, cfg.GetDatabaseURI())

	service := gophermart.NewGophermart(
		ctx,
		cfg,
		repository.NewRepository(cfg, db),
		accrual.NewlClient(cfg.GetAccrualAddress()))

	http.ListenAndServe(service.Config.GetRunAddress(), service)
}
