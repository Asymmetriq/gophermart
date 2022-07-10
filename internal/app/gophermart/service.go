package gophermart

import (
	martMiddleware "github.com/Asymmetriq/gophermart/internal/app/gophermart/middleware"
	"github.com/Asymmetriq/gophermart/internal/config"
	repo "github.com/Asymmetriq/gophermart/internal/pkg/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewGophermart(repo repo.Repository, cfg config.Config) *Service {
	s := &Service{
		Mux:     chi.NewMux(),
		Storage: repo,
		Config:  cfg,
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
					r.Post("/orders", s.processOrderHandler)
					r.Get("/orders", s.asyncDeleteHandler)

					// Balance
					r.Route("/balance", func(r chi.Router) {
						r.Get("/", s.jsonHandler)
						r.Post("/withdraw", s.jsonHandler)
						r.Get("/withdrawals", s.jsonHandler)
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

	return s
}

type Service struct {
	*chi.Mux
	Storage repo.Repository
	Config  config.Config
}
