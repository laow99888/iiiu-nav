package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"iiiu-nav/internal/imagecleanup"
	"iiiu-nav/internal/imagestore"
	"iiiu-nav/internal/linkmeta"
	"iiiu-nav/internal/navigation"
)

type metadataHandler struct {
	links      LinkManager
	recognizer *linkmeta.Recognizer
	logos      *imagestore.Store
	background context.Context
	logger     *slog.Logger
}

type recognitionRequest struct {
	URL string `json:"url"`
}

type recognitionResponse struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	IconSource  navigation.IconSource `json:"iconSource"`
	IconValue   string                `json:"iconValue"`
}

type bulkRecognitionResponse struct {
	Updated int `json:"updated"`
	Failed  int `json:"failed"`
}

func registerMetadataRoutes(mux *http.ServeMux, authenticator Authenticator, links LinkManager, recognizer *linkmeta.Recognizer, logos *imagestore.Store, background context.Context, logger *slog.Logger) *metadataHandler {
	handler := &metadataHandler{links: links, recognizer: recognizer, logos: logos, background: background, logger: logger}
	mux.Handle("POST /api/metadata/recognize", RequireAdmin(authenticator, http.HandlerFunc(handler.recognize)))
	mux.Handle("POST /api/links/{id}/refresh", RequireAdmin(authenticator, http.HandlerFunc(handler.refresh)))
	mux.Handle("POST /api/links/refresh", RequireAdmin(authenticator, http.HandlerFunc(handler.refreshAll)))
	mux.Handle("POST /api/logos", RequireAdmin(authenticator, http.HandlerFunc(handler.upload)))
	return handler
}

func (handler *metadataHandler) refreshImported(ids []int64) {
	if handler == nil || len(ids) == 0 {
		return
	}
	ids = append([]int64(nil), ids...)
	go func() {
		for _, id := range ids {
			// Each link gets a bounded window derived from the cancellable
			// background context, so shutdown aborts pending refreshes
			// instead of querying a closing data store.
			ctx, cancel := context.WithTimeout(handler.background, 10*time.Second)
			link, err := handler.links.Link(ctx, id)
			if err == nil {
				_, _ = handler.refreshLink(ctx, link)
			}
			cancel()
		}
	}()
}

func (handler *metadataHandler) recognize(writer http.ResponseWriter, request *http.Request) {
	var body recognitionRequest
	if err := decodeJSON(writer, request, &body); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request")
		return
	}
	result, err := handler.recognizer.Recognize(request.Context(), normalizeHTTPURL(body.URL))
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, metadataRecognitionErrorCode(err))
		return
	}
	response := handler.responseFromRecognition(body.URL, result, "", navigation.IconSourceGenerated)
	writeJSON(writer, http.StatusOK, response)
}

func (handler *metadataHandler) refresh(writer http.ResponseWriter, request *http.Request) {
	id, ok := positiveID(writer, request.PathValue("id"), "link_id_invalid")
	if !ok {
		return
	}
	link, err := handler.links.Link(request.Context(), id)
	if errors.Is(err, navigation.ErrLinkNotFound) {
		writeError(writer, http.StatusNotFound, "link_not_found")
		return
	}
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "link_read_failed")
		return
	}
	updated, err := handler.refreshLink(request.Context(), link)
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, metadataRecognitionErrorCode(err))
		return
	}
	writeJSON(writer, http.StatusOK, linkResponseFromRecord(updated))
}

func metadataRecognitionErrorCode(err error) string {
	if errors.Is(err, linkmeta.ErrUnsafeURL) {
		return "metadata_target_not_public"
	}
	return "metadata_recognition_failed"
}

func (handler *metadataHandler) refreshAll(writer http.ResponseWriter, request *http.Request) {
	clearResponseDeadline(writer)
	links, err := handler.links.Links(request.Context())
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "links_read_failed")
		return
	}
	result := bulkRecognitionResponse{}
	for _, link := range links {
		if _, err := handler.refreshLink(request.Context(), link); err != nil {
			result.Failed++
		} else {
			result.Updated++
		}
	}
	writeJSON(writer, http.StatusOK, result)
}

func (handler *metadataHandler) refreshLink(ctx context.Context, link navigation.Link) (navigation.Link, error) {
	result, err := handler.recognizer.Recognize(ctx, link.URL)
	if err != nil {
		return navigation.Link{}, err
	}
	response := handler.responseFromRecognition(link.URL, result, link.IconValue, link.IconSource)
	name := response.Name
	if name == "" {
		name = link.Name
	}
	description := response.Description
	if description == "" {
		description = link.Description
	}
	iconSource, iconValue := response.IconSource, response.IconValue
	if link.IconSource == navigation.IconSourceUpload {
		if response.IconValue != "" && response.IconValue != link.IconValue {
			handler.discardLogo(response.IconValue)
		}
		iconSource, iconValue = link.IconSource, link.IconValue
	}
	updated, err := handler.links.UpdateLink(ctx, link.ID, navigation.LinkInput{
		CategoryID: link.CategoryID, Name: name, Description: description, URL: link.URL,
		IconSource: iconSource, IconValue: iconValue, SortOrder: link.SortOrder,
	})
	if err != nil {
		if iconValue != link.IconValue {
			handler.discardLogo(iconValue)
		}
		return navigation.Link{}, err
	}
	if link.IconValue != updated.IconValue {
		imagecleanup.Retire(ctx, handler.logger, handler.logos, handler.links, link.IconValue)
	}
	return updated, nil
}

// discardLogo removes a logo file this handler just saved but that never
// reached the database, so it cannot be referenced by any record.
func (handler *metadataHandler) discardLogo(publicPath string) {
	if err := handler.logos.Remove(publicPath); err != nil {
		handler.logger.Warn("unused logo removal failed", "path", publicPath, "error", err)
	}
}

func (handler *metadataHandler) responseFromRecognition(rawURL string, result linkmeta.Result, currentIcon string, currentSource navigation.IconSource) recognitionResponse {
	name := result.Title
	if name == "" {
		if parsed, err := url.Parse(rawURL); err == nil {
			name = parsed.Hostname()
		}
	}
	response := recognitionResponse{Name: name, Description: result.Description, IconSource: currentSource, IconValue: currentIcon}
	if len(result.Icon) > 0 {
		if publicPath, err := handler.logos.Save(bytes.NewReader(result.Icon)); err == nil {
			response.IconSource = navigation.IconSourceAuto
			response.IconValue = publicPath
			return response
		}
	}
	if response.IconValue == "" {
		response.IconSource = navigation.IconSourceGenerated
		response.IconValue = firstRune(name)
	}
	return response
}

func (handler *metadataHandler) upload(writer http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(writer, request.Body, imagestore.LogoMaxBytes+(64<<10))
	file, _, err := request.FormFile("logo")
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, "logo_invalid")
		return
	}
	defer file.Close()
	publicPath, err := handler.logos.Save(file)
	if err != nil {
		writeError(writer, http.StatusUnprocessableEntity, "logo_invalid")
		return
	}
	writeJSON(writer, http.StatusCreated, map[string]string{"url": publicPath})
}
