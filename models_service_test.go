package snipeit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// The models API returns the fieldset as a nested object ("fieldset": {...})
// or null, never as a fieldset_id — reads must expose it via Model.Fieldset.
func TestModelsListParsesFieldset(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/api/v1/models", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)
		fmt.Fprint(w, `{
			"total": 2,
			"rows": [
				{"id": 56, "name": "iPhone 15 Plus", "model_number": "iPhone15,5", "fieldset": {"id": 1, "name": "Asset with MAC Address"}},
				{"id": 78, "name": "iPhone 17 Pro Max", "model_number": "iPhone18,2", "fieldset": null}
			]
		}`)
	})

	models, _, err := client.Models.List(nil)
	if err != nil {
		t.Fatalf("Models.List returned error: %v", err)
	}
	if len(models.Rows) != 2 {
		t.Fatalf("Models.List returned %d rows, expected 2", len(models.Rows))
	}

	with := models.Rows[0]
	if with.Fieldset == nil {
		t.Fatal("Models.List row 0 Fieldset = nil, expected parsed fieldset")
	}
	if with.Fieldset.ID != 1 || with.Fieldset.Name != "Asset with MAC Address" {
		t.Errorf("Models.List row 0 Fieldset = %+v, expected id 1 name %q", with.Fieldset, "Asset with MAC Address")
	}

	if without := models.Rows[1]; without.Fieldset != nil {
		t.Errorf("Models.List row 1 Fieldset = %+v, expected nil for null fieldset", without.Fieldset)
	}
}

// Patch must use the PATCH verb and send only the fields set on the model —
// Snipe-IT applies partial updates then, which matters because PUT-style
// full updates re-trigger validations on fields the caller didn't touch.
func TestModelsPatch(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	var body map[string]interface{}
	mux.HandleFunc("/api/v1/models/78", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodPatch)
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		fmt.Fprint(w, `{"status": "success", "payload": {"id": 78, "name": "iPhone 17 Pro Max"}}`)
	})

	resp, _, err := client.Models.Patch(78, Model{FieldsetID: 1})
	if err != nil {
		t.Fatalf("Models.Patch returned error: %v", err)
	}
	if string(resp.Status) != "success" {
		t.Errorf("Models.Patch returned Status = %s, expected success", resp.Status)
	}

	if got := body["fieldset_id"]; got != float64(1) {
		t.Errorf("Patch body fieldset_id = %v, expected 1", got)
	}
	if _, ok := body["name"]; ok {
		t.Error("Patch body contains name, expected only explicitly-set fields")
	}
}
