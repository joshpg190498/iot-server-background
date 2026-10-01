package deployment

import (
	"strings"
	"testing"

	"ceiot-tf-background/modules/model-deployment/models"
)

const goodSHA = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"

func mv() models.ModelVersion {
	return models.ModelVersion{ID: 7, ModelName: "fashion-cnn", Version: 3,
		PathTFLite: "fashion-cnn/3/model.tflite", SHA256: strings.ToUpper(goodSHA)}
}

func TestBuildCommand_AbsoluteURL(t *testing.T) {
	cmd, err := BuildCommand(mv(), "http://192.168.1.41:8000/")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.URL != "http://192.168.1.41:8000/artifacts/fashion-cnn/3/model.tflite" {
		t.Fatalf("URL inesperada: %s", cmd.URL)
	}
	if cmd.IDModel != "fashion-cnn" || cmd.IDModelVersion != 7 || cmd.Version != 3 || !cmd.Activate {
		t.Fatalf("campos inesperados: %+v", cmd)
	}
	if cmd.SHA256 != goodSHA {
		t.Fatalf("el sha256 debe ir en minúsculas: %s", cmd.SHA256)
	}
}

func TestBuildCommand_RelativeURL(t *testing.T) {
	cmd, err := BuildCommand(mv(), "")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.URL != "/artifacts/fashion-cnn/3/model.tflite" {
		t.Fatalf("URL relativa inesperada: %s", cmd.URL)
	}
}

func TestBuildCommand_Rejects(t *testing.T) {
	bad := []func(*models.ModelVersion){
		func(m *models.ModelVersion) { m.ModelName = "Fashion CNN" }, // espacio
		func(m *models.ModelVersion) { m.ModelName = "../etc" },
		func(m *models.ModelVersion) { m.SHA256 = "" },
		func(m *models.ModelVersion) { m.Version = 0 },
		func(m *models.ModelVersion) { m.PathTFLite = "" },
		func(m *models.ModelVersion) { m.PathTFLite = "../../etc/passwd" },
	}
	for i, mutate := range bad {
		m := mv()
		mutate(&m)
		if _, err := BuildCommand(m, ""); err == nil {
			t.Errorf("caso %d: esperaba error para %+v", i, m)
		}
	}
}

func TestParseDeployTopic(t *testing.T) {
	if id, ok := ParseDeployTopic("devices/rpi4-01/deploy"); !ok || id != "rpi4-01" {
		t.Fatalf("got %q %v", id, ok)
	}
	for _, tp := range []string{"devices/rpi4-01/config", "devices//deploy", "server/deploy/x", "devices/a/b/deploy"} {
		if _, ok := ParseDeployTopic(tp); ok {
			t.Errorf("%s no debía aceptarse", tp)
		}
	}
}

func TestMapAckStatus(t *testing.T) {
	cases := map[string]string{"activated": DBActive, "downloaded": DBDownloaded, "failed": DBFailed}
	for in, want := range cases {
		if got, ok := MapAckStatus(in); !ok || got != want {
			t.Errorf("%s → %s (%v), esperaba %s", in, got, ok, want)
		}
	}
	if _, ok := MapAckStatus("whatever"); ok {
		t.Error("estado desconocido no debía mapearse")
	}
}

func TestValidateEvent(t *testing.T) {
	devs, err := ValidateEvent(models.DeployEvent{IDModelVersion: 1, Devices: []string{"a", " a ", "", "b"}})
	if err != nil || len(devs) != 2 || devs[0] != "a" || devs[1] != "b" {
		t.Fatalf("got %v %v", devs, err)
	}
	if _, err := ValidateEvent(models.DeployEvent{IDModelVersion: 0, Devices: []string{"a"}}); err == nil {
		t.Error("id_model_version 0 debía fallar")
	}
	if _, err := ValidateEvent(models.DeployEvent{IDModelVersion: 1}); err == nil {
		t.Error("sin dispositivos debía fallar")
	}
}

func TestDeviceTopic(t *testing.T) {
	if got := DeviceTopic("server/deploy/___DEVICE___", "rpi4-01"); got != "server/deploy/rpi4-01" {
		t.Fatal(got)
	}
}
