// Package migrations embeds the goose baseline (F1).
//
// Same tables Prisma owns, IF NOT EXISTS — safe on Prisma-managed databases,
// bootstrap on empty ones. Prisma remains the migration owner.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS

// Dir is the goose migrations directory inside FS.
const Dir = "."
