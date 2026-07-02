//go:build embed

package web

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

const (
	// NonceHTMLPlaceholder is the placeholder for nonce in HTML script tags
	NonceHTMLPlaceholder = "__CSP_NONCE_VALUE__"
)

//go:embed all:dist
var frontendFS embed.FS

//go:embed all:image2-dist
var image2FS embed.FS

// PublicSettingsProvider is an interface to fetch public settings
type PublicSettingsProvider interface {
	GetPublicSettingsForInjection(ctx context.Context) (any, error)
}

// FrontendServer serves the embedded frontend with settings injection
type FrontendServer struct {
	distFS           fs.FS
	fileServer       http.Handler
	baseHTML         []byte
	cache            *HTMLCache
	settings         PublicSettingsProvider
	overrideDir      string // local file override directory
	image2FileServer http.Handler
	image2BaseHTML   []byte
	image2DistFS     fs.FS
}

// NewFrontendServer creates a new frontend server with settings injection
func NewFrontendServer(settingsProvider PublicSettingsProvider) (*FrontendServer, error) {
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		return nil, err
	}

	file, err := distFS.Open("index.html")
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	baseHTML, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	cache := NewHTMLCache()
	cache.SetBaseHTML(baseHTML)

	image2DistFS, err := fs.Sub(image2FS, "image2-dist")
	if err != nil {
		return nil, err
	}

	image2File, err := image2DistFS.Open("index.html")
	if err != nil {
		return nil, err
	}
	defer func() { _ = image2File.Close() }()

	image2BaseHTML, err := io.ReadAll(image2File)
	if err != nil {
		return nil, err
	}

	return &FrontendServer{
		distFS:           distFS,
		fileServer:       http.FileServer(http.FS(distFS)),
		baseHTML:         baseHTML,
		cache:            cache,
		settings:         settingsProvider,
		overrideDir:      filepath.Join("data", "public"),
		image2FileServer: http.StripPrefix("/image2/", http.FileServer(http.FS(image2DistFS))),
		image2BaseHTML:   image2BaseHTML,
		image2DistFS:     image2DistFS,
	}, nil
}

// InvalidateCache invalidates the HTML cache (call when settings change)
func (s *FrontendServer) InvalidateCache() {
	if s != nil && s.cache != nil {
		s.cache.Invalidate()
	}
}

// Middleware returns the Gin middleware handler
func (s *FrontendServer) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if shouldBypassEmbeddedFrontend(path) {
			c.Next()
			return
		}

		if s.serveImage2(c) {
			return
		}

		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		if cleanPath == "index.html" || !s.fileExists(cleanPath) {
			s.serveIndexHTML(c)
			return
		}

		if s.tryServeOverride(c, cleanPath) {
			return
		}

		s.fileServer.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}

func (s *FrontendServer) serveImage2(c *gin.Context) bool {
	path := strings.TrimPrefix(c.Request.URL.Path, "/")
	if path != "image2" && !strings.HasPrefix(path, "image2/") {
		return false
	}

	cleanPath := strings.TrimPrefix(path, "image2")
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	if cleanPath == "" {
		cleanPath = "index.html"
	}

	if cleanPath == "index.html" || !s.image2FileExists(cleanPath) {
		s.serveImage2IndexHTML(c)
		return true
	}

	if tryServeOverrideFile(c, filepath.Join(s.overrideDir, "image2"), cleanPath) {
		return true
	}

	originalPath := c.Request.URL.Path
	c.Request.URL.Path = "/image2/" + cleanPath
	defer func() { c.Request.URL.Path = originalPath }()
	s.image2FileServer.ServeHTTP(c.Writer, c.Request)
	c.Abort()
	return true
}

func (s *FrontendServer) image2FileExists(path string) bool {
	file, err := s.image2DistFS.Open(path)
	if err != nil {
		return false
	}
	_ = file.Close()
	return true
}

func (s *FrontendServer) fileExists(path string) bool {
	file, err := s.distFS.Open(path)
	if err != nil {
		return false
	}
	_ = file.Close()
	return true
}

// tryServeOverride checks if a local override file exists and serves it.
// Files in overrideDir take precedence over embedded files.
func (s *FrontendServer) tryServeOverride(c *gin.Context, cleanPath string) bool {
	if s.overrideDir == "" {
		return false
	}
	filePath := filepath.Join(s.overrideDir, filepath.Clean("/"+cleanPath))
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return false
	}
	c.File(filePath)
	c.Abort()
	return true
}

func (s *FrontendServer) serveIndexHTML(c *gin.Context) {
	nonce := middleware.GetNonceFromContext(c)

	cached := s.cache.Get()
	if cached != nil {
		content := replaceNoncePlaceholder(cached.Content, nonce)
		c.Header("ETag", cached.ETag)
		setIndexHTMLNoStoreHeaders(c)
		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
		c.Abort()
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	settings, err := s.settings.GetPublicSettingsForInjection(ctx)
	if err != nil {
		setIndexHTMLNoStoreHeaders(c)
		c.Data(http.StatusOK, "text/html; charset=utf-8", s.baseHTML)
		c.Abort()
		return
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		setIndexHTMLNoStoreHeaders(c)
		c.Data(http.StatusOK, "text/html; charset=utf-8", s.baseHTML)
		c.Abort()
		return
	}

	rendered := s.injectSettings(settingsJSON)
	s.cache.Set(rendered, settingsJSON)

	content := replaceNoncePlaceholder(rendered, nonce)
	cached = s.cache.Get()
	if cached != nil {
		c.Header("ETag", cached.ETag)
	}
	setIndexHTMLNoStoreHeaders(c)
	c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	c.Abort()
}

func (s *FrontendServer) serveImage2IndexHTML(c *gin.Context) {
	file, err := s.image2DistFS.Open("index.html")
	if err != nil {
		c.String(http.StatusNotFound, "Image playground not found")
		c.Abort()
		return
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to read image2 index.html")
		c.Abort()
		return
	}
	setIndexHTMLNoStoreHeaders(c)
	c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	c.Abort()
}

func (s *FrontendServer) injectSettings(settingsJSON []byte) []byte {
	script := []byte(`<script nonce="` + NonceHTMLPlaceholder + `">window.__APP_CONFIG__=` + string(settingsJSON) + `;</script>`)

	headClose := []byte("</head>")
	result := bytes.Replace(s.baseHTML, headClose, append(script, headClose...), 1)
	result = injectSiteTitle(result, settingsJSON)
	return result
}

func injectSiteTitle(html, settingsJSON []byte) []byte {
	var cfg struct {
		SiteName string `json:"site_name"`
	}
	if err := json.Unmarshal(settingsJSON, &cfg); err != nil || cfg.SiteName == "" {
		return html
	}

	titleStart := bytes.Index(html, []byte("<title>"))
	titleEnd := bytes.Index(html, []byte("</title>"))
	if titleStart == -1 || titleEnd == -1 || titleEnd <= titleStart {
		return html
	}

	newTitle := []byte("<title>" + cfg.SiteName + " - AI API Gateway</title>")
	var buf bytes.Buffer
	buf.Write(html[:titleStart])
	buf.Write(newTitle)
	buf.Write(html[titleEnd+len("</title>"):])
	return buf.Bytes()
}

func replaceNoncePlaceholder(html []byte, nonce string) []byte {
	return bytes.ReplaceAll(html, []byte(NonceHTMLPlaceholder), []byte(nonce))
}

// ServeEmbeddedFrontend returns a middleware for serving embedded frontend
// This is the legacy function for backward compatibility when no settings provider is available
func ServeEmbeddedFrontend() gin.HandlerFunc {
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		panic("failed to get dist subdirectory: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(distFS))
	overrideDir := filepath.Join("data", "public")

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if shouldBypassEmbeddedFrontend(path) {
			c.Next()
			return
		}

		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		if file, err := distFS.Open(cleanPath); err == nil {
			_ = file.Close()
			if tryServeOverrideFile(c, overrideDir, cleanPath) {
				return
			}
			fileServer.ServeHTTP(c.Writer, c.Request)
			c.Abort()
			return
		}

		serveIndexHTML(c, distFS)
	}
}

// tryServeOverrideFile is a standalone version of tryServeOverride for legacy usage.
func tryServeOverrideFile(c *gin.Context, overrideDir, cleanPath string) bool {
	if overrideDir == "" {
		return false
	}
	filePath := filepath.Join(overrideDir, filepath.Clean("/"+cleanPath))
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return false
	}
	c.File(filePath)
	c.Abort()
	return true
}

func shouldBypassEmbeddedFrontend(path string) bool {
	trimmed := strings.TrimSpace(path)
	return strings.HasPrefix(trimmed, "/api/") ||
		strings.HasPrefix(trimmed, "/v1/") ||
		strings.HasPrefix(trimmed, "/v1beta/") ||
		strings.HasPrefix(trimmed, "/backend-api/") ||
		strings.HasPrefix(trimmed, "/antigravity/") ||
		strings.HasPrefix(trimmed, "/setup/") ||
		trimmed == "/health" ||
		trimmed == "/responses" ||
		strings.HasPrefix(trimmed, "/responses/") ||
		strings.HasPrefix(trimmed, "/images/")
}

func serveIndexHTML(c *gin.Context, fsys fs.FS) {
	file, err := fsys.Open("index.html")
	if err != nil {
		c.String(http.StatusNotFound, "Frontend not found")
		c.Abort()
		return
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(file)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to read index.html")
		c.Abort()
		return
	}

	setIndexHTMLNoStoreHeaders(c)
	c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	c.Abort()
}

func setIndexHTMLNoStoreHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
}

func HasEmbeddedFrontend() bool {
	_, err := frontendFS.ReadFile("dist/index.html")
	return err == nil
}
