package repositories

import "errors"

var ErrBrandSlugConflict = errors.New("brand slug already exists")
