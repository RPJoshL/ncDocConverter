package frontend

import (
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"text/template"

	"rpjosh.de/ncDocConverter/web"
)

type serverConfig struct {
	Version      string
	Development  bool
	SourceServer string
}

// the server config does never change again -> set this once at startup
var serverConf *serverConfig = &serverConfig{
	Version: "1.0.0",
}

type templateData struct {
	Version      string
	ServerConfig *serverConfig
}

// Returns the absolute URL on the WebServer to the given TypeScript file given without the file extension
// main -> http://localhost:4000/assets/main.js
func getJSFile(file string) string {
	if serverConf.Development {
		return serverConf.SourceServer + "src/" + file + ".tsx"
	}

	return serverConf.SourceServer + "assets/" + file + ".js"
}

var functions = template.FuncMap{
	"getJSFile": getJSFile,
}

func (app *Frontend) setServerConfiguration() {
	serverConf.Development = app.Config.Server.DevelopmentServer

	sourceServer := ""
	if serverConf.Development {
		sourceServer = fmt.Sprintf("http://localhost:%d/", app.Config.Server.DevelopmentServerPort)
	} else {
		sourceServer = fmt.Sprintf("http://localhost%s/", app.Config.Server.Address)
	}
	serverConf.SourceServer = sourceServer
}

func (app *Frontend) newTemplateData(r *http.Request) *templateData {
	return &templateData{
		ServerConfig: serverConf,
	}
}

// Initializes a new cache containing all templates of the application
// from the embedded file system
func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := fs.Glob(&web.TemplateFiles, "template/pages/*.tmpl.html")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		patterns := []string{
			"template/base.tmpl.html",
			"template/vitejs.tmpl.html",
			page,
		}

		ts, err := template.New(name).Funcs(functions).ParseFS(web.TemplateFiles, patterns...)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}
