package utils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/resource"
)

// ResourcePacks loads all resource packs in a path.
func ResourcePacks(path string) []*resource.Pack {
	var packs []*resource.Pack
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if strings.HasSuffix(info.Name(), ".zip") || strings.HasSuffix(info.Name(), ".mcpack") {
				pack, err := resource.Compile(path)
				if err != nil {
					return err
				}
				packs = append(packs, pack)
			}
			return nil
		})
	}
	return packs
}
