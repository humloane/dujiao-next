// Package jsonmap provides the shared JSON object value used at persistence
// and transport boundaries. It is deliberately independent from GORM models
// and business modules so bounded contexts do not need to import a global
// models package merely to exchange arbitrary JSON objects.
package jsonmap

import (
	"database/sql/driver"
	"encoding/json"
)

// JSON is a JSON object with database/sql serialization support.
type JSON map[string]interface{}

// Value implements driver.Valuer.
func (value JSON) Value() (driver.Value, error) {
	if value == nil {
		return nil, nil
	}
	return json.Marshal(value)
}

// Scan implements sql.Scanner.
func (value *JSON) Scan(source interface{}) error {
	if source == nil {
		*value = make(JSON)
		return nil
	}
	switch typed := source.(type) {
	case []byte:
		return json.Unmarshal(typed, value)
	case string:
		// SQLite 手动写入或外部导入可能把 JSON 存成 TEXT；不处理会静默变成空对象。
		return json.Unmarshal([]byte(typed), value)
	default:
		return nil
	}
}
