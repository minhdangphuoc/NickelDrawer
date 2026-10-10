package main

import (
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	xdraw "golang.org/x/image/draw"

	"koboapps/internal/uikit"
)

const maxPictureTiles = 500

// pictureIDPattern mirrors the drawer's tile id rule (internal/drawer/screen_gallery.go).
var pictureIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

// pictureTile is one item of the array a gallery or viewer is bound to.
type pictureTile struct {
	id, label string
	thumbPath string
	imagePath string
	isCurrent bool
	infoLines string
}

// pictureTiles reads the bound array. Items without a valid id are dropped, as `appsctl drawer
// screen` does; relative file paths resolve against the folder of the --state file.
func (rc renderContext) pictureTiles(c uikit.Component) []pictureTile {
	reference, _ := c.Main().(string)
	values, _ := lookupVar(rc.vars, strings.TrimPrefix(reference, "$")).([]any)
	var tiles []pictureTile
	for _, raw := range values {
		fields, isObject := raw.(map[string]any)
		id, _ := fields["id"].(string)
		if !isObject || !pictureIDPattern.MatchString(id) || len(tiles) == maxPictureTiles {
			continue
		}
		tiles = append(tiles, pictureTile{
			id: id, label: stringField(fields, "label"), infoLines: stringField(fields, "info"),
			thumbPath: rc.resolvePath(stringField(fields, "thumb")), imagePath: rc.resolvePath(stringField(fields, "image")),
			isCurrent: fields["badge"] == "current",
		})
	}
	return tiles
}

func stringField(fields map[string]any, key string) string {
	value, _ := fields[key].(string)
	return value
}

func (rc renderContext) resolvePath(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(rc.baseDir, path)
}

// loadPicture decodes a PNG of at most maxPictureBytes; a missing, big or undecodable file
// returns false and the caller draws the missing-picture glyph.
func loadPicture(path string) (image.Image, bool) {
	if path == "" {
		return nil, false
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer file.Close()
	if info, statErr := file.Stat(); statErr != nil || info.Size() > maxPictureBytes {
		return nil, false
	}
	decoded, err := png.Decode(io.LimitReader(file, maxPictureBytes))
	if err != nil {
		return nil, false
	}
	return decoded, true
}

// pictureItem fits the picture at path into the box and centres it; false when it cannot load.
func pictureItem(path string, box image.Rectangle) (Item, bool) {
	source, loaded := loadPicture(path)
	if !loaded || source.Bounds().Empty() || box.Empty() {
		return Item{}, false
	}
	width, height := fitSize(source.Bounds().Dx(), source.Bounds().Dy(), box.Dx(), box.Dy())
	scaled := image.NewGray(image.Rect(0, 0, width, height))
	xdraw.ApproxBiLinear.Scale(scaled, scaled.Bounds(), source, source.Bounds(), xdraw.Src, nil)
	return Item{Shape: "image", X: box.Min.X + (box.Dx()-width)/2, Y: box.Min.Y + (box.Dy()-height)/2, W: width, H: height,
		Fill: noFill, Picture: scaled}, true
}

// fitSize scales width x height to the largest size inside boxWidth x boxHeight, same aspect.
func fitSize(width, height, boxWidth, boxHeight int) (int, int) {
	if width*boxHeight > height*boxWidth {
		return boxWidth, max(1, height*boxWidth/width)
	}
	return max(1, width*boxHeight/height), boxHeight
}
