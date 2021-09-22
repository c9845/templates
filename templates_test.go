package templates

import (
	"embed"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//go:embed _testdata
var embeddedFiles embed.FS

func TestNewConfig(t *testing.T) {
	c := NewConfig()
	if c == nil {
		t.Fatal("nothing returned")
		return
	}
}

func TestNewOnDiskConfig(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
		return
	}

	base := filepath.Join(dir, "_testdata", "templates")
	subdirs := []string{"app", "help"}
	c := NewOnDiskConfig(base, subdirs)
	if c.BasePath != base {
		t.Fatal("base path not set correctly")
		return
	}
	if len(c.SubDirs) != len(subdirs) {
		t.Fatal("sub dirs not set correctly")
		return
	}
	if c.UseEmbedded {
		t.Fatal("UseEmbedded should have been set to false")
		return
	}
}
func TestNewEmbeddedConfig(t *testing.T) {
	base := filepath.Join("_testdata", "templates")
	subdirs := []string{"app", "help"}
	c := NewEmbeddedConfig(embeddedFiles, base, subdirs)
	if c.BasePath != base {
		t.Fatal("base path not set correctly")
		return
	}
	if len(c.SubDirs) != len(subdirs) {
		t.Fatal("sub dirs not set correctly")
		return
	}
	if !c.UseEmbedded {
		t.Fatal("UseEmbedded should have been set to true")
		return
	}
}

func TestValidate(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
		return
	}

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Don't set a base path.
	base := ""
	subdirs := []string{"app", "help"}
	c := NewOnDiskConfig(base, subdirs)
	err = c.validate()
	if err != ErrBasePathNotSet {
		t.Fatal("ErrBasePathNotSet should have occured but didn't")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Set a base path that doesn't exist.
	base = filepath.Join(dir, "_testdata", "non-existant-templates")
	subdirs = []string{"app", "help"}
	c = NewOnDiskConfig(base, subdirs)
	err = c.validate()
	if err == nil {
		t.Fatal("Error about invalid base path should have occured but didn't")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Provide a blank subdir name.
	base = filepath.Join(dir, "_testdata", "templates")
	subdirs = []string{"", "help"}
	c = NewOnDiskConfig(base, subdirs)
	err = c.validate()
	if err != ErrInvalidSubDir {
		t.Fatal("ErrInvalidSubDir should have occured but didn't")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Set a subdir that doesn't exist.
	base = filepath.Join(dir, "_testdata", "templates")
	subdirs = []string{"non-existant-app", "help"}
	c = NewOnDiskConfig(base, subdirs)
	err = c.validate()
	if err == nil {
		t.Fatal("Error about invalid subdirectory should have occured but didn't")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Test if empty embedded file list was provided.
	base = filepath.Join("_testdata", "templates")
	subdirs = []string{"app", "help"}
	c = NewEmbeddedConfig(embed.FS{}, base, subdirs)
	err = c.validate()
	if err != ErrNoEmbeddedFilesProvided {
		t.Fatal("ErrNoEmbeddedFilesProvided should have occured but didn't")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Make sure default extension was set if left blank.
	base = filepath.Join(dir, "_testdata", "templates")
	subdirs = []string{"app", "help"}
	c = NewOnDiskConfig(base, subdirs)
	err = c.validate()
	if err != nil {
		t.Fatal("Error occured but should not have")
		return
	}
	if c.Extension != defaultExtension {
		t.Fatal("Default extension not set correctly")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Check if a blank extension is set.
	c.Extension = " "
	err = c.validate()
	if err != nil {
		t.Fatal("Error occured but should not have")
		return
	}
	if c.Extension != defaultExtension {
		t.Fatal("Blank extension was not replaced by default as expected")
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Check if an extension is set.
	c.Extension = ".tmpl"
	err = c.validate()
	if err != nil {
		t.Fatal("Error occured but should not have")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
}

func TestBuildPathsToFiles(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
		return
	}

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Build paths for on-disk files.
	base := filepath.Join(dir, "_testdata", "templates")
	subdirs := []string{"app", "help"}
	c := NewOnDiskConfig(base, subdirs)
	err = c.validate()
	if err != nil {
		t.Fatal("Error should not have occured but did")
		return
	}

	paths, err := c.buildPathsToFiles(filepath.Join(base, "app"))
	if err != nil {
		t.Fatal("Error should not have occured but did")
		return
	}
	if len(paths) == 0 {
		t.Fatal("No paths were returned but should have been")
		return
	}
	for _, p := range paths {
		if !strings.Contains(p, c.Extension) {
			t.Fatal("Path does not use extension as expected")
			return
		}
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Build paths for embedded files.
	base = filepath.Join("_testdata", "templates")
	subdirs = []string{"app", "help"}
	c = NewEmbeddedConfig(embeddedFiles, base, subdirs)
	err = c.validate()
	if err != nil {
		t.Fatal("Error should not have occured but did")
		return
	}

	paths, err = c.buildPathsToFiles(filepath.Join(base, "app"))
	if err != nil {
		t.Fatal("Error should not have occured but did", err)
		return
	}
	if len(paths) == 0 {
		t.Fatal("No paths were returned but should have been")
		return
	}
	for _, p := range paths {
		if !strings.Contains(p, c.Extension) {
			t.Fatal("Path does not use extension as expected")
			return
		}
		if !strings.Contains(p, filepath.ToSlash(base)) {
			t.Fatal("Path does not use base path as expected")
			return
		}
		if strings.Contains(p, "\\") {
			t.Fatal("Paths to files build for embedded files should only use forward slashes")
			return
		}
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
}

func TestBuild(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
		return
	}

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Create config that will fail validation.
	base := filepath.Join(dir, "_testdata", "templates-invalid")
	subdirs := []string{" ", "help"}
	c := NewOnDiskConfig(base, subdirs)
	err = c.Build()
	if err == nil {
		t.Fatal("Error should have occured but didn't")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Create on disk config that successfully build.
	base = filepath.Join(dir, "_testdata", "templates")
	subdirs = []string{"app", "help"}
	c = NewOnDiskConfig(base, subdirs)
	err = c.Build()
	if err != nil {
		t.Fatal("Error should have occured but didn't")
		return
	}
	if c.templates == nil {
		t.Fatal("Templates not built as expected")
		return
	}

	if len(c.templates) != len(subdirs)+1 {
		//number of key-values created in map of templates
		//1 for each subdirectory plus 1 for base batch
		t.Fatal("Incorrect number of template.Templates created in map")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Create embedded config that successfully build.
	base = filepath.Join("_testdata", "templates")
	subdirs = []string{"app", "help"}
	c = NewEmbeddedConfig(embeddedFiles, base, subdirs)
	err = c.Build()
	if err != nil {
		t.Fatal("Error should have occured but didn't", err)
		return
	}
	if c.templates == nil {
		t.Fatal("Templates not built as expected")
		return
	}

	if len(c.templates) != len(subdirs)+1 {
		//number of key-values created in map of templates
		//1 for each subdirectory plus 1 for base batch
		t.Fatal("Incorrect number of template.Templates created in map")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
}

func TestDefaultConfig(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
		return
	}

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//GetConfig()
	base := filepath.Join(dir, "_testdata", "templates")
	subdirs := []string{"app", "help"}
	DefaultOnDiskConfig(base, subdirs)
	c := GetConfig()
	if c.BasePath != base {
		t.Fatal("Default config not saved correctly")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Development
	Development(true)
	c = GetConfig()
	if !c.Development {
		t.Fatal("Development field not set correctly")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//UseLocalFiles
	UseLocalFiles(true)
	c = GetConfig()
	if !c.UseLocalFiles {
		t.Fatal("UseLocalFiles field not set correctly")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//CacheBustingFilePairs
	pairs := map[string]string{
		"original.css": "modified.css",
	}
	CacheBustingFilePairs(pairs)
	c = GetConfig()
	if len(c.CacheBustingFilePairs) != len(pairs) {
		t.Fatal("CacheBustingFilePairs field not set correctly")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
}

func TestDefaultFuncMap(t *testing.T) {
	tfm := DefaultFuncMap()
	if tfm == nil {
		t.Fatal("Func map not returned as expected")
		return
	}
}

func TestShow(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
		return
	}

	base := filepath.Join(dir, "_testdata", "templates")
	subdirs := []string{"app", "help"}
	DefaultOnDiskConfig(base, subdirs)
	c := GetConfig()
	err = c.Build()
	if err != nil {
		t.Fatal("failed building for some reason...", err)
		return
	}

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Good file to serve.
	w := httptest.NewRecorder()
	c.Show(w, "app", "app", nil)
	if w.Code != http.StatusOK {
		t.Fatal("Error showing", w.Code, w.Body)
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

	//Test Start>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//Bad subdir to serve.
	w = httptest.NewRecorder()
	c.Show(w, "app-subdir-non-existant", "app", nil)
	if w.Code == http.StatusOK {
		t.Fatal("Error did not occur as expected")
		return
	}
	//Test End<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
}
