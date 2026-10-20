package mapper

import (
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 029 D-07: the device fields must survive the map. A mapper that silently dropped
// them would leave the endpoint documented as carrying them while returning nothing,
// and no other test looks at this layer.
func TestMapToActivityEventResCarriesDeviceFields(t *testing.T) {
	name, ip, ua := "iPhone 15", "203.0.113.7", "Closy/1.0"
	row := repositories.AdminActivityEventRow{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		ActivityType: "login",
		OccurredAt:   time.Now(),
		CreatedAt:    time.Now(),
		Email:        "user@example.com",
		DeviceName:   &name,
		DeviceIP:     &ip,
		UserAgent:    &ua,
	}

	res := MapToActivityEventRes(row)
	require.NotNil(t, res.DeviceName)
	require.NotNil(t, res.DeviceIP)
	require.NotNil(t, res.UserAgent)
	assert.Equal(t, name, *res.DeviceName)
	assert.Equal(t, ip, *res.DeviceIP)
	assert.Equal(t, ua, *res.UserAgent)
}

// A row written before the migration has no device columns. They must stay nil so
// the JSON omits them: "not recorded" is not the empty string.
func TestMapToActivityEventResKeepsMissingDeviceFieldsNil(t *testing.T) {
	row := repositories.AdminActivityEventRow{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		ActivityType: "refresh",
		OccurredAt:   time.Now(),
		CreatedAt:    time.Now(),
		Email:        "user@example.com",
	}

	res := MapToActivityEventRes(row)
	assert.Nil(t, res.DeviceName)
	assert.Nil(t, res.DeviceIP)
	assert.Nil(t, res.UserAgent)
}
