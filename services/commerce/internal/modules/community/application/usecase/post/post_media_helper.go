package post

import (
	"net/url"
	"path"
	"slices"
	"strings"

	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/services/commerce/internal/modules/community/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/mediatype"
)

// validateMediaItems checks every media item of a post: type, file extension,
// declared size, declared video duration and ordering. A non-positive limit in
// configuration disables the corresponding check.
func (uc *PostUseCase) validateMediaItems(media []dto.PostMediaReq) error {
	limits := uc.cfg.Community
	for _, m := range media {
		mt := mediatype.MediaType(m.MediaType)
		if !mt.IsValid() {
			return communityerrors.ErrInvalidMedia()
		}
		if err := validateMediaExtension(m.MediaURL, mt); err != nil {
			return err
		}
		if err := validateMediaSize(m.Bytes, mt, limits.MaxImageBytes, limits.MaxVideoBytes); err != nil {
			return err
		}
		if err := validateMediaDuration(mt, m.DurationSeconds, limits.MaxVideoDurationSeconds); err != nil {
			return err
		}
	}
	return validateMediaOrder(media)
}

// validateMediaExtension rejects a media URL whose file extension is not in the
// whitelist of its media type. A URL without an extension is accepted so that
// extension-less delivery URLs (e.g. f_auto) keep working.
func validateMediaExtension(mediaURL string, mt mediatype.MediaType) error {
	parsed, err := url.Parse(mediaURL)
	if err != nil {
		return communityerrors.ErrInvalidMedia()
	}
	ext := strings.TrimPrefix(strings.ToLower(path.Ext(parsed.Path)), ".")
	if ext == "" {
		return nil
	}
	allowed := mt.AllowedExtensions()
	if !slices.Contains(allowed, ext) {
		return communityerrors.ErrMediaExtensionNotAllowed(string(mt), "."+ext, allowed)
	}
	return nil
}

// validateMediaSize compares the client-declared byte size against the limit of
// the media type. A nil size skips the check; a non-positive limit disables it.
func validateMediaSize(bytes *int64, mt mediatype.MediaType, maxImageBytes, maxVideoBytes int64) error {
	if bytes == nil {
		return nil
	}
	limit := maxImageBytes
	if mt == mediatype.Video {
		limit = maxVideoBytes
	}
	if limit <= 0 || *bytes <= limit {
		return nil
	}
	return communityerrors.ErrMediaTooLarge(string(mt), limit)
}

// validateMediaDuration checks the client-declared video duration. Only applies
// to videos; a nil duration skips the check and a non-positive limit disables it.
// The value is a float because Cloudinary reports video duration as a decimal.
func validateMediaDuration(mt mediatype.MediaType, durationSeconds *float64, maxSeconds int) error {
	if mt != mediatype.Video || durationSeconds == nil {
		return nil
	}
	if maxSeconds <= 0 || *durationSeconds <= float64(maxSeconds) {
		return nil
	}
	return communityerrors.ErrVideoDurationTooLong(maxSeconds)
}

// validateMediaOrder requires every media item to carry a sort order and
// forbids duplicates. Any int16 value is accepted (no range restriction).
func validateMediaOrder(media []dto.PostMediaReq) error {
	seen := make(map[int16]struct{}, len(media))
	for _, m := range media {
		if m.SortOrder == nil {
			return communityerrors.ErrInvalidMediaOrder()
		}
		if _, dup := seen[*m.SortOrder]; dup {
			return communityerrors.ErrInvalidMediaOrder()
		}
		seen[*m.SortOrder] = struct{}{}
	}
	return nil
}
