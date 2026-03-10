package http

import "github.com/benlocal/cli-manager/pkg/db"

type RegistryContext struct {
	database *db.DB
}

func NewRegistryContext(database *db.DB) *RegistryContext {
	return &RegistryContext{
		database: database,
	}
}

func (c *RegistryContext) Database() *db.DB {
	return c.database
}
