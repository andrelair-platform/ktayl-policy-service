// Package db embeds the SQL migration files so they are included in the binary.
package db

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS
