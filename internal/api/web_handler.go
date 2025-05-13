package api

import (
        "html/template"
        "net/http"
        "os"
        "path/filepath"

        "github.com/go-chi/chi/v5"
        "github.com/hashicorp/go-hclog"
)

// WebHandler handles the web UI
type WebHandler struct {
        logger     hclog.Logger
        templates  *template.Template
        staticPath string
}

// NewWebHandler creates a new web UI handler
func NewWebHandler(logger hclog.Logger) *WebHandler {
        return &WebHandler{
                logger:     logger,
                staticPath: "web/static",
        }
}

// RegisterRoutes sets up the routes for the web UI
func (h *WebHandler) RegisterRoutes(r chi.Router) {
        // Load templates
        err := h.loadTemplates()
        if err != nil {
                h.logger.Error("Failed to load templates", "error", err)
                return
        }

        // Setup static file serving
        fs := http.FileServer(http.Dir(h.staticPath))
        r.Handle("/static/*", http.StripPrefix("/static/", fs))

        // Setup page routes
        r.Get("/", h.handleIndex)
        r.Get("/logs", h.handleLogs)
        r.Get("/analytics", h.handleAnalytics)
        r.Get("/settings", h.handleSettings)
}

// loadTemplates loads HTML templates from the templates directory
func (h *WebHandler) loadTemplates() error {
        templatesDir := "web/templates"
        
        // Create a new template with functions
        tmpl := template.New("").Funcs(template.FuncMap{
                "formatTimestamp": func(timestamp string) string {
                        // This could be expanded to format timestamps nicely
                        return timestamp
                },
        })
        
        // Walk through the templates directory and parse all files
        err := filepath.Walk(templatesDir, func(path string, info os.FileInfo, err error) error {
                if err != nil {
                        return err
                }
                
                // Skip if not a file or not an HTML file
                if info.IsDir() || filepath.Ext(path) != ".html" {
                        return nil
                }
                
                // Read and parse template file
                b, err := os.ReadFile(path)
                if err != nil {
                        return err
                }
                
                name := filepath.Base(path)
                _, err = tmpl.New(name).Parse(string(b))
                return err
        })
        
        if err != nil {
                return err
        }
        
        h.templates = tmpl
        return nil
}

// handleIndex renders the main index page
func (h *WebHandler) handleIndex(w http.ResponseWriter, r *http.Request) {
        if h.templates == nil {
                http.Error(w, "Templates not loaded", http.StatusInternalServerError)
                return
        }
        
        err := h.templates.ExecuteTemplate(w, "index.html", nil)
        if err != nil {
                h.logger.Error("Failed to render template", "error", err)
                http.Error(w, "Failed to render page", http.StatusInternalServerError)
        }
}

// handleLogs renders the logs page
func (h *WebHandler) handleLogs(w http.ResponseWriter, r *http.Request) {
        // For now, just redirect to index
        http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// handleAnalytics renders the analytics page
func (h *WebHandler) handleAnalytics(w http.ResponseWriter, r *http.Request) {
        // For now, just redirect to index
        http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// handleSettings renders the settings page
func (h *WebHandler) handleSettings(w http.ResponseWriter, r *http.Request) {
        // For now, just redirect to index
        http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}