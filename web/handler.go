package web

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"podpal/internal/downloader"
	"podpal/internal/imgconv"
	"podpal/internal/models"
	"podpal/internal/patcher"
)

type pendingResult struct {
	zipData   []byte
	modelKey  string
	createdAt time.Time
}

// Handler holds HTTP handlers and their dependencies.
type Handler struct {
	tmpl    *template.Template
	dl      *downloader.Downloader
	results sync.Map // map[string]*pendingResult
}

// New creates a Handler with parsed templates and a downloader.
func New(tmpl *template.Template, dl *downloader.Downloader) *Handler {
	h := &Handler{tmpl: tmpl, dl: dl}
	go h.cleanupLoop()
	return h
}

// Routes returns a chi router with all routes mounted.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.index)
	r.Post("/patch", h.patch)
	r.Get("/download/{id}", h.download)
	return r
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Models []models.IPod
	}{
		Models: models.All(),
	}
	if err := h.tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		log.Printf("template error: %v", err)
	}
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		h.renderError(w, "Failed to parse form. Max file size is 10MB.")
		return
	}

	modelKey := r.FormValue("model")
	model := models.ByKey(modelKey)
	if model == nil {
		h.renderError(w, "Unknown iPod model selected.")
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		h.renderError(w, "Please select an image file to upload.")
		return
	}
	defer file.Close()

	// Process image
	userImg, err := imgconv.CropAndResize(file, model.LogoWidth, model.LogoHeight)
	if err != nil {
		h.renderError(w, fmt.Sprintf("Failed to process image: %v", err))
		return
	}

	// Download firmware
	firmware, err := h.dl.GetFirmware(modelKey)
	if err != nil {
		h.renderError(w, fmt.Sprintf("Failed to download Rockbox firmware: %v", err))
		return
	}

	// Patch
	result, err := patcher.Patch(firmware, userImg, model)
	if err != nil {
		h.renderError(w, fmt.Sprintf("Patching failed: %v", err))
		return
	}

	// Build ZIP
	zipData, err := h.buildZIP(result, model)
	if err != nil {
		h.renderError(w, fmt.Sprintf("Failed to build ZIP: %v", err))
		return
	}

	// Store with UUID, 5-min TTL
	id := generateID()
	h.results.Store(id, &pendingResult{
		zipData:   zipData,
		modelKey:  modelKey,
		createdAt: time.Now(),
	})

	// Return HTMX partial with download link
	h.renderSuccess(w, id, model)
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	val, ok := h.results.Load(id)
	if !ok {
		http.Error(w, "Download expired or not found. Please patch again.", http.StatusNotFound)
		return
	}

	pr := val.(*pendingResult)

	// Clean up after download
	h.results.Delete(id)

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="rockbox-%s-patched.zip"`, pr.modelKey))
	w.Write(pr.zipData)
}

func (h *Handler) buildZIP(result *patcher.PatchResult, model *models.IPod) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Patched firmware
	f, err := zw.Create("rockbox.ipod")
	if err != nil {
		return nil, err
	}
	if _, err := f.Write(result.Patched); err != nil {
		return nil, err
	}

	// Original backup
	f, err = zw.Create("rockbox.ipod.original")
	if err != nil {
		return nil, err
	}
	if _, err := f.Write(result.Original); err != nil {
		return nil, err
	}

	// Restore instructions
	f, err = zw.Create("RESTORE.txt")
	if err != nil {
		return nil, err
	}
	restoreText := fmt.Sprintf(`RESTORE INSTRUCTIONS — %s
========================================

If something goes wrong with the patched firmware, follow these steps
to restore the original Rockbox boot logo:

1. Connect your iPod to your computer via USB.
2. The iPod will mount as a drive (e.g., "IPOD").
3. Copy "rockbox.ipod.original" from this ZIP to the iPod:
      <iPod>/.rockbox/rockbox.ipod
   (Rename it from "rockbox.ipod.original" to "rockbox.ipod")
4. Safely eject/unmount the iPod.
5. Reboot the iPod (hold MENU + SELECT for 6 seconds).

The original Rockbox boot logo will be restored.

If Rockbox fails to load entirely:
- Boot into disk mode (hold SELECT + PLAY immediately after reboot)
- Connect via USB and repeat steps 3-5

For more help: https://www.rockbox.org/wiki/IpodFAQ
`, model.Description)
	if _, err := f.Write([]byte(restoreText)); err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (h *Handler) renderError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<div class="result error"><p>%s</p></div>`, template.HTMLEscapeString(msg))
}

func (h *Handler) renderSuccess(w http.ResponseWriter, id string, model *models.IPod) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<div class="result success">
	<p>Patched successfully for <strong>%s</strong>!</p>
	<a href="/download/%s" class="download-btn">Download Patched ZIP</a>
	<p class="hint">ZIP contains: patched rockbox.ipod + original backup + restore instructions</p>
	<p class="hint">This download link expires in 5 minutes.</p>
</div>`, template.HTMLEscapeString(model.Description), template.HTMLEscapeString(id))
}

func (h *Handler) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		now := time.Now()
		h.results.Range(func(key, value any) bool {
			pr := value.(*pendingResult)
			if now.Sub(pr.createdAt) > 5*time.Minute {
				h.results.Delete(key)
			}
			return true
		})
	}
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
