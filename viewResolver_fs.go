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
	return &viewResolver_fs{
		fs:      fs,
		resolve: simpleViewResolver(".tsx", ".jsx"),
	}
}

func NewFsViewResolverCustom(fs fs.FS, r FSViewResolveFunc) ViewResolver {
	return &viewResolver_fs{
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

type viewResolver_fs struct {
	fs      fs.FS
	resolve FSViewResolveFunc
}

func (r *viewResolver_fs) ResolveViewFile(viewName string) (*url.URL, error) {
	return r.resolve(r.fs, viewName)
}

func (r *viewResolver_fs) ResolveModuleFile(fromModule ModuleMeta, importPath string) (*url.URL, error) {
	f, _ := filepath.Rel("/", filepath.Join(filepath.Join(fromModule.Dirname(), importPath)))
	f = filepath.ToSlash(f)
	if stat, err := fs.Stat(r.fs, f); err != nil {
		return nil, err
	} else {
		return url.ParseRequestURI("file:///" + f + "?ts=" + strconv.FormatInt(stat.ModTime().UnixMicro(), 16))
	}
}

func (r *viewResolver_fs) GetContent(url url.URL) (string, error) {
	f, _ := filepath.Rel("/", url.Path)
	f = filepath.ToSlash(f)
	if content, err := fs.ReadFile(r.fs, f); err != nil {
		return "", err
	} else {
		return string(content), nil
	}
}
