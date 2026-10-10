package wallpaper

import (
	"bytes"
	"errors"
	"image/png"
	"os"

	"koboapps/internal/i18n"
)

const (
	thumbMaxWidth  = 300
	thumbMaxHeight = 400

	// maxLazyThumbnails keeps `state` under its time budget on the Kobo's CPU.
	maxLazyThumbnails = 2
)

// MakeThumbnail shrinks a converted picture to fit inside 300 x 400 and quantizes it to the same
// 16 grey levels as the wallpaper itself.
func MakeThumbnail(convertedPNG []byte) ([]byte, error) {
	decoded, err := png.Decode(bytes.NewReader(convertedPNG))
	if err != nil {
		return nil, i18n.NewError("msg.err_decode").WithCause(err)
	}
	gray := toGray(decoded)
	size := fitInside(gray.Rect.Dx(), gray.Rect.Dy(), thumbMaxWidth, thumbMaxHeight)
	thumb := fitToScreen(gray, size, ModeFit)
	ditherFloydSteinberg(thumb)
	return encodePNG(thumb)
}

// fitInside is the largest size with the picture's aspect ratio inside maxWidth x maxHeight.
func fitInside(width, height, maxWidth, maxHeight int) Size {
	if width*maxHeight > height*maxWidth {
		return Size{Width: maxWidth, Height: max(1, height*maxWidth/width)}
	}
	return Size{Width: max(1, width*maxHeight/height), Height: maxHeight}
}

// writeThumb stores the thumbnail of a freshly converted picture. A failure is only logged: the
// gallery regenerates missing thumbnails lazily.
func (library *Library) writeThumb(id int, convertedPNG []byte, log *Logger) {
	if err := storeThumb(library.Paths, id, convertedPNG); err != nil {
		log.Printf("thumbnail %s not written: %v", imageFileName(id), err)
	}
}

func storeThumb(paths Paths, id int, convertedPNG []byte) error {
	thumb, err := MakeThumbnail(convertedPNG)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(paths.Thumbs(), 0o755); err != nil {
		return i18n.NewError("msg.err_mkdir", "path", paths.Thumbs()).WithCause(err)
	}
	return writeFileAtomic(paths.Thumb(id), ".thumb-", thumb)
}

func (library *Library) removeThumb(id int, log *Logger) {
	if err := os.Remove(library.Paths.Thumb(id)); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("cannot remove thumbnail %s: %v", imageFileName(id), err)
	}
}

// ensureThumbnails creates up to limit missing thumbnails from the stored pictures.
func (library *Library) ensureThumbnails(limit int, log *Logger) {
	made := 0
	for _, entry := range library.Entries {
		if made >= limit {
			return
		}
		if _, err := os.Stat(library.Paths.Thumb(entry.ID)); err == nil {
			continue
		}
		content, err := os.ReadFile(library.Paths.Image(entry.ID))
		if err != nil {
			log.Printf("thumbnail %s: cannot read the picture: %v", imageFileName(entry.ID), err)
			continue
		}
		library.writeThumb(entry.ID, content, log)
		made++
	}
}
