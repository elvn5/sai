// Package nodepath reproduces Node.js path helpers whose behavior the
// NestJS API's file names depended on.
package nodepath

import (
	"path"
	"strings"
)

// Extname mirrors Node's path.extname, which (unlike Go's path.Ext) treats
// a leading dot as part of the name: ".env" has no extension.
func Extname(name string) string {
	base := path.Base(strings.ReplaceAll(name, "\\", "/"))
	if ext := path.Ext(base); ext != base {
		return ext
	}
	return ""
}
