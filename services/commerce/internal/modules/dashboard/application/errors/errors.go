package dashboarderrors

import (
	"smart-wardrobe-be/shared/application/constants/apperror"
)

func ErrDashboardDataNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy dữ liệu thống kê dashboard.")
}

func ErrBrandAnalyticsNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy dữ liệu thống kê cho thương hiệu này.")
}
