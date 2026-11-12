package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/spanner"
	"github.com/tcit/challenge-full-stack/backend/internal/domain"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// ErrPostNoEncontrado se retorna cuando no existe ningún post con el Id especificado.
var ErrPostNoEncontrado = errors.New("el post solicitado no existe en Cloud Spanner")

// PostRepository define las operaciones de acceso a datos sobre la tabla Posts en Cloud Spanner.
type PostRepository interface {
	CrearPost(ctx context.Context, post domain.Post) (domain.Post, error)
	ListarPosts(ctx context.Context) ([]domain.Post, error)
	ListarPostsPendientes(ctx context.Context) ([]domain.Post, error)
	EliminarPost(ctx context.Context, id string) (domain.Post, error)
	Cerrar()
}

// SpannerPostRepository implementa PostRepository usando cloud.google.com/go/spanner.
type SpannerPostRepository struct {
	client *spanner.Client
	dsn    string
}

// NuevoSpannerPostRepository crea el cliente de Cloud Spanner leyendo SPANNER_DSN desde las variables de entorno.
func NuevoSpannerPostRepository(ctx context.Context) (*SpannerPostRepository, error) {
	dsn := os.Getenv("SPANNER_DSN")
	if dsn == "" {
		return nil, errors.New("la variable de entorno SPANNER_DSN no está configurada")
	}

	clienteSpanner, err := spanner.NewClient(ctx, dsn)
	if err != nil {
		log.Printf("Error al consultar Spanner (inicialización): %v", err)
		return nil, fmt.Errorf("error conectando a Cloud Spanner (%s): %w", dsn, err)
	}

	log.Printf("[SpannerRepository] Conectado a Cloud Spanner (%s) -> DSN: %s",
		os.Getenv("SPANNER_EMULATOR_HOST"), dsn)

	return &SpannerPostRepository{
		client: clienteSpanner,
		dsn:    dsn,
	}, nil
}

// CrearPost inserta el registro del Post en Cloud Spanner con EstadoProcesamiento = 'PENDIENTE'.
func (r *SpannerPostRepository) CrearPost(ctx context.Context, post domain.Post) (domain.Post, error) {
	columnas := []string{
		"Id",
		"Nombre",
		"Descripcion",
		"Adjunto",
		"DatosExtraidosIA",
		"EstadoProcesamiento",
	}
	valores := []interface{}{
		post.ID,
		post.Nombre,
		aNullString(post.Descripcion),
		aNullString(post.Adjunto),
		aNullString(post.DatosExtraidosIA),
		aNullString(post.EstadoProcesamiento),
	}

	mutacion := spanner.Insert("Posts", columnas, valores)
	if _, err := r.client.Apply(ctx, []*spanner.Mutation{mutacion}); err != nil {
		log.Printf("Error al consultar Spanner (CrearPost): %v", err)
		return domain.Post{}, fmt.Errorf("error insertando post en Cloud Spanner: %w", err)
	}

	return post, nil
}

// ListarPosts retorna todos los registros de la tabla Posts desde Cloud Spanner.
func (r *SpannerPostRepository) ListarPosts(ctx context.Context) ([]domain.Post, error) {
	declaracion := spanner.Statement{
		SQL: `SELECT Id, Nombre, Descripcion, Adjunto, DatosExtraidosIA, EstadoProcesamiento
		      FROM Posts
		      ORDER BY Nombre ASC`,
	}

	iterador := r.client.Single().Query(ctx, declaracion)
	defer iterador.Stop()

	listaPosts := make([]domain.Post, 0)
	for {
		fila, err := iterador.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			log.Printf("Error al consultar Spanner: %v", err)
			return nil, fmt.Errorf("error leyendo fila de Cloud Spanner: %w", err)
		}

		var (
			id                  string
			nombre              string
			descripcion         spanner.NullString
			adjunto             spanner.NullString
			datosExtraidosIA    spanner.NullString
			estadoProcesamiento spanner.NullString
		)

		if errScan := fila.Columns(&id, &nombre, &descripcion, &adjunto, &datosExtraidosIA, &estadoProcesamiento); errScan != nil {
			log.Printf("Error al consultar Spanner (Scan): %v", errScan)
			return nil, fmt.Errorf("error mapeando columnas de Posts: %w", errScan)
		}

		listaPosts = append(listaPosts, domain.Post{
			ID:                  id,
			Nombre:              nombre,
			Descripcion:         descripcion.StringVal,
			Adjunto:             adjunto.StringVal,
			DatosExtraidosIA:    datosExtraidosIA.StringVal,
			EstadoProcesamiento: estadoProcesamiento.StringVal,
		})
	}

	return listaPosts, nil
}

// ListarPostsPendientes consulta todos los posts cuyo EstadoProcesamiento sea 'PENDIENTE',
// escaneando columnas anulables en spanner.NullString y extrayendo .StringVal.
func (r *SpannerPostRepository) ListarPostsPendientes(ctx context.Context) ([]domain.Post, error) {
	declaracion := spanner.Statement{
		SQL: `SELECT Id, Nombre, Descripcion, Adjunto, DatosExtraidosIA, EstadoProcesamiento FROM Posts WHERE EstadoProcesamiento = 'PENDIENTE'`,
	}

	iterador := r.client.Single().Query(ctx, declaracion)
	defer iterador.Stop()

	listaPendientes := make([]domain.Post, 0)
	for {
		fila, err := iterador.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			log.Printf("Error al consultar Spanner: %v", err)
			return nil, fmt.Errorf("error leyendo fila pendiente de Cloud Spanner: %w", err)
		}

		var (
			id                  string
			nombre              string
			descripcion         spanner.NullString
			adjunto             spanner.NullString
			datosExtraidosIA    spanner.NullString
			estadoProcesamiento spanner.NullString
		)

		if errScan := fila.Columns(&id, &nombre, &descripcion, &adjunto, &datosExtraidosIA, &estadoProcesamiento); errScan != nil {
			log.Printf("Error al consultar Spanner (Scan): %v", errScan)
			return nil, fmt.Errorf("error escaneando columnas de post pendiente: %w", errScan)
		}

		postMapeado := domain.Post{
			ID:                  id,
			Nombre:              nombre,
			Descripcion:         descripcion.StringVal,
			Adjunto:             adjunto.StringVal,
			DatosExtraidosIA:    datosExtraidosIA.StringVal,
			EstadoProcesamiento: estadoProcesamiento.StringVal,
		}

		listaPendientes = append(listaPendientes, postMapeado)
	}

	return listaPendientes, nil
}

// EliminarPost elimina el registro por su llave primaria Id dentro de una transacción ReadWrite
// y retorna el objeto eliminado.
func (r *SpannerPostRepository) EliminarPost(ctx context.Context, id string) (domain.Post, error) {
	var postEliminado domain.Post

	_, err := r.client.ReadWriteTransaction(ctx, func(ctxTx context.Context, tx *spanner.ReadWriteTransaction) error {
		columnas := []string{
			"Id",
			"Nombre",
			"Descripcion",
			"Adjunto",
			"DatosExtraidosIA",
			"EstadoProcesamiento",
		}
		fila, errLectura := tx.ReadRow(ctxTx, "Posts", spanner.Key{id}, columnas)
		if errLectura != nil {
			if spanner.ErrCode(errLectura) == codes.NotFound || errors.Is(errLectura, spanner.ErrRowNotFound) {
				return ErrPostNoEncontrado
			}
			log.Printf("Error al consultar Spanner: %v", errLectura)
			return fmt.Errorf("error consultando post a eliminar: %w", errLectura)
		}

		var (
			postId              string
			nombre              string
			descripcion         spanner.NullString
			adjunto             spanner.NullString
			datosExtraidosIA    spanner.NullString
			estadoProcesamiento spanner.NullString
		)

		if errScan := fila.Columns(&postId, &nombre, &descripcion, &adjunto, &datosExtraidosIA, &estadoProcesamiento); errScan != nil {
			log.Printf("Error al consultar Spanner (Scan): %v", errScan)
			return fmt.Errorf("error mapeando columnas del post a eliminar: %w", errScan)
		}

		postEliminado = domain.Post{
			ID:                  postId,
			Nombre:              nombre,
			Descripcion:         descripcion.StringVal,
			Adjunto:             adjunto.StringVal,
			DatosExtraidosIA:    datosExtraidosIA.StringVal,
			EstadoProcesamiento: estadoProcesamiento.StringVal,
		}

		return tx.BufferWrite([]*spanner.Mutation{spanner.Delete("Posts", spanner.Key{id})})
	})

	if err != nil {
		return domain.Post{}, err
	}

	return postEliminado, nil
}

// Cerrar cierra el cliente de sesiones de Cloud Spanner.
func (r *SpannerPostRepository) Cerrar() {
	if r.client != nil {
		r.client.Close()
	}
}

func aNullString(valor string) spanner.NullString {
	return spanner.NullString{
		StringVal: valor,
		Valid:     true,
	}
}
