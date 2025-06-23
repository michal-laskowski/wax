package wax

import (
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

func NewFsViewResolver(fs fs.FS) ViewResolver {
	return &viewResolverFS{
		fs:      fs,
		resolve: simpleViewResolver(".tsx", ".jsx"),
	}
}

func NewFsViewResolverCustom(fs fs.FS, r FSViewResolveFunc) ViewResolver {
	return &viewResolverFS{
		fs:      fs,
		resolve: r,
	}
}

type FSViewResolveFunc = func(fs fs.FS, viewName string) (*url.URL, error)

func simpleViewResolver(ext ...string) FSViewResolveFunc {
	for _, e := range ext {
		if e[0] != '.' {
			panic("extension must start with dot")
		}
	}

	return func(onFS fs.FS, viewName string) (*url.URL, error) {
		for _, e := range ext {
			f := viewName + e
			if stat, err := fs.Stat(onFS, f); err != nil {
				// continue
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
}

type viewResolverFS struct {
	fs      fs.FS
	resolve FSViewResolveFunc
}

func (r *viewResolverFS) ResolveViewFile(viewName string) (*url.URL, error) {
	return r.resolve(r.fs, viewName)
}

func (r *viewResolverFS) ResolveModuleFile(fromModule ModuleMeta, importPath string) (*url.URL, error) {
	fromDir := filepath.Dir(fromModule.URL.Path)
	f, _ := filepath.Rel("/", filepath.Join(filepath.Join(fromDir, importPath)))
	f = filepath.ToSlash(f)
	stat, err := fs.Stat(r.fs, f)
	if err != nil {
		return nil, err
	}

	return url.ParseRequestURI("file:///" + f + "?ts=" + strconv.FormatInt(stat.ModTime().UnixMicro(), 16))
}

func (r *viewResolverFS) GetContent(url url.URL) (string, error) {
	f, _ := filepath.Rel("/", url.Path)
	f = filepath.ToSlash(f)
	content, err := fs.ReadFile(r.fs, f)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
