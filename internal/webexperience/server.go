package webexperience

import (
	"bytes"
	"embed"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/full-lover/sports-encyclopedia/internal/publishedatlas"
)

//go:embed templates/*.html
var templates embed.FS

type pageData struct {
	Title string
}

func New(staticDir string, atlasReader publishedatlas.Reader) http.Handler {
	home := template.Must(template.ParseFS(templates, "templates/home.html"))
	mux := http.NewServeMux()
	registerAtlasRoutes(mux, atlasReader)

	mux.Handle("GET /assets/", noCache(http.StripPrefix(
		"/assets/",
		http.FileServer(http.Dir(filepath.Join(staticDir, "assets"))),
	)))
	mux.HandleFunc("GET /healthz", func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "text/plain; charset=utf-8")
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /favicon.ico", func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		response.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /{$}", func(response http.ResponseWriter, _ *http.Request) {
		var rendered bytes.Buffer
		if err := home.ExecuteTemplate(&rendered, "home.html", pageData{
			Title: "Sports Encyclopedia",
		}); err != nil {
			http.Error(response, "The page could not be rendered.", http.StatusInternalServerError)
			return
		}

		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.WriteHeader(http.StatusOK)
		_, _ = rendered.WriteTo(response)
	})

	return mux
}

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-cache")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(response, request)
	})
}
