package schema

import (
	"fmt"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ------------ discriminated unions handling
// ------------ volumes
type VolumeType string

const (
	VolumeTypeStandardTmpfs VolumeType = "tmpfs"
	VolumeTypeStandardLocal VolumeType = "local"
	VolumeTypeRaw           VolumeType = "raw"
	VolumeTypePersistent    VolumeType = "persistent"
	VolumeTypeSecret        VolumeType = "secret"
	VolumeTypeConfigMap     VolumeType = "configMap"
)

type Volume struct {
	Type   VolumeType                 `json:"type" validate:"required"`
	Mounts map[string]VolumeMountList `json:"mounts" validate:"required,dive,dive"`

	Variant VolumeVariant `json:"-"`
}

type VolumeMount struct {
	ContainerPath string  `json:"containerPath" validate:"required"`
	VolumePath    *string `json:"volumePath,omitempty"`

	// OPTIONAL - overrides the type-based default (secret/configMap default to true, everything else to false)
	ReadOnly *bool `json:"readOnly,omitempty"`
	// OPTIONAL - overrides the type-based default (mirrors the current implicit behavior: unset/None for readonly mounts, HostToContainer otherwise)
	MountPropagation *v1.MountPropagationMode `json:"mountPropagation,omitempty" validate:"omitempty,oneof=None HostToContainer Bidirectional"`
}

// VolumeMountList accepts either a single mount object or a sequence of them in YAML,
// so the same volume can be mounted into one container more than once (e.g. at different subPaths)
type VolumeMountList []VolumeMount

func (l *VolumeMountList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var single VolumeMount
	if err := unmarshal(&single); err == nil {
		*l = VolumeMountList{single}
		return nil
	}

	var list []VolumeMount
	if err := unmarshal(&list); err != nil {
		return fmt.Errorf("volume mount must be a mapping or a sequence of mappings: %w", err)
	}
	*l = list
	return nil
}

// Go doesn't have a type union, like `StandardVolume | RawVolume | ...`
// but while putting `interface{}` or `any` works, you could assign anything to that field even if it doesn't make sense
// putting an interface "marks" the struct as being allowed to be used in the field
type VolumeVariant interface {
	// the function can be a noop and not do anything, it just has to be implemented
	// since Go doesn't have any `implements` keyword, and is duck typed instead

	IsVolumeVariant()
}

type StandardVolume struct {
	// type: `tmpfs` or `local`
}

func (StandardVolume) IsVolumeVariant() {}

type RawVolume struct {
	// type: `raw`
	Spec v1.VolumeSource `json:"spec" validate:"required"`
}

func (RawVolume) IsVolumeVariant() {}

type PersistentVolume struct {
	// type: `persistent`
	// `required` only validates if the value is not "zero", which for booleans is `false`
	// but `false` here is a valid value, so we have to make it a pointer to actually validate its presence
	Existing *bool `json:"existing" validate:"required"`

	// again splitting into variants, so we can validate each separately
	Variant PersistentVolumeVariant `json:"-"`
}

func (PersistentVolume) IsVolumeVariant() {}

type PersistentVolumeVariant interface{ IsPersistentVolumeVariant() }

type PersistentVolumeExisting struct {
	PvcName string `json:"pvcName" validate:"required"`
}

func (PersistentVolumeExisting) IsPersistentVolumeVariant() {}

type PersistentVolumeNew struct {
	// type: `persistent`
	AccessModes      []v1.PersistentVolumeAccessMode `json:"accessModes" validate:"required"`
	Size             string                          `json:"size,omitempty"  validate:"required"`
	StorageClassName string                          `json:"storageClassName,omitempty"  validate:"required"`
	VolumeMode       *v1.PersistentVolumeMode        `json:"volumeMode,omitempty"`
}

func (PersistentVolumeNew) IsPersistentVolumeVariant() {}

type SecretVolume struct {
	// type: `secret`
	Mode       *int32             `json:"mode"`
	SecretName string             `json:"secretName" validate:"required"`
	Items      map[string]*string `json:"items,omitempty"`
}

func (SecretVolume) IsVolumeVariant() {}

type ConfigMapVolume struct {
	// type: `configMap`
	Mode          *int32             `json:"mode"`
	ConfigMapName string             `json:"configMapName" validate:"required"`
	Items         map[string]*string `json:"items,omitempty"`
}

func (ConfigMapVolume) IsVolumeVariant() {}

// YAML package uses reflection to look for this interface on the given types and use these instead of the default behavior
func (v *Volume) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// `v` is not unmarshaled at this point yet, just created with zero values

	// type alias => identical in structure but doesn't copy methods => doesn't lead to recursion
	type alias Volume
	var a alias

	if err := unmarshal(&a); err != nil {
		return err
	}

	// initialize the fields from `a` which copies the parsed fields
	*v = Volume(a)

	switch v.Type {
	case VolumeTypeStandardTmpfs:
		fallthrough
	case VolumeTypeStandardLocal:
		var variant StandardVolume
		if err := unmarshal(&variant); err != nil {
			return err
		}
		v.Variant = variant
	case VolumeTypeRaw:
		var variant RawVolume
		if err := unmarshal(&variant); err != nil {
			return err
		}
		v.Variant = variant
	case VolumeTypePersistent:
		var persistentVariant PersistentVolume
		if err := unmarshal(&persistentVariant); err != nil {
			return err
		}

		if persistentVariant.Existing == nil {
			return fmt.Errorf("persistent volume missing 'existing' field")
		}
		if *persistentVariant.Existing {
			var variant PersistentVolumeExisting
			if err := unmarshal(&variant); err != nil {
				return err
			}
			persistentVariant.Variant = variant
		} else {
			var variant PersistentVolumeNew
			if err := unmarshal(&variant); err != nil {
				return err
			}
			if _, err := resource.ParseQuantity(variant.Size); err != nil {
				return fmt.Errorf("invalid volume size: %v", err)
			}
			persistentVariant.Variant = variant
		}
		v.Variant = persistentVariant
	case VolumeTypeSecret:
		var variant SecretVolume
		if err := unmarshal(&variant); err != nil {
			return err
		}
		v.Variant = variant
	case VolumeTypeConfigMap:
		var variant ConfigMapVolume
		if err := unmarshal(&variant); err != nil {
			return err
		}
		v.Variant = variant
	default:
		return fmt.Errorf("unknown volume type: %s", v.Type)
	}

	return nil
}
