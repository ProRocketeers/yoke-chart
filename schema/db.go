package schema

import "github.com/ProRocketeers/yoke-chart/resources/postgresql"

type Database struct {
	Enabled          *bool                           `json:"enabled" validate:"required"`
	ClusterName      string                          `json:"clusterName" validate:"required"`
	Replicas         int                             `json:"replicas" validate:"required"`
	Version          int                             `json:"version" validate:"required"`
	Size             string                          `json:"size" validate:"required"`
	StorageClass     string                          `json:"storageClass" validate:"required"`
	Backup           *bool                           `json:"backup,omitempty"`
	Users            map[string]postgresql.UserFlags `json:"users" validate:"required"`
	Databases        map[string]string               `json:"databases" validate:"required"`
	AdditionalConfig *postgresql.PostgresSpec        `json:"additionalConfig,omitempty"`
}
