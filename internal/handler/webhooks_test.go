package handler

import (
	"reflect"
	"testing"
)

func TestDecodeGrafanaAlertFiring(t *testing.T) {
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

	got, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}

	want := grafanaNotification{
		State:    "alerting",
		RuleName: "Position mismatch",
		Message:  "Current position difference exceeds the threshold",
		Matches: []grafanaMatch{
			{Metric: "ROAM, position: 58816.2444, valuation", Value: 623.39},
			{Metric: "H, position: 6839.5352, valuation", Value: 406.92},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decoded alert = %#v, want %#v", got, want)
	}
}

func TestDecodeGrafanaAlertResolved(t *testing.T) {
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

	got, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}
	if got.State != "ok" || got.RuleName != "Runner error log count" || len(got.Matches) != 0 {
		t.Fatalf("decoded alert = %#v", got)
	}
}

func TestDecodeGrafanaAlertSortsMatches(t *testing.T) {
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

	got, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}
	wantMetrics := []string{"ADA", "H", "ROAM"}
	for i, metric := range wantMetrics {
		if got.Matches[i].Metric != metric {
			t.Fatalf("Matches[%d].Metric = %q, want %q", i, got.Matches[i].Metric, metric)
		}
	}
}

func TestDecodeGrafanaAlertSortsTextAscending(t *testing.T) {
	body := []byte(`{
		"receiver":"Lark - Test",
		"status":"firing",
		"commonLabels":{"alertname":"New symbol"},
		"commonAnnotations":{"notify_sort_order":"asc"},
		"alerts":[
			{"status":"firing","labels":{"alertname":"New symbol"},"annotations":{"lark_metric":"ZETA","lark_value":"1","notify_sort_key":"ZETA"}},
			{"status":"firing","labels":{"alertname":"New symbol"},"annotations":{"lark_metric":"ADA","lark_value":"1"}}
		]
	}`)

	got, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}
	if got.Matches[0].Metric != "ADA" || got.Matches[1].Metric != "ZETA" {
		t.Fatalf("Matches = %#v", got.Matches)
	}
}

func TestDecodeGrafanaAlertFallbackFields(t *testing.T) {
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

	got, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}
	want := grafanaNotification{
		State:    "alerting",
		RuleName: "Runner error log count",
		Matches: []grafanaMatch{
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

	got, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}
	if got.RuleName != "DatasourceError" || got.Message != "Position mismatch: database connection failed" {
		t.Fatalf("decoded alert = %#v", got)
	}
	if len(got.Matches) != 0 {
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

	got, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}
	if len(got.Matches) != 0 {
		t.Fatalf("decoded alert = %#v", got)
	}
}

func TestFormatGrafanaAlertForFeishuPreservesCard(t *testing.T) {
	alert := grafanaNotification{
		State:    "alerting",
		RuleName: "Position mismatch",
		Message:  "Current position difference exceeds the threshold",
		Matches: []grafanaMatch{
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
	if _, err := decodeGrafanaAlert([]byte(`{"foo":"bar"}`)); err == nil {
		t.Fatal("decodeGrafanaAlert() error = nil, want unsupported payload error")
	}
}

func TestDecodeGrafanaAlertRejectsLegacyPayload(t *testing.T) {
	body := []byte(`{
		"state":"alerting",
		"ruleName":"Position mismatch",
		"evalMatches":[{"metric":"ROAM","value":623.39}]
	}`)
	if _, err := decodeGrafanaAlert(body); err == nil {
		t.Fatal("decodeGrafanaAlert() error = nil, want legacy payload error")
	}
}
