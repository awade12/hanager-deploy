package archive

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func PackDir(root string, w io.Writer, ignoreFile string) error {
	ignores, _ := loadIgnore(root, ignoreFile)
	gz := gzip.NewWriter(w)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if skipPath(rel, ignores) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSocket != 0 {
			return nil
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = rel
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(tw, f)
		_ = f.Close()
		return err
	})
}

func PackDirToFile(root, dest, ignoreFile string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	return PackDir(root, f, ignoreFile)
}

func loadIgnore(root, name string) ([]string, error) {
	path := filepath.Join(root, name)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{".git", "node_modules", ".hangar-data"}, nil
		}
		return nil, err
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	return lines, sc.Err()
}

func skipPath(rel string, ignores []string) bool {
	for _, ig := range ignores {
		ig = strings.TrimSuffix(ig, "/")
		if rel == ig || strings.HasPrefix(rel, ig+"/") {
			return true
		}
	}
	base := filepath.Base(rel)
	for _, ig := range ignores {
		if strings.Contains(ig, "*") {
			continue
		}
		if base == ig {
			return true
		}
	}
	return false
}
