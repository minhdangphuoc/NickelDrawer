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

	// thumbCompression: level 9 on dithered pixels takes about 6 times as long as the default
	// for a few percent of a 300 x 400 file, and it was most of a thumbnail's time.
	thumbCompression = png.DefaultCompression
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
	return encodePNG(thumb, thumbCompression)
}

// fitInside is the largest size with the picture's aspect ratio inside maxWidth x maxHeight.
func fitInside(width, height, maxWidth, maxHeight int) Size {
	if width*maxHeight > height*maxWidth {
		return Size{Width: maxWidth, Height: max(1, height*maxWidth/width)}
	}
	return Size{Width: max(1, width*maxHeight/height), Height: maxHeight}
}

// writeThumb stores the thumbnail of a freshly converted picture, replacing an older one. When it
// cannot be made, the old thumbnail (of the previous fit) is removed so the gallery never shows a
// stale picture, and the failure is logged; the background thumbnail job tries again later.
func (library *Library) writeThumb(id int, convertedPNG []byte, log *Logger) bool {
	err := storeThumb(library.Paths, id, convertedPNG)
	if err == nil {
		return true
	}
	log.Printf("thumbnail %s not written: %v", imageFileName(id), err)
	library.removeThumb(id, log)
	return false
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

// MissingThumbnails lists the ids of the pictures without a thumbnail. It only stats files, so the
// state read can afford it.
func (library *Library) MissingThumbnails() []int {
	var missing []int
	for _, entry := range library.Entries {
		if _, err := os.Stat(library.Paths.Thumb(entry.ID)); err != nil {
			missing = append(missing, entry.ID)
		}
	}
	return missing
}
