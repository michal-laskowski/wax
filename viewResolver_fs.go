package wax

import (
	"io/fs"
	"net/url"
	"path/filepath"
	"time"
)

type viewResolver_fs struct {
	fs fs.FS
}

func NewFsViewResolver(fs fs.FS) ViewResolver {
	return &viewResolver_fs{
		fs: fs,
	}
}

func (this *viewResolver_fs) ResolveViewFile(viewName string) (*url.URL, error) {
	f := viewName + ".tsx"
	if stat, err := fs.Stat(this.fs, f); err != nil {
		return nil, err
	} else {
		return url.ParseRequestURI("file:///" + f + "?ts=" + stat.ModTime().Format(time.RFC3339))
	}
}

func (this *viewResolver_fs) ResolveModuleFile(fromModule ModuleMeta, importPath string) (*url.URL, error) {
	f, _ := filepath.Rel("/", filepath.Join(filepath.Join(fromModule.Dirname(), importPath)))
	f = filepath.ToSlash(f)
	if stat, err := fs.Stat(this.fs, f); err != nil {
		return nil, err
	} else {
		return url.ParseRequestURI("file:///" + f + "?ts=" + stat.ModTime().Format(time.RFC3339))
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
