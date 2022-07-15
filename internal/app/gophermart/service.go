package gophermart

import (
	"context"

	martMiddleware "github.com/Asymmetriq/gophermart/internal/app/gophermart/middleware"
	"github.com/Asymmetriq/gophermart/internal/config"
	"github.com/Asymmetriq/gophermart/internal/pkg/accrual"
	repo "github.com/Asymmetriq/gophermart/internal/pkg/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewGophermart(ctx context.Context, cfg config.Config, repo repo.Repository, client accrual.Client) *Service {
	s := &Service{
		Mux:           chi.NewMux(),
		Storage:       repo,
		Config:        cfg,
		AccrualClient: client,
	}

	s.Use(
		middleware.Recoverer,
		middleware.RealIP,
		middleware.Logger,

		martMiddleware.Gzip,
	)
	s.Route("/", func(r chi.Router) {
		r.Route("/api", func(r chi.Router) {
			r.Route("/user", func(r chi.Router) {
				r.Group(func(r chi.Router) {
					// Check token
					r.Use(martMiddleware.TokenValidation(s.Config.GetTokenSignKey()))

					// Orders
					r.Route("/orders", func(r chi.Router) {
						r.Post("/", s.processOrderHandler)
						r.Get("/", s.getOrdersHandler)
					})
					// Balance
					r.Get("/withdrawals", s.getWithdrawalsHandler)
					r.Route("/balance", func(r chi.Router) {
						r.Get("/", s.getBalanceHandler)
						r.Post("/withdraw", s.withdrawHandler)
					})
				})
				r.Group(func(r chi.Router) {
					// Check user credentials
					r.Use(martMiddleware.UserValidation)

					r.Post("/register", s.registerHandler)
					r.Post("/login", s.loginHandler)
				})

			})
		})
	})
	s.updateOrdersBackground(ctx)
	return s
}

type Service struct {
	*chi.Mux
	Storage       repo.Repository
	Config        config.Config
	AccrualClient accrual.Client
}
