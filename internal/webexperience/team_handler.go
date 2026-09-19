package webexperience

import (
	"bytes"
	"errors"
	"html/template"
	"net/http"
	"net/url"

	"github.com/full-lover/sports-encyclopedia/internal/publishedatlas"
)

type teamPageData struct {
	Team         publishedatlas.TeamPageDocument
	OfficialSite string
}

func registerTeamRoute(mux *http.ServeMux, reader publishedatlas.Reader) {
	page := template.Must(template.ParseFS(templates, "templates/team.html"))
	mux.HandleFunc("GET /teams/{slug}", func(response http.ResponseWriter, request *http.Request) {
		slug := request.PathValue("slug")
		if !validTeamSlug(slug) {
			http.NotFound(response, request)
			return
		}
		result, err := reader.Read(request.Context(), publishedatlas.ReadRequest{
			Kind: publishedatlas.ReadTeam, Slug: slug,
		})
		if err != nil {
			var fault *publishedatlas.ReadFault
			if errors.As(err, &fault) && fault.Code == publishedatlas.FaultTeamMissing {
				http.NotFound(response, request)
				return
			}
			http.Error(response, "The atlas is temporarily unavailable.", http.StatusServiceUnavailable)
			return
		}
		if result.Team == nil {
			http.Error(response, "The page could not be rendered.", http.StatusInternalServerError)
			return
		}
		var rendered bytes.Buffer
		data := teamPageData{Team: *result.Team, OfficialSite: secureOfficialSite(result.Team.OfficialWebsiteURL)}
		if err := page.ExecuteTemplate(&rendered, "team.html", data); err != nil {
			http.Error(response, "The page could not be rendered.", http.StatusInternalServerError)
			return
		}
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		response.Header().Set("Cache-Control", "no-cache")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = rendered.WriteTo(response)
	})
}

func validTeamSlug(slug string) bool {
	if len(slug) == 0 || len(slug) > 80 || slug[0] == '-' || slug[len(slug)-1] == '-' {
		return false
	}
	for _, character := range slug {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
			return false
		}
	}
	return true
}

func secureOfficialSite(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return ""
	}
	return parsed.String()
}
