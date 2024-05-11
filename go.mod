module micrified.com/server

replace micrified.com/internal/user => ./internal/user

replace micrified.com/route => ./route

replace micrified.com/route/blog => ./route/blog

replace micrified.com/route/login => ./route/login

replace micrified.com/route/logout => ./route/logout

replace micrified.com/service/auth => ./service/auth

replace micrified.com/service/database => ./service/database

go 1.22.3

require (
	micrified.com/internal/user v0.0.0-00010101000000-000000000000
	micrified.com/route v0.0.0-00010101000000-000000000000
	micrified.com/route/blog v0.0.0-00010101000000-000000000000
	micrified.com/route/login v0.0.0-00010101000000-000000000000
	micrified.com/route/logout v0.0.0-00010101000000-000000000000
	micrified.com/service/auth v0.0.0-00010101000000-000000000000
	micrified.com/service/database v0.0.0-00010101000000-000000000000
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	golang.org/x/crypto v0.23.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
)
