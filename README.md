# Backend REST API — Campaigns & Briefs Service (`/backend`)

![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white)
![Cloud Spanner](https://img.shields.io/badge/Google_Cloud_Spanner-Emulator-4285F4?logo=googlecloud&logoColor=white)
![Architecture](https://img.shields.io/badge/Architecture-Borg_Modular-0D47A1)

Microservicio de **API REST** de baja latencia desarrollado en **Golang**, responsable de gestionar el ciclo de vida de los **Posts (Campañas y Briefs)** e interactuar directamente con **Google Cloud Spanner**.

Siguiendo una arquitectura modular inspirada en **Google Borg**, este servicio se enfoca exclusivamente en operaciones síncronas y veloces contra la base de datos distribuida, delegando el procesamiento pesado de Inteligencia Artificial y archivos adjuntos al orquestador externo en Symfony mediante el estado `PENDIENTE`.

---

## Tech Stack

- **Lenguaje:** Go `1.23+`
- **Base de Datos:** SDK oficial de Google Cloud Spanner (`cloud.google.com/go/spanner`)
- **Identificadores Distribuidos:** `github.com/google/uuid` (generación de llaves primarias `UUIDv4` anti-*hotspotting*)
- **Configuración por Entorno:** `github.com/joho/godotenv`
- **Enrutamiento HTTP:** `net/http` (`http.ServeMux` nativo de Go 1.22+)

---

## Estructura del Proyecto

```text
backend/
├── .env                  # Variables de entorno locales (no versionado)
├── .env.example          # Plantilla de variables de entorno
├── .gitignore            # Reglas de exclusión de Git para Go
├── go.mod                # Definición del módulo y dependencias
├── main.go               # Punto de entrada, carga de .env y servidor HTTP
├── schema.sql            # Referencia del esquema DDL en Cloud Spanner
└── internal/
    ├── domain/
    │   └── post.go               # Entidad Post, estados y DTOs en camelCase
    ├── repository/
    │   └── spanner_repository.go # Cliente Cloud Spanner y mapeo con spanner.NullString
    ├── service/
    │   └── post_service.go       # Reglas de negocio y generación de UUIDv4
    └── handler/
        └── post_handler.go       # Controladores HTTP REST y serialización JSON
```

---

## Endpoints Disponibles

| Método | Ruta | Descripción | Código de Éxito |
| :--- | :--- | :--- | :--- |
| `POST` | `/posts` | Crea un nuevo Post con `UUIDv4` y `EstadoProcesamiento = 'PENDIENTE'`. | `201 Created` |
| `GET` | `/posts` | Retorna el listado completo de Posts almacenados en Cloud Spanner. | `200 OK` |
| `GET` | `/posts/pending` | Retorna únicamente los Posts con `EstadoProcesamiento = 'PENDIENTE'` (consumido por Symfony). | `200 OK` |
| `DELETE` | `/posts/{id}` | Elimina un Post dentro de una transacción `ReadWriteTransaction` y retorna el objeto eliminado. | `200 OK` |
| `GET` | `/healthz` | Endpoint de observabilidad y estado de conexión con el emulador de Cloud Spanner. | `200 OK` |

---

## Requisitos Previos y Variables de Entorno

1. **Go `1.23` o superior** instalado en el sistema.
2. **Emulador de Google Cloud Spanner** corriendo en Docker sobre el puerto `9010` (ver [`/database/README.md`](../database/README.md)).
3. Archivo `.env` en la raíz de `/backend` (puedes copiarlo desde `.env.example`):

```dotenv
PORT=8080
SPANNER_EMULATOR_HOST=localhost:9010
SPANNER_DSN=projects/mi-proyecto-local/instances/mi-instancia-local/databases/posts_db
```

---

## Instrucciones Paso a Paso para Levantar Localmente

Abre una terminal en **PowerShell** y ejecuta:

```powershell
# 1. Posicionarse en la carpeta del backend
cd backend

# 2. (Opcional) Crear el archivo .env a partir de la plantilla si no existe
if (!(Test-Path .env)) { Copy-Item .env.example .env }

# 3. Descargar y verificar dependencias del módulo
go mod tidy

# 4. Ejecutar el servidor en http://localhost:8080
go run .
```

### Ejemplo de Prueba Rápida (`curl` / PowerShell)

```powershell
# Verificar estado de salud del servicio
Invoke-RestMethod -Uri "http://localhost:8080/healthz"

# Consultar posts pendientes de procesamiento
Invoke-RestMethod -Uri "http://localhost:8080/posts/pending"
```

