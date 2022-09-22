package frontend

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"rpjosh.de/ncDocConverter/pkg/logger"
	"rpjosh.de/ncDocConverter/web"
)

func (app *Frontend) routes(router *chi.Mux) {

	if app.Config.Server.DevelopmentServer {
		app.renderForDev(router)
	} else {
		app.renderForProd(router)
	}

	router.Route("/", func(mainRouter chi.Router) {
		mainRouter.Get("/", app.home)
	})

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	})
}

// Runs the vite Server as an sub process for serving the files with the hot reload function.
// The src directory will also be exposed from within this server for the asserts
func (app *Frontend) renderForDev(router *chi.Mux) {
	// serve assets from src folder
	FileServer(router, "/src", http.Dir("./web/app/src"))

	logger.Info("[DEV] Started vite dev server on http://localhost:%d", app.Config.Server.DevelopmentServerPort)
	vite := exec.Command(filepath.Join(".", "node_modules", ".bin", "vite"), "--mode", "development", "--port", fmt.Sprint(app.Config.Server.DevelopmentServerPort))
	vite.Dir = "./web/app/"
	vite.Stdout = os.Stdout
	vite.Stderr = os.Stderr
	err := vite.Start()
	if err != nil {
		logger.Error("Failed to start the vite development server: %s", err)
	}
}

// This serves all the needed files from the ebedded file system within the binary
// -> no additional WebServer
func (app *Frontend) renderForProd(router *chi.Mux) {
	staticFolder, err := fs.Sub(web.FrontendFiles, "app/dist/assets")
	if err != nil {
		logger.Fatal("Cannot access the embedded directory 'src'. %s", err)
	}
	FileServer(router, "/assets", http.FS(staticFolder))
}

func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		logger.Info("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}

		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
