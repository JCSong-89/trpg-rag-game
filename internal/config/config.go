package config

import (
	"github.com/JCSong-89/trpg-rag-game/pkg/types"
	"os"
)

func LoadConfig() types.Config {
	neo4jURI := os.Getenv("NEO4J_URL")
	neo4jUser := os.Getenv("NEO4J_USER")
	neo4jPass := os.Getenv("NEO4J_PASSWORD")
	quadrantURI := os.Getenv("QUADRANT_URL")
	serverPort := os.Getenv("SERVER_PORT")

	dbConfig := types.DbConfig{
		Neo4jUrl:    neo4jURI,
		Neo4jUser:   neo4jUser,
		Neo4jPass:   neo4jPass,
		QuadrantUrI: quadrantURI,
	}

	return types.Config{
		ServerPort: serverPort,
		Db:         dbConfig,
	}
}