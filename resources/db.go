package resources

import (
	"fmt"
	"strconv"

	"dario.cat/mergo"
	postgres "github.com/ProRocketeers/yoke-chart/resources/postgresql"
	"github.com/yokecd/yoke/pkg/flight"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func CreateDB(values DeploymentValues) (bool, ResourceCreator) {
	return values.DB != nil && *values.DB.Enabled == true, func(values DeploymentValues) ([]flight.Resource, error) {
		db := values.DB

		spec := postgres.PostgresSpec{
			TeamID: values.Metadata.Namespace,
			PostgresqlParam: postgres.PostgresqlParam{
				PgVersion: strconv.Itoa(db.Version),
			},
			NumberOfInstances: int32(db.Replicas),
			Volume: postgres.Volume{
				Size:         db.Size,
				StorageClass: db.StorageClass,
			},
			// false when `Backup` is nil, or the value is false
			EnableLogicalBackup: db.Backup != nil && *db.Backup == true,
			Databases:           db.Databases,
			Users:               map[string]postgres.UserFlags{},
			Resources: &postgres.Resources{
				ResourceRequests: postgres.ResourceDescription{
					CPU:    ptr.To("100m"),
					Memory: ptr.To("100Mi"),
				},
				ResourceLimits: postgres.ResourceDescription{
					CPU:    ptr.To("1"),
					Memory: ptr.To("500Mi"),
				},
			},
		}
		for user, flags := range db.Users {
			spec.Users[user] = flags
		}

		if db.AdditionalConfig != nil {
			if err := mergo.Merge(&spec, *db.AdditionalConfig, mergo.WithOverride); err != nil {
				return []flight.Resource{}, fmt.Errorf("error while merging additional DB config: %v", err)
			}
		}

		postgres := postgres.Postgresql{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "acid.zalan.do/v1",
				Kind:       "postgresql",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      db.ClusterName,
				Namespace: values.Metadata.Namespace,
			},
			Spec: spec,
		}
		return []flight.Resource{&postgres}, nil
	}
}
