package wax

import (
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

func NewFsViewResolver(fs fs.FS, options ...FsViewResolverOptions) ViewResolver {
	return &viewResolver_fs{
		fs: fs,
		ext: []string{
			".tsx",
			".jsx",
		},
	}
}

type viewResolver_fs struct {
	fs  fs.FS
	ext []string
}
type FsViewResolverOptions func(*viewResolver_fs)

func SearchExtensions(ext ...string) FsViewResolverOptions {
	return func(e *viewResolver_fs) {
		ext = []string{}
		for _, e := range ext {
			if e[0] != '.' {
				panic("extension must start with dot")
			}
			ext = append(ext, e)
		}
	}
}

func (this *viewResolver_fs) ResolveViewFile(viewName string) (*url.URL, error) {
	for _, e := range this.ext {
		f := viewName + e
		if stat, err := fs.Stat(this.fs, f); err != nil {
			//continue
		} else {
			return url.ParseRequestURI("file:///" + f + "?ts=" + strconv.FormatInt(stat.ModTime().UnixMicro(), 16))
		}
	}
	return nil, &os.PathError{
		Op:   "not_found",
		Path: viewName,
		Err:  errors.New("could not resolve view file"),
	}
}

func (this *viewResolver_fs) ResolveModuleFile(fromModule ModuleMeta, importPath string) (*url.URL, error) {
	f, _ := filepath.Rel("/", filepath.Join(filepath.Join(fromModule.Dirname(), importPath)))
	f = filepath.ToSlash(f)
	if stat, err := fs.Stat(this.fs, f); err != nil {
		return nil, err
	} else {
		return url.ParseRequestURI("file:///" + f + "?ts=" + strconv.FormatInt(stat.ModTime().UnixMicro(), 16))
	}
}

func (this *viewResolver_fs) GetContent(url url.URL) (string, error) {
	f, _ := filepath.Rel("/", url.Path)
	f = filepath.ToSlash(f)
	if content, err := fs.ReadFile(this.fs, f); err != nil {
		return "", err
	} else {
		return string(content), nil
	}
}
