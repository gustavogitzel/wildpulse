module wildpulse/apps/api

go 1.26

require (
	github.com/go-chi/chi/v5 v5.0.12
	github.com/go-chi/cors v1.2.1
	github.com/jackc/pgx/v5 v5.5.5
	wildpulse/pkg v0.0.0
)

replace wildpulse/pkg => ../../pkg
