package handler

import (
	"reflect"
	"testing"

	"notify/internal/service"
)

func TestDecodeGrafanaAlertLegacy(t *testing.T) {
	body := []byte(`{
		"state":"alerting",
		"ruleName":"Position mismatch",
		"message":"Current position difference exceeds the threshold",
		"evalMatches":[
			{"metric":"ROAM, position: 58816.2444, valuation","value":623.39},
			{"metric":"H, position: 6839.5352, valuation","value":406.92}
		]
	}`)

	got, payloadFormat, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}
	if payloadFormat != "legacy" {
		t.Fatalf("payload format = %q, want legacy", payloadFormat)
	}

	want := service.GrafanaAlert{
		State:    "alerting",
		RuleName: "Position mismatch",
		Message:  "Current position difference exceeds the threshold",
		EvalMatches: []service.EvalMatch{
			{Metric: "ROAM, position: 58816.2444, valuation", Value: 623.39},
			{Metric: "H, position: 6839.5352, valuation", Value: 406.92},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decoded alert = %#v, want %#v", got, want)
	}
}

func TestDecodeGrafanaAlertUnifiedFiring(t *testing.T) {
	body := []byte(`{
		"receiver":"Lark - Test",
		"status":"firing",
		"groupLabels":{"alertname":"Position mismatch"},
		"commonLabels":{"alertname":"Position mismatch","grafana_folder":"Arbitrage"},
		"commonAnnotations":{"lark_message":"Current position difference exceeds the threshold"},
		"alerts":[
			{
				"status":"firing",
				"labels":{"alertname":"Position mismatch","currency":"ROAM"},
				"annotations":{"lark_metric":"ROAM, position: 58816.2444, valuation","lark_value":"623.39"},
				"values":{"A":623.39,"B":1}
			},
			{
				"status":"resolved",
				"labels":{"alertname":"Position mismatch","currency":"ES"},
				"annotations":{"lark_metric":"ES, position: -118820.7919, valuation","lark_value":"248.34"},
				"values":{"A":248.34,"B":1}
			},
			{
				"status":"firing",
				"labels":{"alertname":"Position mismatch","currency":"H"},
				"annotations":{"lark_metric":"H, position: 6839.5352, valuation","lark_value":"406.92"},
				"values":{"A":406.92,"B":1}
			}
		]
	}`)

	got, payloadFormat, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}
	if payloadFormat != "unified" {
		t.Fatalf("payload format = %q, want unified", payloadFormat)
	}

	want := service.GrafanaAlert{
		State:    "alerting",
		RuleName: "Position mismatch",
		Message:  "Current position difference exceeds the threshold",
		EvalMatches: []service.EvalMatch{
			{Metric: "ROAM, position: 58816.2444, valuation", Value: 623.39},
			{Metric: "H, position: 6839.5352, valuation", Value: 406.92},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decoded alert = %#v, want %#v", got, want)
	}
}

func TestDecodeGrafanaAlertUnifiedResolved(t *testing.T) {
	body := []byte(`{
		"receiver":"Lark - Test",
		"status":"resolved",
		"commonLabels":{"alertname":"Runner error log count"},
		"alerts":[{
			"status":"resolved",
			"labels":{"alertname":"Runner error log count","instance_id":"221"},
			"values":{"A":0}
		}]
	}`)

	got, payloadFormat, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}
	if payloadFormat != "unified" {
		t.Fatalf("payload format = %q, want unified", payloadFormat)
	}
	if got.State != "ok" || got.RuleName != "Runner error log count" || len(got.EvalMatches) != 0 {
		t.Fatalf("decoded alert = %#v", got)
	}
}

func TestDecodeGrafanaAlertUnifiedSortsMatches(t *testing.T) {
	body := []byte(`{
		"receiver":"Lark - Test",
		"status":"firing",
		"commonLabels":{"alertname":"Position mismatch"},
		"commonAnnotations":{"notify_sort_order":"desc","notify_sort_abs":"true"},
		"alerts":[
			{
				"status":"firing",
				"labels":{"alertname":"Position mismatch","currency":"ROAM"},
				"annotations":{"lark_metric":"ROAM","lark_value":"200","notify_sort_key":"-200"},
				"values":{"A":200}
			},
			{
				"status":"firing",
				"labels":{"alertname":"Position mismatch","currency":"H"},
				"annotations":{"lark_metric":"H","lark_value":"500","notify_sort_key":"500"},
				"values":{"A":500}
			},
			{
				"status":"firing",
				"labels":{"alertname":"Position mismatch","currency":"ADA"},
				"annotations":{"lark_metric":"ADA","lark_value":"500","notify_sort_key":"-500"},
				"values":{"A":500}
			}
		]
	}`)

	got, _, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}
	wantMetrics := []string{"ADA", "H", "ROAM"}
	for i, metric := range wantMetrics {
		if got.EvalMatches[i].Metric != metric {
			t.Fatalf("EvalMatches[%d].Metric = %q, want %q", i, got.EvalMatches[i].Metric, metric)
		}
	}
}

func TestDecodeGrafanaAlertUnifiedSortsTextAscending(t *testing.T) {
	body := []byte(`{
		"receiver":"Lark - Test",
		"status":"firing",
		"commonLabels":{"alertname":"New symbol"},
		"commonAnnotations":{"notify_sort_order":"asc"},
		"alerts":[
			{"status":"firing","labels":{"alertname":"New symbol"},"annotations":{"lark_metric":"ZETA","lark_value":"1","notify_sort_key":"ZETA"}},
			{"status":"firing","labels":{"alertname":"New symbol"},"annotations":{"lark_metric":"ADA","lark_value":"1","notify_sort_key":"ADA"}}
		]
	}`)

	got, _, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}
	if got.EvalMatches[0].Metric != "ADA" || got.EvalMatches[1].Metric != "ZETA" {
		t.Fatalf("EvalMatches = %#v", got.EvalMatches)
	}
}

func TestDecodeGrafanaAlertUnifiedFallbackFields(t *testing.T) {
	body := []byte(`{
		"receiver":"Lark - Test",
		"status":"firing",
		"alerts":[{
			"status":"firing",
			"labels":{"alertname":"Runner error log count","instance_id":"221","service":"runner"},
			"annotations":{},
			"values":{"A":310,"B":1}
		}]
	}`)

	got, _, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}
	want := service.GrafanaAlert{
		State:    "alerting",
		RuleName: "Runner error log count",
		EvalMatches: []service.EvalMatch{
			{Metric: "instance_id=221, service=runner", Value: 310},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decoded alert = %#v, want %#v", got, want)
	}
}

func TestDecodeGrafanaAlertDatasourceError(t *testing.T) {
	body := []byte(`{
		"receiver":"Lark - Test",
		"status":"firing",
		"commonLabels":{"alertname":"DatasourceError","datasource_uid":"al4sYqNVz"},
		"alerts":[{
			"status":"firing",
			"labels":{"alertname":"DatasourceError","datasource_uid":"al4sYqNVz","rulename":"Position mismatch"},
			"annotations":{"Error":"database connection failed","lark_metric":"[no value]","lark_value":"[no value]"},
			"values":{}
		}]
	}`)

	got, _, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}
	if got.RuleName != "DatasourceError" || got.Message != "Position mismatch: database connection failed" {
		t.Fatalf("decoded alert = %#v", got)
	}
	if len(got.EvalMatches) != 0 {
		t.Fatalf("decoded alert = %#v", got)
	}
}

func TestDecodeGrafanaAlertSkipsMissingValue(t *testing.T) {
	body := []byte(`{
		"receiver":"Lark - Test",
		"status":"firing",
		"commonLabels":{"alertname":"Query warning"},
		"alerts":[{
			"status":"firing",
			"labels":{"alertname":"Query warning","rulename":"Position mismatch"},
			"annotations":{"lark_metric":"[no value]","lark_value":"[no value]"},
			"values":{}
		}]
	}`)

	got, _, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}
	if len(got.EvalMatches) != 0 {
		t.Fatalf("decoded alert = %#v", got)
	}
}

func TestFormatGrafanaAlertForFeishuPreservesLegacyCard(t *testing.T) {
	alert := service.GrafanaAlert{
		State:    "alerting",
		RuleName: "Position mismatch",
		Message:  "Current position difference exceeds the threshold",
		EvalMatches: []service.EvalMatch{
			{Metric: "ROAM, position: 58816.2444, valuation", Value: 623.39},
			{Metric: "H, position: 6839.5352, valuation", Value: 406.92},
		},
	}

	card := formatGrafanaAlertForFeishu(alert)
	header := card["header"].(map[string]any)
	title := header["title"].(map[string]any)
	if title["content"] != "Position mismatch" || header["template"] != "Orange" {
		t.Fatalf("header = %#v", header)
	}

	elements := card["elements"].([]any)
	content := elements[0].(map[string]any)["content"]
	wantContent := "ROAM, position: 58816.2444, valuation: 623.39\nH, position: 6839.5352, valuation: 406.92"
	if content != wantContent {
		t.Fatalf("content = %q, want %q", content, wantContent)
	}
}

func TestDecodeGrafanaAlertRejectsUnsupportedPayload(t *testing.T) {
	if _, _, err := decodeGrafanaAlert([]byte(`{"foo":"bar"}`)); err == nil {
		t.Fatal("decodeGrafanaAlert() error = nil, want unsupported payload error")
	}
}
