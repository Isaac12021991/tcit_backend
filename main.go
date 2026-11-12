package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/tcit/challenge-full-stack/backend/internal/handler"
	"github.com/tcit/challenge-full-stack/backend/internal/repository"
	"github.com/tcit/challenge-full-stack/backend/internal/service"
)

func main() {
	// Cargar variables desde el archivo .env si existe; si no existe, usar el entorno del sistema operativo
	if err := godotenv.Load(); err != nil {
		log.Println("[BorgRuntime] Archivo .env no encontrado; utilizando variables de entorno del sistema operativo")
	}

	ctx := context.Background()

	puerto := os.Getenv("PORT")
	if puerto == "" {
		puerto = "8080"
	}

	// 1. Inicializar Repositorio de Google Cloud Spanner (consume SPANNER_DSN desde variables de entorno)
	postRepo, err := repository.NuevoSpannerPostRepository(ctx)
	if err != nil {
		log.Fatalf("[BorgRuntime] Error fatal al conectar con Cloud Spanner: %v", err)
	}
	defer postRepo.Cerrar()

	// 2. Inicializar Servicio y Handlers HTTP REST
	postService := service.NuevoPostService(postRepo)
	postHandler := handler.NuevoPostHandler(postService)

	mux := http.NewServeMux()
	postHandler.RegistrarRutas(mux)

	// Endpoint Borg de salud (/healthz)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":              "OK",
			"spannerEmulatorHost": os.Getenv("SPANNER_EMULATOR_HOST"),
			"spannerDsn":          os.Getenv("SPANNER_DSN"),
		})
	})

	servidor := &http.Server{
		Addr:         ":" + puerto,
		Handler:      middlewareCORS(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("[BorgRuntime] Servidor API REST ejecutándose en http://localhost:%s", puerto)
	if errSrv := servidor.ListenAndServe(); errSrv != nil && errSrv != http.ErrServerClosed {
		log.Fatalf("[BorgRuntime] Error fatal en servidor HTTP: %v", errSrv)
	}
}

func middlewareCORS(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		inicio := time.Now()
		siguiente.ServeHTTP(w, r)
		log.Printf("[BorgAccessLog] %s %s (%s)", r.Method, r.URL.Path, time.Since(inicio))
	})
}
