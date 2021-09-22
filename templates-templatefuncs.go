/*
Package templates2 handles parsing and rendering HTML. This more-or-less wraps the golang
'html/template' package with some tooling for storing the parsed templates, showing
a requested template, and using source HTML stored in on-disk or embedded files.

This file defines template level functions for use in templates. These funcs are additional
functions used in {{ }} declarations inside templates. To use one or more of these funcs, you
must add them to the FuncMap field on your config prior to calling Build().

Function names start with "Func" so that they can be easily differentiated from other funcs
in this package when a user is looking for funcs to add to FuncMap.

Note that function names are capitalized for exporting. This is only done so that users
can add one or more of these funcs to a FuncMap as needed on their config. These funcs are
not meant for use outside of templates, i.e. not in golang code. The funcs are named with
"Func" in attempts to make these funcs stand out a bit to prevent misuse.

For more info, see https://pkg.go.dev/text/template#hdr-Functions
*/

package templates

import (
	"strings"
	"time"
)

//FuncIndexOf returns the position of needle in haystack. If needle does not exist in haystack,
//-1 is returned.
func FuncIndexOf(needle, haystack string) int {
	return strings.Index(haystack, needle)
}

//FuncDateReformat is used to transform a date from the yyyy-mm-dd format to another format in
//templates.
func FuncDateReformat(date, format string) (d string) {
	//convert to time.Time so we can reformat
	//Assume that all dates being provided are in yyyy-mm-dd format.
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		//just return original value if error occurs
		d = date
		return
	}

	//reformat time.Time to requested format
	d = t.Format(format)
	return
}

//FuncAddInt performs addition.
func FuncAddInt(x, y int) (z int) {
	return x + y
}
