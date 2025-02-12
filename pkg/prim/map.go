package prim

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/spf13/cast"
)

type Map map[string]interface{}

// Scan sqlx JSON scan method
func (m *Map) Scan(val interface{}) error {
	switch v := val.(type) {
	case []byte:
		return json.Unmarshal(v, &m)
	case string:
		return json.Unmarshal([]byte(v), &m)
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
}

// Value sqlx JSON value method
func (m Map) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m Map) ToStruct(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &m)
}

func (m Map) ToJSON() string {
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}

func (m Map) Map(key string) Map {
	return cast.ToStringMap(m[key])
}

func (m Map) String(key string) string {
	return cast.ToString(m[key])
}

func (m Map) Bool(key string) bool {
	return cast.ToBool(m[key])
}

func (m Map) Int(key string) int {
	return cast.ToInt(m[key])
}

func (m Map) Int64(key string) int64 {
	return cast.ToInt64(m[key])
}

func (m Map) Float(key string) float32 {
	return cast.ToFloat32(m[key])
}

func (m Map) Float64(key string) float64 {
	return cast.ToFloat64(m[key])
}

func (m Map) StringSlice(key string) []string {
	return cast.ToStringSlice(m[key])
}

func (m Map) MapSlice(key string) []Map {
	b, err := json.Marshal(m[key])
	if err != nil {
		return nil
	}

	var v []Map
	if err = json.Unmarshal(b, &v); err != nil {
		return nil
	}

	return v
}
