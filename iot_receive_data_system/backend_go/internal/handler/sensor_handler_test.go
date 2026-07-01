package handler

import (
	"encoding/json"
	"testing"
)

func TestIPBufferStoreKeepsNormalFragmentRemainder(t *testing.T) {
	store := newIPBufferStore()

	first := `{"sensor_number":"201_03","sensorNo":"201_03","devicetype":"esp32","heart_rate":`
	combined, err := store.appendChunk("10.89.1.3", first, 10240)
	if err != nil {
		t.Fatalf("append first chunk: %v", err)
	}
	objects, remainder := extractJSONObjects(combined)
	if len(objects) != 0 {
		t.Fatalf("expected no complete objects, got %d", len(objects))
	}
	store.setRemainder("10.89.1.3", remainder)

	second := `94,"breath_rate":20,"temperature":[40,32.75,34]}`
	combined, err = store.appendChunk("10.89.1.3", second, 10240)
	if err != nil {
		t.Fatalf("append second chunk: %v", err)
	}
	objects, remainder = extractJSONObjects(combined)
	if remainder != "" {
		t.Fatalf("expected empty remainder, got %q", remainder)
	}
	if len(objects) != 1 {
		t.Fatalf("expected one complete object, got %d", len(objects))
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(objects[0]), &payload); err != nil {
		t.Fatalf("merged object is invalid JSON: %v", err)
	}
	if payload["sensor_number"] != "201_03" {
		t.Fatalf("unexpected sensor_number: %v", payload["sensor_number"])
	}
}

func TestIPBufferStoreDropsPoisonedRemainderWhenFreshObjectStarts(t *testing.T) {
	store := newIPBufferStore()
	store.setRemainder("10.89.1.3", `{"sensor_number":"201_03","broken": `)

	fresh := `{"sensor_number":"201_03","heart_rate":94,"breath_rate":20,"temperature":[36.5]}`
	combined, err := store.appendChunk("10.89.1.3", fresh, 10240)
	if err != nil {
		t.Fatalf("append fresh chunk: %v", err)
	}

	objects, remainder := extractJSONObjects(combined)
	if remainder != "" {
		t.Fatalf("expected empty remainder, got %q", remainder)
	}
	if len(objects) != 1 {
		t.Fatalf("expected one fresh object, got %d", len(objects))
	}
	if objects[0] != fresh {
		t.Fatalf("expected poisoned prefix to be dropped, got %s", objects[0])
	}
}

func TestExtractJSONObjectsRecoversFromCorruptedConcatenation(t *testing.T) {
	input := `{"heart_rate":82,"temperature":[22.5,2{"heart_rate":83,"breath_rate":20,"temperature":[36.5],"sensor_number":"201_03"}`

	objects, remainder := extractJSONObjects(input)
	if remainder != "" {
		t.Fatalf("expected empty remainder, got %q", remainder)
	}
	if len(objects) != 1 {
		t.Fatalf("expected recovered object, got %d", len(objects))
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(objects[0]), &payload); err != nil {
		t.Fatalf("recovered object is invalid JSON: %v", err)
	}
	if payload["sensor_number"] != "201_03" {
		t.Fatalf("unexpected sensor_number: %v", payload["sensor_number"])
	}
}
