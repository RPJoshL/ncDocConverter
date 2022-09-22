package frontend

import (
	"bytes"
	"fmt"
	"net/http"
)

func (app *Frontend) home(w http.ResponseWriter, r *http.Request) {
	app.render(w, http.StatusOK, "main.tmpl.html", app.newTemplateData(r))
}

func (app *Frontend) render(w http.ResponseWriter, status int, page string, data *templateData) {
	ts, ok := app.templateCache[page]
	if !ok {
		err := fmt.Errorf("the template %s does not exist", page)
		app.serverError(w, err)
		return
	}

	buf := new(bytes.Buffer)

	err := ts.ExecuteTemplate(buf, "base", data)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.WriteHeader(status)
	buf.WriteTo(w)
}
