package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/tcit/challenge-full-stack/backend/internal/domain"
	"github.com/tcit/challenge-full-stack/backend/internal/repository"
	"github.com/tcit/challenge-full-stack/backend/internal/service"
)

// PostHandler maneja las rutas HTTP REST para el recurso /posts.
type PostHandler struct {
	postService *service.PostService
}

// NuevoPostHandler construye una instancia de PostHandler.
func NuevoPostHandler(postService *service.PostService) *PostHandler {
	return &PostHandler{
		postService: postService,
	}
}

// RegistrarRutas registra los endpoints REST en el multiplexor HTTP.
func (h *PostHandler) RegistrarRutas(mux *http.ServeMux) {
	mux.HandleFunc("POST /posts", h.CrearPost)
	mux.HandleFunc("GET /posts", h.ListarPosts)
	mux.HandleFunc("GET /posts/pending", h.ListarPostsPendientes)
	mux.HandleFunc("DELETE /posts/{id}", h.EliminarPost)
}

// CrearPost maneja POST /posts: guarda el post con EstadoProcesamiento = 'PENDIENTE'
// y responde HTTP 201 inmediatamente.
func (h *PostHandler) CrearPost(w http.ResponseWriter, r *http.Request) {
	var solicitud domain.CrearPostRequest
	if err := json.NewDecoder(r.Body).Decode(&solicitud); err != nil {
		responderError(w, http.StatusBadRequest, "El cuerpo de la petición no es un JSON válido", err.Error())
		return
	}

	postCreado, err := h.postService.CrearPost(r.Context(), solicitud)
	if err != nil {
		if errors.Is(err, service.ErrDatosPostInvalidos) {
			responderError(w, http.StatusBadRequest, err.Error(), "")
			return
		}
		log.Printf("Error al consultar Spanner: %v", err)
		responderError(w, http.StatusInternalServerError, "Error al guardar el post en Cloud Spanner", err.Error())
		return
	}

	responderJSON(w, http.StatusCreated, postCreado)
}

// ListarPosts maneja GET /posts: retorna todos los posts desde Cloud Spanner.
func (h *PostHandler) ListarPosts(w http.ResponseWriter, r *http.Request) {
	listaPosts, err := h.postService.ListarPosts(r.Context())
	if err != nil {
		log.Printf("Error al consultar Spanner: %v", err)
		responderError(w, http.StatusInternalServerError, "Error al consultar posts en Cloud Spanner", err.Error())
		return
	}

	responderJSON(w, http.StatusOK, listaPosts)
}

// ListarPostsPendientes maneja GET /posts/pending: retorna los posts con EstadoProcesamiento = 'PENDIENTE'
// en un array JSON (o [] si no hay registros pendientes).
func (h *PostHandler) ListarPostsPendientes(w http.ResponseWriter, r *http.Request) {
	listaPendientes, err := h.postService.ListarPostsPendientes(r.Context())
	if err != nil {
		log.Printf("Error al consultar Spanner: %v", err)
		responderError(w, http.StatusInternalServerError, "Error al consultar posts pendientes en Cloud Spanner", err.Error())
		return
	}
	if listaPendientes == nil {
		listaPendientes = make([]domain.Post, 0)
	}

	responderJSON(w, http.StatusOK, listaPendientes)
}

// EliminarPost maneja DELETE /posts/{id}: elimina el registro en Cloud Spanner y retorna el objeto eliminado.
func (h *PostHandler) EliminarPost(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		responderError(w, http.StatusBadRequest, "El parámetro 'id' es obligatorio", "")
		return
	}

	postEliminado, err := h.postService.EliminarPost(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrPostNoEncontrado) {
			responderError(w, http.StatusNotFound, "El post solicitado no existe", err.Error())
			return
		}
		log.Printf("Error al consultar Spanner: %v", err)
		responderError(w, http.StatusInternalServerError, "Error al eliminar el post en Cloud Spanner", err.Error())
		return
	}

	responderJSON(w, http.StatusOK, postEliminado)
}

func responderJSON(w http.ResponseWriter, codigoHttp int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigoHttp)
	_ = json.NewEncoder(w).Encode(payload)
}

func responderError(w http.ResponseWriter, codigoHttp int, mensaje string, detalle string) {
	respuesta := map[string]string{
		"mensaje": mensaje,
	}
	if detalle != "" {
		respuesta["detalle"] = detalle
	}
	responderJSON(w, codigoHttp, respuesta)
}
