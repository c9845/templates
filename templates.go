/*
Package templates2 handles parsing and rendering HTML. This more-or-less wraps the golang
'html/template' package with some tooling for storing the parsed templates, showing
a requested template, and using source HTML stored in on-disk or embedded files.

Handling of HTML templates is done by parsing files in given directories and
caching them for future use within your application. Templates can be stored in
numerous subdirectories for ease of organization and allowing the same
filename or template declaration ({{define}}) to be used. Files stored at the
root templates directory are inherited into each subdirectory; this is useful
for storing files with shared {{declare}} blocks that are imported into other
templates stored in numerous subdirectories (ex.: header and footers).

Serving of a template is done by providing the subdirectory and name of the
template (aka filename). Note that due to this, you cannot serve templates from
the root directory. Again, the root directory is for storing templates shared
templates between multiple subdirectories.

An example of a directory structure for storing templates is below.
templates/
├─ header.html
├─ footer.html
├─ docs/
│  ├─ index.html
│  ├─ faq.html
│  ├─ how-to.html
├─ app/
│  ├─ index.html
│  ├─ users.html
│  ├─ widgits.html
*/
package templates

import (
	"embed"
	"errors"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//Config is the set of configuration settings for working with templates.
type Config struct {
	//Development is passed to each template when rendering the HTML to be sent
	//to the user so that the HTML can be altered based on if you are running
	//your app in a development mode/enviroment. Typically this is used to show
	//a banner on the page, loads extra diagnostic libraries or tools, and uses
	//non-cache busted static files.
	Development bool

	//UseLocalFiles is passed to each template when rendering the HTML to be sent
	//to the user so that the HTML can be altered to use locally hosted third
	//party libraries (JS, CSS) versus libraries retrieve from the internet.
	UseLocalFiles bool

	//BasePath is the full path to the directory where template files are stored
	//not including any subdirectories. There should be at least one template file
	//at this path. Files in this directory will be inherited into each subdirectory
	//when the templates are parsed; this is useful for commonelements such as
	//headers, footers, and scripts. For handling embedded files, this would be set
	//to a path based upon the location of your package main file. See
	//https://pkg.go.dev/embed#hdr-Directives for more information.
	BasePath string

	//SubDirs is a list of subdirectories of the BasePath where you store template
	//files. This may be empty if you have no subdirectories. This must only be the
	//actual directory names, not full paths. Full paths will be constructed from
	//BasePath.
	SubDirs []string

	//Extension is the extension you use for your HTML files. This defaults to "html".
	Extension string

	//UseEmbedded means files built into the golang executable will be used rather
	//than files stored on-disk. You must have read the embedded files, with code
	//such as var embeddedFiles embed.FS, prior and you must provide the embed.FS to
	//the field EmbeddedFS.
	UseEmbedded bool

	//EmbeddedFiles is the filesystem embedded into this executable via the embed package.
	//You must have read the embedded files, with code such as var embeddedFiles embed.FS,
	//prior and you must set UseEmbedded to true to enable use of these files.
	EmbeddedFS embed.FS

	//FuncMap is a collection of functions that you want to use in your templates to
	//augment the golang provided templating funcs. This package provides some default
	//extra funcs in templates-templatefuncs.go. See https://pkg.go.dev/text/template for
	//more info.
	//To provide extra funcs to templates, use code such as the following:
	/*
		config, err := templates..DefaultOnDiskConfig("/path/to/templates", []string{"subdir1", "subdir2"})
		if err != nil {
			//handle err
		}
		config.FuncMap = template.FuncMap{
			"indexOf":   templates.FuncIndexOf,
			"myNewFunc": myNewFunc,
		}
		err = config.Build()
		if err != nil {
			//handle err
		}
	*/
	FuncMap template.FuncMap

	//CacheBustingFilePairs is a key-value list of filesnames that match up an original
	//file name to the file's cache busting file name. This list is then passed to your
	//templates when rendered to replace the known original filename (i.e.: script.min.js)
	//with the cache busting filename that is most likely programmatically created (i.e.:
	//A1B2C3D4.script.min.js). See the package github.com/c9845/cachebusting for an example
	//implementation and tooling.
	//
	//To use the cache busting file, you would use template code similar to the following
	//to handle the filename replacement:
	/*
		<head>
			{{$originalFile := "styles.min.css"}}
			{{$cacheBustFiles := .CacheBustFiles}}

			{{/*If the key "styles.min.css" exists in $cacheBustFiles, then the associated cache-busted filename will be returned as {{.}}. *\/}}
			{{with index $cacheBustFiles $originalFile}}
			{{$cacheBustedFile := .}}
			<link rel="stylesheet" href="/static/css/{{$cacheBustedFile}}">
			{{else}}
			<link rel="stylesheet" href="/static/css/{{$originalFile}}">
			{{end}}
		</head>
	*/
	CacheBustingFilePairs map[string]string

	//templates holds the list of parsed files constructed into golang templates.
	//Templates are organized by subdirectory since that is how they are organized on
	//disk and this allows for filenames, or {{define}} blocks, to only need to be
	//unique within a subdirectory. This is where a specific template is looked up when
	//Show() is called to actually show and return the HTML to a user and their browser.
	templates map[string]*template.Template
}

//defaults
const (
	defaultExtension = "html"
)

//errors
var (
	//ErrBasePathNotSet is returned if a user calls Save() and not path to the
	//templates was provided.
	ErrBasePathNotSet = errors.New("templates: no value set for TemplatesBasePath")

	//ErrNoSubDirsProvided is returned when no subdirectories were provided. As of
	//now we require at least one subdirectory.
	ErrNoSubDirsProvided = errors.New("templates: no template subdirectories were provided, at least one must be")

	//ErrInvalidSubDir is returned if a user calls Save() and the provided
	//subdirectory cannot be found.
	ErrInvalidSubDir = errors.New("templates: empty or all whitespace string provided for TemplatesSubDirs is not allowed")

	//ErrNoEmbeddedFilesProvided is returned when a user is using a config with embedded files
	//but no embedded files were provided.
	ErrNoEmbeddedFilesProvided = errors.New("templates: no embedded files provided")
)

//config is the package level saved config. This stores your config when you want to store
//it for global use. It is populated when you use one of the Default...Config() funcs.
var config Config

//NewConfig returns a config for managing your templates with some defaults set.
func NewConfig() *Config {
	return &Config{
		Extension: defaultExtension,
	}
}

//DefaultConfig initializes the package level config with some defaults set. This wraps
//NewConfig() and saves the config to the package.
func DefaultConfig() {
	cfg := NewConfig()
	config = *cfg
}

//NewOnDiskConfig returns a config for managing your templates when the source files are
//stored on disk.
func NewOnDiskConfig(basePath string, subdirs []string) *Config {
	return &Config{
		BasePath:  basePath,
		SubDirs:   subdirs,
		Extension: defaultExtension,
		templates: make(map[string]*template.Template),
	}
}

//DefaultOnDiskConfig initializes the package level config with the path and directories
//provided and some defaults.
func DefaultOnDiskConfig(basePath string, subdirs []string) {
	cfg := NewOnDiskConfig(basePath, subdirs)
	cfg.FuncMap = DefaultFuncMap()
	config = *cfg
}

//NewEmbeddedConfig returns a config for managing your templates when the source files are
//stored embedded in the app executable.
func NewEmbeddedConfig(embeddedFS embed.FS, basePath string, subdirs []string) *Config {
	//build base config
	return &Config{
		BasePath:    basePath,
		SubDirs:     subdirs,
		Extension:   defaultExtension,
		UseEmbedded: true,
		EmbeddedFS:  embeddedFS,
		templates:   make(map[string]*template.Template),
	}
}

//DefaultEmbeddedConfig initializes the package level config with the path and directories
//provided and some defaults.
func DefaultEmbeddedConfig(embeddedFS embed.FS, basePath string, subdirs []string) {
	cfg := NewEmbeddedConfig(embeddedFS, basePath, subdirs)
	cfg.FuncMap = DefaultFuncMap()
	config = *cfg
}

//validate handles validation of a provided config.
func (c *Config) validate() (err error) {
	//Check if BasePath is set.
	c.BasePath = strings.TrimSpace(c.BasePath)
	if c.BasePath == "" {
		return ErrBasePathNotSet
	}

	//Check that BasePath exists. This only needs to be done for on-disk configurations
	//since we assume that if you are using embedded files you know your directory
	//structure and what subdirectories exist.
	if !c.UseEmbedded {
		if _, err := os.Stat(c.BasePath); os.IsNotExist(err) {
			return err
		}
	}

	//Check if SubDirs was provided and if so, make sure that each directory provided
	//exists. SubDirs could be blank if you have no subdirectories for organizing your
	//template files. This only needs to be done for on-disk configurations since we
	//assume that if you are using embedded files you know your directory structure and
	//what subdirectories exist.
	if !c.UseEmbedded {
		for idx, p := range c.SubDirs {
			p = strings.TrimSpace(p)
			if p == "" {
				return ErrInvalidSubDir
			}

			p = filepath.FromSlash(p)

			if _, err := os.Stat(filepath.Join(c.BasePath, p)); os.IsNotExist(err) {
				return err
			}

			c.SubDirs[idx] = p
		}
	}

	//Make sure a filename extension was provided, if not use the default.
	c.Extension = strings.TrimSpace(c.Extension)
	if c.Extension == "" {
		c.Extension = defaultExtension
	}

	//If user is using embedded files, make sure something was provided.
	if c.UseEmbedded && c.EmbeddedFS == (embed.FS{}) {
		return ErrNoEmbeddedFilesProvided
	}

	return
}

//Build handles finding the templates files, parsing them, and building the golang templates.
//This func works by looking for files with the correct extension in the provided BasePath
//and in subdirectories built from the BasePath and each SubDirs provided. Templates in
//subdirectories inherit templates from the base directory (for usage of common templates
//such as headers, footers). Files in each subdirectory are handled independently and cannot
//reference a template from another subdirectory; this allows for templates that use the same
//name ({{define}}) or same filename to exist and be used.
func (c *Config) Build() (err error) {
	//validate the config
	err = c.validate()
	if err != nil {
		return
	}

	//empty out field that holds built templates in case Build() is called more than once.
	c.templates = make(map[string]*template.Template)

	//Build complete paths to each file in the root directory. This list of paths will be
	//appended to the list of files from each subdirectory (for inheritance). These files
	//can also be served independently from a subdirectory using "" as the subdir to Show().
	baseFilePaths, err := c.buildPathsToFiles(c.BasePath)
	if err != nil {
		return
	}

	//Parse the templates in the base directory since the user may have not provided any
	//subdirectories. These templates are parsed with a blank subdirectory name so that
	//when templates are shown a user can provide Show(w, "", "template name", nil).
	//Note the template.New("") with the blank template name. This is needed so that we
	//can add the FuncMap to the template files we are about to parse.
	if len(baseFilePaths) > 0 {
		t, innerErr := template.New("").Funcs(c.FuncMap).ParseFiles(baseFilePaths...)
		if innerErr != nil {
			log.Println("templates.Build", "error parsing files at base path", innerErr)
			return innerErr
		}
		c.templates[""] = t
	}

	//Build complete paths to each file in each subdirectory and parse the templates in
	//each after appending the filepaths from the base directory. This is similar to how
	//the base files were handled above except that we inheret the base files into each
	//subdirectory and we parse each subdirectory independently from each other.
	for _, subDir := range c.SubDirs {
		//When subdirectory(ies) are provided, each is only a subdirectory name(s), not a
		//complete path(s). We have the build the complete path to each subdirectory first.
		//Note that we have to handle paths specially for embedded files since the path
		//separator is always "/" even on Windows.
		completePathToSubdDir := filepath.Join(c.BasePath, subDir)
		if c.UseEmbedded {
			completePathToSubdDir = filepath.ToSlash(completePathToSubdDir)
		}

		//Build complete paths to each file in the subdirectory.
		subdirFilepaths, innerErr := c.buildPathsToFiles(completePathToSubdDir)
		if innerErr != nil {
			return innerErr
		}

		//Skip this subdirectory if no template files are in it.
		if len(subdirFilepaths) == 0 {
			continue
		}

		//Add the base file paths to the subdirectory's file for inheritance.
		subdirFilepaths = append(subdirFilepaths, baseFilePaths...)

		//Parse the templates in the subdirectory. These templates are parsed with the
		//subdirecotry name so that when templates are shown a user can provide
		//Show(w, "subdir", "template name", nil).
		//Note the template.New("") with the blank template name. This is needed so that we
		//can add the FuncMap to the template files we are about to parse.
		t, innerErr := template.New("").Funcs(c.FuncMap).ParseFiles(subdirFilepaths...)
		if innerErr != nil {
			log.Println("templates.Build", "error parsing files at subdir '"+subDir+"'", innerErr)
			return innerErr
		}
		c.templates[subDir] = t
	}

	return
}

//Build builds the templates using the default package level config.
func Build() (err error) {
	err = config.Build()
	return
}

//buildPathsToFiles constructs the full path to each template file since we need the full, complete
//path to each for parsing in ParseFiles().
//pathToDirectory may seem like a duplicate and we could just use c.TemplatesBasePath, however,
//then we could not reuse this func for handling subdirectory files.
func (c *Config) buildPathsToFiles(pathToDirectory string) (paths []string, err error) {
	//Determine the correct ReadDir func. This is used to handle reading files stored
	//on disk or files that are embedded in the app's executable.
	var readFunc func(string) ([]fs.DirEntry, error)
	if c.UseEmbedded {
		readFunc = c.EmbeddedFS.ReadDir
	} else {
		readFunc = os.ReadDir
	}

	//Build complete paths to each file in the directory.
	//Make sure that path to embedded files always uses forward slash separators per embed package docs.
	if c.UseEmbedded {
		pathToDirectory = filepath.ToSlash(pathToDirectory)
	}
	files, err := readFunc(pathToDirectory)
	if err != nil {
		return
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		//Ignore files that don't end in the required extension. Not just checking for
		//existance of the extension (using strings.Contains) since the same set of
		//characters may exist elsewhere in the file's name.
		if filepath.Ext(f.Name()) != "."+c.Extension {
			continue
		}

		//Add complete path to template to list of paths. Have to handle path to embedded
		//files specially since they always use a "/" separator, even on Windows.
		completePathToFile := filepath.Join(pathToDirectory, f.Name())
		if c.UseEmbedded {
			completePathToFile = filepath.ToSlash(completePathToFile)
		}

		paths = append(paths, completePathToFile)
	}

	return
}

//Show renders a template as HTML. This returns the page to the user's browser. This works
//by taking a subdirectory's name subdir and the name of a template (a filename) templateName
//and looks up the associated template that was parsed earlier returning it with any
//injected data and cache busting files.
//Note that the user provided injectedData will be available at {{.Data}} in HTML templates.
func (c *Config) Show(w http.ResponseWriter, subdir, templateName string, injectedData interface{}) {
	//Get data to render html template.
	//We provide some of the config defined data as well as user-provided data via
	//the injectedData field. The injectedData field can hold any data.
	//We aren't just reusing the Config{} struct here since we want better control
	//over what data is used in the rendering process. Plus, not all the information
	//stored in a Config{} object is needed here.
	data := struct {
		Development    bool
		UseLocalFiles  bool
		CacheBustFiles map[string]string
		InjectedData   interface{}
	}{
		Development:    c.Development,
		UseLocalFiles:  c.UseLocalFiles,
		CacheBustFiles: c.CacheBustingFilePairs,
		InjectedData:   injectedData,
	}

	//Add the extension to the template (file) name if needed. This handles instances
	//where Show() was called without the extension (which is semi-expected since it
	//shortens up the Show() call and removes the need to provide the extension each
	//time). We need the extension since that was the name of the file when it was
	//parsed to cache the templates.
	ext := filepath.Ext(templateName)
	if ext == "" {
		templateName += "." + c.Extension
	}

	//Serve the correct template based on the subdirectory. Remember, you could have
	//the same template name in multiple subdirectories! While we could return the error
	//here (return errror.New...), we don't because we assume that anyone developing
	//using this package is acutely aware of their subdirectory name(s) and will test
	//this prior.
	t, ok := c.templates[subdir]
	if !ok {
		err := errors.New("templates.Show: invalid subdirectory '" + subdir + "'")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := t.ExecuteTemplate(w, templateName, data); err != nil {
		//handle displaying of the templates if some kind of error occurs.
		http.Error(w, err.Error(), http.StatusNotFound)

		//log errors out since they may not always show up in gui
		log.Println("templates.Show: error during execute", err)

		return
	}
}

//Show handles showing a template using the default package-level config.
func Show(w http.ResponseWriter, subdir, templateName string, injectedData interface{}) {
	config.Show(w, subdir, templateName, injectedData)
}

//GetConfig returns the current state of the package level config.
func GetConfig() (c *Config) {
	return &config
}

//Development sets the Development field on the package level config.
func Development(yes bool) {
	config.Development = yes
}

//UseLocalFiles sets the UseLocalFiles field on the package level config.
func UseLocalFiles(yes bool) {
	config.UseLocalFiles = yes
}

//CacheBustingFilePairs sets the CacheBustingFilePairs field on the package level config.
func CacheBustingFilePairs(pairs map[string]string) {
	config.CacheBustingFilePairs = pairs
}

//DefaultFuncMap returns the list of extra funcs defined for use in templates.
func DefaultFuncMap() template.FuncMap {
	return template.FuncMap{
		"indexOf":      FuncIndexOf,
		"dateReformat": FuncDateReformat,
		"addInt":       FuncAddInt,
	}
}

//PrintEmbeddedFileList prints out the list of files embedded into the executable. This should
//be used for diagnostics purposes only to confirm which files are embedded with the //go:embed
//directives elsewhere in your app.
func PrintEmbeddedFileList(e embed.FS) {
	//the directory "." means the root directory of the embedded file.
	const startingDirectory = "."

	err := fs.WalkDir(e, startingDirectory, func(path string, d fs.DirEntry, err error) error {
		log.Println(path)
		return nil
	})
	if err != nil {
		log.Fatalln("templates.PrintEmbeddedFiles", "error walking embedded directory", err)
		return
	}

	//exit after printing since you should never need to use this function outside of testing
	//or development.
	log.Println("templates.PrintEmbeddedFiles", "os.Exit() called, remove or skip PrintEmbeddedFileList to continue execution.")
	os.Exit(0)
}
