package entities

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Vector []float32

func (v Vector) Value() (driver.Value, error) {
	if len(v) == 0 {
		return nil, nil
	}
	var builder strings.Builder
	builder.WriteByte('[')
	for i, val := range v {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(strconv.FormatFloat(float64(val), 'f', -1, 32))
	}
	builder.WriteByte(']')
	return builder.String(), nil
}

func (v *Vector) Scan(value any) error {
	if value == nil {
		*v = nil
		return nil
	}

	strVal, ok := value.(string)
	if !ok {
		byteVal, ok := value.([]byte)
		if !ok {
			return errors.New("failed to scan Vector: unsupported data type")
		}
		strVal = string(byteVal)
	}

	strVal = strings.TrimSpace(strVal)
	if !strings.HasPrefix(strVal, "[") || !strings.HasSuffix(strVal, "]") {
		return fmt.Errorf("failed to scan Vector: invalid format: %s", strVal)
	}

	strVal = strVal[1 : len(strVal)-1]
	if len(strVal) == 0 {
		*v = Vector{}
		return nil
	}

	parts := strings.Split(strVal, ",")
	res := make(Vector, len(parts))
	for i, part := range parts {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(part), 32)
		if err != nil {
			return fmt.Errorf("failed to scan Vector: error parsing float: %w", err)
		}
		res[i] = float32(parsed)
	}

	*v = res
	return nil
}

func (Vector) GormDataType() string {
	return "vector"
}
