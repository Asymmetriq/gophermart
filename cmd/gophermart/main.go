package main

import (
	"net/http"

	"github.com/Asymmetriq/gophermart/internal/app/gophermart"
	"github.com/Asymmetriq/gophermart/internal/config"
	"github.com/Asymmetriq/gophermart/internal/pkg/database"
	"github.com/Asymmetriq/gophermart/internal/pkg/repository"
	_ "github.com/jackc/pgx/v4/stdlib"
)

const driverName = "pgx"

func main() {
	cfg := config.InitConfig()
	db := database.ConnectToDatabase(driverName, cfg.GetDatabaseDSN())

	service := gophermart.NewGophermart(repository.NewRepository(cfg, db), cfg)
	http.ListenAndServe(service.Config.GetAddress(), service)
}
