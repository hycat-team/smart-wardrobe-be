package entities

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// UUIDList is a list of UUIDs stored in a Postgres uuid[] column.
//
// A bare []uuid.UUID cannot be written through pgx: database/sql has no
// default converter for a slice of uuid.UUID, so the parameter reaches the
// server as a record ("column ... is of type uuid[] but expression is of type
// record") or as an unparsable literal. Implementing driver.Valuer and
// sql.Scanner keeps the column type uuid[] and lets the value round-trip as
// the Postgres array literal {uuid1,uuid2}.
//
// A nil list is written as SQL NULL; an empty list is written as {} so the
// server-side CHECK constraint (cardinality > 0) still rejects it.
type UUIDList []uuid.UUID

// Value renders the list as a Postgres array literal accepted by a uuid[]
// column.
func (l UUIDList) Value() (driver.Value, error) {
	if l == nil {
		return nil, nil
	}
	var builder strings.Builder
	builder.WriteByte('{')
	for i, id := range l {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(id.String())
	}
	builder.WriteByte('}')
	return builder.String(), nil
}

// Scan parses the Postgres text representation of a uuid[] ({...}) or SQL
// NULL.
func (l *UUIDList) Scan(value any) error {
	if value == nil {
		*l = nil
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("failed to scan UUIDList: unsupported type %T", value)
	}

	raw := strings.TrimSpace(string(data))
	if len(raw) < 2 || raw[0] != '{' || raw[len(raw)-1] != '}' {
		return fmt.Errorf("failed to scan UUIDList: invalid array literal %q", raw)
	}

	body := raw[1 : len(raw)-1]
	if body == "" {
		*l = UUIDList{}
		return nil
	}

	parts := strings.Split(body, ",")
	result := make(UUIDList, len(parts))
	for i, part := range parts {
		id, err := uuid.Parse(strings.TrimSpace(part))
		if err != nil {
			return fmt.Errorf("failed to scan UUIDList: element %d: %w", i, err)
		}
		result[i] = id
	}
	*l = result
	return nil
}
