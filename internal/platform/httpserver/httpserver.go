package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	authpkg "github.com/hopeIsCo0l/anu-pro/internal/platform/auth"
	appcfg "github.com/hopeIsCo0l/anu-pro/internal/platform/config"
	"github.com/hopeIsCo0l/anu-pro/internal/platform/storage"
	"github.com/hopeIsCo0l/anu-pro/internal/platform/tenant"
	"github.com/hopeIsCo0l/anu-pro/internal/inventory"
	"github.com/hopeIsCo0l/anu-pro/internal/recipe"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r *chi.Mux, pool *pgxpool.Pool, store storage.Store, cfg *appcfg.Config) {
	authSvc := authpkg.NewService(pool, cfg)
	authHandler := authpkg.NewHandler(authSvc)

	r.Route("/api", func(r chi.Router) {
		// Public auth endpoints
		r.Post("/auth/signup", authHandler.Signup)
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.Refresh)
		r.Post("/auth/logout", authHandler.Logout)

		// Protected endpoints — JWT required
		r.Group(func(r chi.Router) {
			r.Use(tenant.RequireAuth(cfg))

			r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
				ctx := r.Context()
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]string{
					"user_id":     tenant.UserIDFromContext(ctx),
					"tenant_id":   tenant.TenantIDFromContext(ctx),
					"tenant_slug": tenant.TenantSlugFromContext(ctx),
					"role":        tenant.RoleFromContext(ctx),
				})
			})

			// Inventory
			inv := inventory.NewHandler(inventory.NewService(pool))
			r.Route("/items", func(r chi.Router) {
				r.Get("/", inv.ListItems)
				r.Post("/", inv.CreateItem)
				r.Get("/{id}", inv.GetItem)
				r.Patch("/{id}", inv.UpdateItem)
				r.Delete("/{id}", inv.DeactivateItem)
			})
			r.Route("/locations", func(r chi.Router) {
				r.Get("/", inv.ListLocations)
				r.Post("/", inv.CreateLocation)
				r.Get("/{id}", inv.GetLocation)
			})
			r.Route("/lots", func(r chi.Router) {
				r.Get("/", inv.ListLots)
				r.Post("/", inv.CreateLot)
				r.Get("/{id}", inv.GetLot)
			})
			r.Route("/stock", func(r chi.Router) {
				r.Get("/current", inv.GetCurrentStock)
				r.Get("/movements", inv.ListMovements)
				r.Post("/movements", inv.RecordMovement)
				r.Post("/transfers", inv.Transfer)
			})
			r.Post("/import/items", inv.ImportItems)

			// Recipes
			rec := recipe.NewHandler(recipe.NewService(pool))
			r.Route("/recipes", func(r chi.Router) {
				r.Get("/", rec.ListRecipes)
				r.Post("/", rec.CreateRecipe)
				r.Get("/{id}", rec.GetRecipe)
				r.Post("/{id}/versions", rec.AddVersion)
				r.Get("/{id}/versions/{vid}", rec.GetVersion)
			})
		})
	})
}
