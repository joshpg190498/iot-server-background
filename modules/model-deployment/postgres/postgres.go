package postgres

import (
	"context"
	"errors"
	"log"
	"time"

	"ceiot-tf-background/modules/model-deployment/deployment"
	"ceiot-tf-background/modules/model-deployment/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var db *pgxpool.Pool

// Todas las marcas de tiempo se guardan en UTC (columnas TIMESTAMP sin zona).
const nowUTC = `(NOW() AT TIME ZONE 'UTC')`

func ConnectDB(connString string) error {
	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		return err
	}
	db = pool
	log.Println("Connected to PostgreSQL")
	return nil
}

func CloseDB() {
	if db != nil {
		db.Close()
		log.Println("PostgreSQL connection closed")
	}
}

// GetModelVersion devuelve nombre, versión, ruta y hash de una MODEL_VERSION.
// found=false si no existe (evento inválido: no se reintenta).
func GetModelVersion(ctx context.Context, idModelVersion int) (mv models.ModelVersion, found bool, err error) {
	query := `
		SELECT mv.ID, m.NAME, mv.VERSION, COALESCE(mv.PATH_TFLITE, ''), COALESCE(mv.SHA256, '')
		FROM MODEL_VERSION mv
		JOIN MODEL m ON m.ID = mv.ID_MODEL
		WHERE mv.ID = $1
	`
	err = db.QueryRow(ctx, query, idModelVersion).
		Scan(&mv.ID, &mv.ModelName, &mv.Version, &mv.PathTFLite, &mv.SHA256)
	if errors.Is(err, pgx.ErrNoRows) {
		return mv, false, nil
	}
	if err != nil {
		return mv, false, err
	}
	return mv, true, nil
}

// CreateDeployments registra (o reinicia) el despliegue de una versión en
// varios dispositivos, en una sola transacción:
//
//   - Descarta dispositivos que no existen en DEVICES (se devuelven en unknown)
//     para que un id mal escrito no rompa la FK y bloquee el consumidor Kafka.
//   - Si el dispositivo tenía OTRA versión del mismo modelo aún 'pending', la
//     marca 'superseded': así un reenvío tardío de la versión vieja nunca
//     pisa a la nueva en la Pi.
//   - Upsert en DEVICE_MODEL con STATUS='pending' y DEPLOYED_AT_UTC=ahora.
//     Redesplegar una versión ya activa también vuelve a 'pending'; la Pi
//     contesta "ya activa" y queda 'active' de nuevo.
func CreateDeployments(ctx context.Context, idModelVersion int, devices []string) (known, unknown []string, err error) {
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT ID_DEVICE FROM DEVICES WHERE ID_DEVICE = ANY($1)`, devices)
	if err != nil {
		return nil, nil, err
	}
	exists := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, nil, err
		}
		exists[id] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	for _, d := range devices {
		if !exists[d] {
			unknown = append(unknown, d)
			continue
		}
		known = append(known, d)

		if _, err := tx.Exec(ctx, `
			UPDATE DEVICE_MODEL dm
			SET STATUS = $3
			FROM MODEL_VERSION mv
			WHERE dm.ID_MODEL_VERSION = mv.ID
			  AND dm.ID_DEVICE = $1
			  AND dm.STATUS = $4
			  AND dm.ID_MODEL_VERSION <> $2
			  AND mv.ID_MODEL = (SELECT ID_MODEL FROM MODEL_VERSION WHERE ID = $2)
		`, d, idModelVersion, deployment.DBSuperseded, deployment.DBPending); err != nil {
			return nil, nil, err
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO DEVICE_MODEL (ID_DEVICE, ID_MODEL_VERSION, STATUS, DEPLOYED_AT_UTC)
			VALUES ($1, $2, $3, `+nowUTC+`)
			ON CONFLICT (ID_DEVICE, ID_MODEL_VERSION) DO UPDATE SET
				STATUS = EXCLUDED.STATUS,
				DEPLOYED_AT_UTC = EXCLUDED.DEPLOYED_AT_UTC,
				ACTIVATED_AT_UTC = NULL
		`, d, idModelVersion, deployment.DBPending); err != nil {
			return nil, nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return known, unknown, nil
}

// ApplyAck registra el ACK del dispositivo. Si la versión quedó activa, las
// demás versiones del mismo modelo que figuraban 'active' en ese dispositivo
// pasan a 'inactive' (el symlink 'current' solo apunta a una).
//
// Un ACK sobre una fila 'superseded' se ignora: ya se pidió otra versión y su
// ACK es el que manda. Devuelve applied=false si no había fila que actualizar.
func ApplyAck(ctx context.Context, ack models.DeployStatus, dbStatus string) (applied bool, err error) {
	ackTime, perr := time.Parse(time.RFC3339, ack.TimestampUTC)
	if perr != nil {
		ackTime = time.Now()
	}
	ackTime = ackTime.UTC()

	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE DEVICE_MODEL
		SET STATUS = $3::varchar,
		    ACTIVATED_AT_UTC = CASE WHEN $3::varchar = $4::varchar THEN $5::timestamp ELSE ACTIVATED_AT_UTC END
		WHERE ID_DEVICE = $1
		  AND ID_MODEL_VERSION = $2
		  AND STATUS <> $6::varchar
	`, ack.IDDevice, ack.IDModelVersion, dbStatus, deployment.DBActive, ackTime, deployment.DBSuperseded)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}

	if dbStatus == deployment.DBActive {
		if _, err := tx.Exec(ctx, `
			UPDATE DEVICE_MODEL dm
			SET STATUS = $3
			FROM MODEL_VERSION mv
			WHERE dm.ID_MODEL_VERSION = mv.ID
			  AND dm.ID_DEVICE = $1
			  AND dm.ID_MODEL_VERSION <> $2
			  AND dm.STATUS = $4
			  AND mv.ID_MODEL = (SELECT ID_MODEL FROM MODEL_VERSION WHERE ID = $2)
		`, ack.IDDevice, ack.IDModelVersion, deployment.DBInactive, deployment.DBActive); err != nil {
			return false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

// GetPendingDeployments devuelve los despliegues 'pending' cuyo último envío
// fue hace más de olderThan (sin ACK: dispositivo apagado, ACK perdido, etc.).
func GetPendingDeployments(ctx context.Context, olderThan time.Duration) ([]models.PendingDeployment, error) {
	rows, err := db.Query(ctx, `
		SELECT dm.ID_DEVICE, mv.ID, m.NAME, mv.VERSION,
		       COALESCE(mv.PATH_TFLITE, ''), COALESCE(mv.SHA256, '')
		FROM DEVICE_MODEL dm
		JOIN MODEL_VERSION mv ON mv.ID = dm.ID_MODEL_VERSION
		JOIN MODEL m ON m.ID = mv.ID_MODEL
		WHERE dm.STATUS = $1
		  AND dm.DEPLOYED_AT_UTC < `+nowUTC+` - make_interval(secs => $2)
		ORDER BY dm.DEPLOYED_AT_UTC
	`, deployment.DBPending, olderThan.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.PendingDeployment
	for rows.Next() {
		var p models.PendingDeployment
		if err := rows.Scan(&p.IDDevice, &p.ID, &p.ModelName, &p.Version, &p.PathTFLite, &p.SHA256); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// TouchDeployment actualiza DEPLOYED_AT_UTC tras un reenvío, para que el
// próximo reintento espere otro intervalo completo.
func TouchDeployment(ctx context.Context, idDevice string, idModelVersion int) error {
	_, err := db.Exec(ctx, `
		UPDATE DEVICE_MODEL SET DEPLOYED_AT_UTC = `+nowUTC+`
		WHERE ID_DEVICE = $1 AND ID_MODEL_VERSION = $2 AND STATUS = $3
	`, idDevice, idModelVersion, deployment.DBPending)
	return err
}

// MarkFailed deja un despliegue en 'failed' cuando el servidor no puede ni
// construir la orden (p. ej. MODEL_VERSION sin SHA256): reintentar no sirve.
func MarkFailed(ctx context.Context, idDevice string, idModelVersion int) error {
	_, err := db.Exec(ctx, `
		UPDATE DEVICE_MODEL SET STATUS = $3
		WHERE ID_DEVICE = $1 AND ID_MODEL_VERSION = $2 AND STATUS = $4
	`, idDevice, idModelVersion, deployment.DBFailed, deployment.DBPending)
	return err
}
