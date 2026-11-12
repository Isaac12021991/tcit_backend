package domain

// Estados del ciclo de vida de un Post en Cloud Spanner.
const (
	EstadoPendiente  = "PENDIENTE"
	EstadoCompletado = "COMPLETADO"
)

// Post modela el registro almacenado en la tabla Posts de Google Cloud Spanner.
// Esquema:
// CREATE TABLE Posts (
//   Id STRING(36) NOT NULL,
//   Nombre STRING(255) NOT NULL,
//   Descripcion STRING(MAX),
//   Adjunto STRING(MAX),
//   DatosExtraidosIA STRING(MAX),
//   EstadoProcesamiento STRING(50)
// ) PRIMARY KEY (Id);
type Post struct {
	ID                  string `json:"id" spanner:"Id"`
	Nombre              string `json:"nombre" spanner:"Nombre"`
	Descripcion         string `json:"descripcion" spanner:"Descripcion"`
	Adjunto             string `json:"adjunto" spanner:"Adjunto"`
	DatosExtraidosIA    string `json:"datosExtraidosIA" spanner:"DatosExtraidosIA"`
	EstadoProcesamiento string `json:"estadoProcesamiento" spanner:"EstadoProcesamiento"`
}

// CrearPostRequest define el payload JSON (en camelCase) recibido en POST /posts.
// El campo Adjunto puede contener el archivo en base64 o nombre/contenido para que
// posteriormente el orquestador en Symfony lo suba a Google Drive.
type CrearPostRequest struct {
	Nombre      string `json:"nombre"`
	Descripcion string `json:"descripcion"`
	Adjunto     string `json:"adjunto"`
}
