package types

type DbConfig struct {
	Neo4jUrl    string
	Neo4jUser   string
	Neo4jPass   string
	QuadrantUrI string
}

type Config struct {
	ServerPort string
	Db         DbConfig
}