module wildpulse/apps/workers

go 1.26

require (
	github.com/jackc/pgx/v5 v5.5.5
	github.com/robfig/cron/v3 v3.0.1
	wildpulse/pkg v0.0.0
)

replace wildpulse/pkg => ../../pkg
