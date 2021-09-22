## Introduction:
This package wraps around the golang `html/templates` package to provide some additional tooling and aiding in ease of use around working with HTML templates for building web pages.

## Details:
- Works with on-disk or embedded source template files.
- Store configuration in-package (globally) or elsewhere (dependency injection).
- Doesn't require a set directory layout.
- Allows for inheriting some templates, such as headers and footers.
- Group source templates files into subdirectories for organization.
- Allows for using the same template name as long as each is in a separate subdirectory.

## Getting Started:
With your templates in a directory structure similar to as follows:
```
/path/to/templates/
├─ header.html
├─ footer.html
├─ docs/
│  ├─ index.html
│  ├─ faq.html
├─ app/
│  ├─ index.html
│  ├─ users.html
```

1) Initialize your configuration using `NewConfig()` or `NewOnDiskConfig` if you want to store your configuration elsewhere; or `DefaultConfig` or `DefaultOnDiskConfig` if you want to use the globally stored configuration.
    
2) Call `Build()` to validate your configuration and parse your template files.

3) Call `Show(w, dir, template, interface{})` to render your parsed template and show it to the user. See more info below.

## Using Embedded Files:
This package can work the files embedded via the `embded` package. You *must* have already "read" the embedded files using code similar to below *prior* to providing the `embed.FS` object to this package. Note that the path *must* use a forward slash separator!

```golang
package main

//go:embed path/to/templates
var embeddedFiles embed.FS

func init() {
    c := NewEmbeddedConfig(embeddedFiles, "path/to/templates", []string{"app", "docs"})
    err := c.Build()
    if err != nil {
        log.Fatal(err)
        return
    }
}
```

## Rendering a Page:
Use code similar to the following, providing your subdirectory the template is located in, the template's name (i.e.: filename), and any data you want to inject into the template for modifying the HTML or displaying.

```golang
    func MyHttpHandler(w http.ResponseWriter, r *http.Request) {
    //nil is an interface{} value and can be replaced with anything to
    //be injected into your templates under the .Data field.
    subdir := "app"
    page := "users"
    data := struct{
        Fname  string
        Age    int
        Active bool
    }{"Mike", 46, true}
    templates.Show(w, subdir, page, data)
}
```

The `data` parameter can be any data you want, or `nil`, and is available at the `{{.InjectedData}}` field.

This package also returns some other information for use when rendering pages:

- **{{.Development}}:** boolean field useful for showing a "dev" banner or altering what script are included for diagnostics.
- **{{.UseLocalFiles}}:** boolean field used for toggling CSS or JS files between files served from CDN/internet or files served from your local web server/app.
- **{{.CacheBustFiles}}:** a set of key-value pairs of the original filename to the filename of the cache busting version of a file for use in replacing the known original filename with the generated cache busting filename. See notes below.

## Cache Busting:
This package does not force any style or type of cache busting upon you. You simply need to provide the original file's name and name of the cache busting version of the file as a key-value map. 

You *must* know the original file name of your files (i.e.: script.min.js). If not, this won't work for you. However, if you do, and you have programatic access to the name of the cache busting version of the file, then you can provide the filename pairing and handle the replacement of the original filename in each of your HTML templates using code similar to as follows:

```html
<html>
  <head>
    {{$originalFile := "styles.min.css"}}
	{{$cacheBustFiles := .CacheBustFiles}}

	{{/*If the key "styles.min.css" exists in $cacheBustFiles, then the associated cache-busted filename will be returned as {{.}}. */}}
	{{with index $cacheBustFiles $originalFile}}
	  {{$cacheBustedFile := .}}
	  <link rel="stylesheet" href="/static/css/{{$cacheBustedFile}}">
    {{else}}
      <link rel="stylesheet" href="/static/css/{{$originalFile}}">
    {{end}}
  </head>
</html>
```
