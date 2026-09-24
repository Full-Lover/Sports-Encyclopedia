package webexperience

import (
	"bytes"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/full-lover/sports-encyclopedia/internal/publishedatlas"
)

type teamPageData struct {
	Team         publishedatlas.TeamPageDocument
	OfficialSite string
	Capacity     string
	OpenedYear   string
	VenueSource  string
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
		data := teamPageData{
			Team:         *result.Team,
			OfficialSite: secureExternalURL(result.Team.OfficialWebsiteURL),
			VenueSource:  secureExternalURL(result.Team.VenueFactsSourceURL),
		}
		if result.Team.RegularGameCapacity > 0 {
			data.Capacity = formatCount(result.Team.RegularGameCapacity)
		}
		if year := result.Team.OpenedYear; year >= 1800 && year <= time.Now().Year() {
			data.OpenedYear = strconv.Itoa(year)
		}
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

func secureExternalURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return ""
	}
	return parsed.String()
}

func formatCount(value int) string {
	digits := strconv.Itoa(value)
	for index := len(digits) - 3; index > 0; index -= 3 {
		digits = digits[:index] + "," + digits[index:]
	}
	return digits
}
