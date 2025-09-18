package resources

import (
	"fmt"
	"strconv"

	"dario.cat/mergo"
	postgres "github.com/ProRocketeers/yoke-chart/resources/postgresql"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/ptr"
)

func CreateDB(values DeploymentValues) (bool, ResourceCreator) {
	return values.DB != nil && *values.DB.Enabled, func(values DeploymentValues) ([]unstructured.Unstructured, error) {
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
			EnableLogicalBackup: db.Backup != nil && *db.Backup,
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
				return []unstructured.Unstructured{}, fmt.Errorf("error while merging additional DB config: %v", err)
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
		u, err := toUnstructured(&postgres)
		if err != nil {
			return []unstructured.Unstructured{}, err
		}
		return u, nil
	}
}
