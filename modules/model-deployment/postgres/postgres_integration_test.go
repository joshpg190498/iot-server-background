//go:build integration

// Test de integración de la capa SQL contra un PostgreSQL real con el esquema
// 001_init.sql + 002_fl_schema.sql. No corre con `go test ./...` normal.
//
//	PG_TEST_URL="postgres://user:pass@host:5432/db" \
//	  go test -tags integration -v ./modules/model-deployment/postgres/
//
// Usa una BD de PRUEBA: crea y borra filas en DEVICES/MODEL/DEVICE_MODEL.
package postgres

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"ceiot-tf-background/modules/model-deployment/deployment"
	"ceiot-tf-background/modules/model-deployment/models"
)

const sha = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"

var ctx = context.Background()

func setup(t *testing.T) (v1, v2 int) {
	t.Helper()
	url := os.Getenv("PG_TEST_URL")
	if url == "" {
		t.Skip("PG_TEST_URL no definido")
	}
	if db == nil {
		if err := ConnectDB(url); err != nil {
			t.Fatal(err)
		}
	}
	cleanup := func() {
		db.Exec(ctx, `DELETE FROM MODEL WHERE NAME = 'itest-model'`)
		db.Exec(ctx, `DELETE FROM DEVICES WHERE ID_DEVICE IN ('itest-a','itest-b')`)
	}
	cleanup()
	t.Cleanup(cleanup)

	mustExec(t, `INSERT INTO DEVICES (ID_DEVICE, DESCRIPTION) VALUES ('itest-a','a'),('itest-b','b')`)
	var idModel int
	if err := db.QueryRow(ctx, `INSERT INTO MODEL (NAME) VALUES ('itest-model') RETURNING ID`).Scan(&idModel); err != nil {
		t.Fatal(err)
	}
	for i, dst := range []*int{&v1, &v2} {
		if err := db.QueryRow(ctx, `
			INSERT INTO MODEL_VERSION (ID_MODEL, VERSION, PATH_TFLITE, SHA256)
			VALUES ($1, $2, $3, $4) RETURNING ID`,
			idModel, i+1, "itest-model/"+string(rune('1'+i))+"/model.tflite", sha).Scan(dst); err != nil {
			t.Fatal(err)
		}
	}
	return v1, v2
}

func mustExec(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, err := db.Exec(ctx, q, args...); err != nil {
		t.Fatalf("%v\n%s", err, q)
	}
}

func status(t *testing.T, dev string, mv int) string {
	t.Helper()
	var s string
	if err := db.QueryRow(ctx, `SELECT STATUS FROM DEVICE_MODEL WHERE ID_DEVICE=$1 AND ID_MODEL_VERSION=$2`, dev, mv).Scan(&s); err != nil {
		return "<none>"
	}
	return s
}

func ack(dev string, mv int, st string) models.DeployStatus {
	return models.DeployStatus{IDDevice: dev, IDModelVersion: mv, Status: st,
		TimestampUTC: time.Now().UTC().Format(time.RFC3339)}
}

func TestDeploymentLifecycle(t *testing.T) {
	v1, v2 := setup(t)

	// GetModelVersion
	mv, found, err := GetModelVersion(ctx, v1)
	if err != nil || !found || mv.ModelName != "itest-model" || mv.Version != 1 || mv.SHA256 != sha {
		t.Fatalf("GetModelVersion: %+v found=%v err=%v", mv, found, err)
	}
	if _, found, _ := GetModelVersion(ctx, -1); found {
		t.Fatal("MODEL_VERSION inexistente no debía encontrarse")
	}

	// 1) Deploy v1 a un dispositivo real y a uno inexistente
	known, unknown, err := CreateDeployments(ctx, v1, []string{"itest-a", "ghost"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(known, ",") != "itest-a" || strings.Join(unknown, ",") != "ghost" {
		t.Fatalf("known=%v unknown=%v", known, unknown)
	}
	if s := status(t, "itest-a", v1); s != deployment.DBPending {
		t.Fatalf("v1 debía estar pending, está %s", s)
	}

	// 2) Sin ACK: aparece como pendiente a reenviar; tras Touch ya no (aún)
	pend, err := GetPendingDeployments(ctx, 0)
	if err != nil || !containsPending(pend, "itest-a", v1) {
		t.Fatalf("esperaba v1 pendiente: %+v err=%v", pend, err)
	}
	if err := TouchDeployment(ctx, "itest-a", v1); err != nil {
		t.Fatal(err)
	}
	pend, _ = GetPendingDeployments(ctx, time.Hour)
	if containsPending(pend, "itest-a", v1) {
		t.Fatal("tras Touch no debía figurar como pendiente > 1h")
	}

	// 3) ACK activated → active con ACTIVATED_AT_UTC
	if ok, err := ApplyAck(ctx, ack("itest-a", v1, "activated"), deployment.DBActive); err != nil || !ok {
		t.Fatalf("ApplyAck v1: ok=%v err=%v", ok, err)
	}
	var activated *time.Time
	db.QueryRow(ctx, `SELECT ACTIVATED_AT_UTC FROM DEVICE_MODEL WHERE ID_DEVICE='itest-a' AND ID_MODEL_VERSION=$1`, v1).Scan(&activated)
	if status(t, "itest-a", v1) != deployment.DBActive || activated == nil {
		t.Fatal("v1 debía quedar active con ACTIVATED_AT_UTC")
	}

	// 4) Deploy v2 y, antes del ACK, se pide de nuevo v1 (rollback):
	//    v2 pasa a superseded y un ACK tardío de v2 se ignora.
	if _, _, err := CreateDeployments(ctx, v2, []string{"itest-a"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CreateDeployments(ctx, v1, []string{"itest-a"}); err != nil {
		t.Fatal(err)
	}
	if s := status(t, "itest-a", v2); s != deployment.DBSuperseded {
		t.Fatalf("v2 debía quedar superseded, está %s", s)
	}
	if ok, _ := ApplyAck(ctx, ack("itest-a", v2, "activated"), deployment.DBActive); ok {
		t.Fatal("ACK de una versión superseded no debía aplicarse")
	}
	if ok, _ := ApplyAck(ctx, ack("itest-a", v1, "activated"), deployment.DBActive); !ok {
		t.Fatal("ACK de v1 debía aplicarse")
	}

	// 5) Deploy v2 de verdad → activa v2 y v1 pasa a inactive
	if _, _, err := CreateDeployments(ctx, v2, []string{"itest-a"}); err != nil {
		t.Fatal(err)
	}
	if ok, _ := ApplyAck(ctx, ack("itest-a", v2, "activated"), deployment.DBActive); !ok {
		t.Fatal("ACK de v2 debía aplicarse")
	}
	if a, b := status(t, "itest-a", v1), status(t, "itest-a", v2); a != deployment.DBInactive || b != deployment.DBActive {
		t.Fatalf("esperaba v1=inactive v2=active, got v1=%s v2=%s", a, b)
	}

	// 6) Un fallo en otro dispositivo no afecta al primero
	if _, _, err := CreateDeployments(ctx, v2, []string{"itest-b"}); err != nil {
		t.Fatal(err)
	}
	if ok, _ := ApplyAck(ctx, ack("itest-b", v2, "failed"), deployment.DBFailed); !ok {
		t.Fatal("ACK failed debía aplicarse")
	}
	if status(t, "itest-b", v2) != deployment.DBFailed || status(t, "itest-a", v2) != deployment.DBActive {
		t.Fatal("estados cruzados entre dispositivos")
	}

	// 7) ACK sin despliegue registrado
	if ok, _ := ApplyAck(ctx, ack("itest-b", v1, "activated"), deployment.DBActive); ok {
		t.Fatal("ACK sin fila en DEVICE_MODEL no debía aplicarse")
	}

	// 8) MarkFailed solo afecta a pendientes
	if _, _, err := CreateDeployments(ctx, v1, []string{"itest-b"}); err != nil {
		t.Fatal(err)
	}
	if err := MarkFailed(ctx, "itest-b", v1); err != nil || status(t, "itest-b", v1) != deployment.DBFailed {
		t.Fatalf("MarkFailed: %v %s", err, status(t, "itest-b", v1))
	}
}

func containsPending(ps []models.PendingDeployment, dev string, mv int) bool {
	for _, p := range ps {
		if p.IDDevice == dev && p.ID == mv {
			return true
		}
	}
	return false
}
