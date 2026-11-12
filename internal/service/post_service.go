package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/tcit/challenge-full-stack/backend/internal/domain"
	"github.com/tcit/challenge-full-stack/backend/internal/repository"
)

// ErrDatosPostInvalidos se retorna cuando el campo Nombre está vacío.
var ErrDatosPostInvalidos = errors.New("el campo 'nombre' del post es obligatorio")

// PostService gestiona exclusivamente la lógica síncrona y rápida contra Cloud Spanner.
// El procesamiento de Google Drive y Gemini IA es delegado al orquestador en Symfony,
// el cual consume los registros con EstadoProcesamiento = 'PENDIENTE'.
type PostService struct {
	postRepo repository.PostRepository
}

// NuevoPostService construye una instancia de PostService.
func NuevoPostService(postRepo repository.PostRepository) *PostService {
	return &PostService{
		postRepo: postRepo,
	}
}

// CrearPost genera un UUIDv4, guarda el Post en Cloud Spanner con EstadoProcesamiento = 'PENDIENTE'
// y retorna inmediatamente el registro creado sin bloquear la respuesta HTTP.
func (s *PostService) CrearPost(ctx context.Context, solicitud domain.CrearPostRequest) (domain.Post, error) {
	nombreLimpio := strings.TrimSpace(solicitud.Nombre)
	descripcionLimpia := strings.TrimSpace(solicitud.Descripcion)
	adjuntoRecibido := strings.TrimSpace(solicitud.Adjunto)

	if nombreLimpio == "" {
		return domain.Post{}, ErrDatosPostInvalidos
	}

	nuevoPost := domain.Post{
		ID:                  uuid.NewString(),
		Nombre:              nombreLimpio,
		Descripcion:         descripcionLimpia,
		Adjunto:             adjuntoRecibido,
		DatosExtraidosIA:    "",
		EstadoProcesamiento: domain.EstadoPendiente,
	}

	postGuardado, err := s.postRepo.CrearPost(ctx, nuevoPost)
	if err != nil {
		return domain.Post{}, fmt.Errorf("error guardando post en Cloud Spanner: %w", err)
	}

	return postGuardado, nil
}

// ListarPosts retorna todos los posts almacenados en Cloud Spanner.
func (s *PostService) ListarPosts(ctx context.Context) ([]domain.Post, error) {
	return s.postRepo.ListarPosts(ctx)
}

// ListarPostsPendientes retorna todos los posts con EstadoProcesamiento = 'PENDIENTE'.
func (s *PostService) ListarPostsPendientes(ctx context.Context) ([]domain.Post, error) {
	return s.postRepo.ListarPostsPendientes(ctx)
}

// EliminarPost elimina el registro por Id en Cloud Spanner y retorna el post eliminado.
func (s *PostService) EliminarPost(ctx context.Context, id string) (domain.Post, error) {
	idLimpio := strings.TrimSpace(id)
	if idLimpio == "" {
		return domain.Post{}, repository.ErrPostNoEncontrado
	}
	return s.postRepo.EliminarPost(ctx, idLimpio)
}
