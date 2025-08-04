package db

import (
	"github.com/JCSong-89/trpg-rag-game/pkg/types"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"log"
)

func NewNeo4jDriver(cfg types.Config) neo4j.DriverWithContext {
	driver, err := neo4j.NewDriverWithContext(cfg.Db.Neo4jUrl, neo4j.BasicAuth(cfg.Db.Neo4jUser, cfg.Db.Neo4jPass, ""))
	if err != nil {
		log.Fatalf("Failed to create Neo4j driver: %v", err)
	}
	return driver
}