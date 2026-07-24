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
		"commonAnnotations":{"description":"Current position difference exceeds the threshold","notificationType":"alert"},
		"alerts":[
			{
				"status":"firing",
				"labels":{"alertname":"Position mismatch","currency":"ROAM"},
				"annotations":{"summary":"ROAM, position: 58816.2444, valuation: 623.39"},
				"values":{"A":623.39,"B":1}
			},
			{
				"status":"resolved",
				"labels":{"alertname":"Position mismatch","currency":"ES"},
				"annotations":{"summary":"ES, position: -118820.7919, valuation: 248.34"},
				"values":{"A":248.34,"B":1}
			},
			{
				"status":"firing",
				"labels":{"alertname":"Position mismatch","currency":"H"},
				"annotations":{"summary":"H, position: 6839.5352, valuation: 406.92"},
				"values":{"A":406.92,"B":1}
			}
		]
	}`)

	got, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}

	want := grafanaNotification{
		State:            "alerting",
		RuleName:         "Position mismatch",
		NotificationType: grafanaNotificationTypeAlert,
		Message:          "Current position difference exceeds the threshold",
		Matches: []grafanaMatch{
			{Summary: "ROAM, position: 58816.2444, valuation: 623.39"},
			{Summary: "H, position: 6839.5352, valuation: 406.92"},
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
		"commonAnnotations":{"notificationType":"alert"},
		"alerts":[{
			"status":"resolved",
			"labels":{"alertname":"Runner error log count","instance_id":"221"},
			"annotations":{"summary":"实例编号：221；错误日志数：0"},
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

func TestDecodeGrafanaReportType(t *testing.T) {
	body := []byte(`{
		"receiver":"Lark - Operations",
		"status":"firing",
		"commonLabels":{"alertname":"昨日新上线策略表现"},
		"commonAnnotations":{"notificationType":"report"},
		"alerts":[{
			"status":"firing",
			"labels":{"alertname":"昨日新上线策略表现"},
			"annotations":{"summary":"1.1 策略数: 10"},
			"values":{"A":1}
		}]
	}`)

	got, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}
	if got.NotificationType != grafanaNotificationTypeReport {
		t.Fatalf("NotificationType = %q, want report", got.NotificationType)
	}

	card := formatGrafanaAlertForFeishu(got)
	header := card["header"].(map[string]any)
	if header["template"] != "Blue" {
		t.Fatalf("header = %#v", header)
	}
}

func TestDecodeGrafanaAlertRequiresCommonMetadata(t *testing.T) {
	body := []byte(`{
		"receiver":"Lark - Test",
		"status":"firing",
		"title":"Position mismatch",
		"groupLabels":{"alertname":"Position mismatch"},
		"alerts":[{
			"status":"firing",
			"labels":{"alertname":"Position mismatch"},
			"annotations":{"notificationType":"alert","summary":"ROAM: 623.39U"},
			"values":{"A":623.39}
		}]
	}`)

	if _, err := decodeGrafanaAlert(body); err == nil {
		t.Fatal("decodeGrafanaAlert() error = nil, want missing common metadata error")
	}
}

func TestDecodeGrafanaAlertSortsMatches(t *testing.T) {
	body := []byte(`{
		"receiver":"Lark - Test",
		"status":"firing",
		"commonLabels":{"alertname":"Position mismatch"},
		"commonAnnotations":{"notificationType":"alert","notificationSortOrder":"desc","notificationSortAbsolute":"true"},
		"alerts":[
			{
				"status":"firing",
				"labels":{"alertname":"Position mismatch","currency":"ROAM"},
				"annotations":{"summary":"ROAM：200","notificationSortKey":"-200"},
				"values":{"A":200}
			},
			{
				"status":"firing",
				"labels":{"alertname":"Position mismatch","currency":"H"},
				"annotations":{"summary":"H：500","notificationSortKey":"500"},
				"values":{"A":500}
			},
			{
				"status":"firing",
				"labels":{"alertname":"Position mismatch","currency":"ADA"},
				"annotations":{"summary":"ADA：500","notificationSortKey":"-500"},
				"values":{"A":500}
			}
		]
	}`)

	got, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}
	wantSummaries := []string{"ADA：500", "H：500", "ROAM：200"}
	for i, summary := range wantSummaries {
		if got.Matches[i].Summary != summary {
			t.Fatalf("Matches[%d].Summary = %q, want %q", i, got.Matches[i].Summary, summary)
		}
	}
}

func TestDecodeGrafanaAlertSortsTextAscending(t *testing.T) {
	body := []byte(`{
		"receiver":"Lark - Test",
		"status":"firing",
		"commonLabels":{"alertname":"New symbol"},
		"commonAnnotations":{"notificationType":"alert","notificationSortOrder":"asc"},
		"alerts":[
			{"status":"firing","labels":{"alertname":"New symbol"},"annotations":{"summary":"ZETA：1","notificationSortKey":"ZETA"}},
			{"status":"firing","labels":{"alertname":"New symbol"},"annotations":{"summary":"ADA：1"}}
		]
	}`)

	got, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}
	if got.Matches[0].Summary != "ADA：1" || got.Matches[1].Summary != "ZETA：1" {
		t.Fatalf("Matches = %#v", got.Matches)
	}
}

func TestDecodeGrafanaAlertSummary(t *testing.T) {
	body := []byte(`{
		"receiver":"Lark - Test",
		"status":"firing",
		"commonLabels":{"alertname":"现货与合约头寸差异过大"},
		"commonAnnotations":{"notificationType":"alert"},
		"alerts":[
			{
				"status":"firing",
				"labels":{"alertname":"现货与合约头寸差异过大","currency":"ES"},
				"annotations":{"summary":"基础币：ES；头寸：-230911.4119；估值：248.34","notificationSortKey":"248.34"},
				"values":{"R":248.34,"RB":-230911.4119}
			}
		]
	}`)

	got, err := decodeGrafanaAlert(body)
	if err != nil {
		t.Fatalf("decodeGrafanaAlert() error = %v", err)
	}
	want := grafanaNotification{
		State:            "alerting",
		RuleName:         "现货与合约头寸差异过大",
		NotificationType: grafanaNotificationTypeAlert,
		Matches: []grafanaMatch{
			{Summary: "基础币：ES；头寸：-230911.4119；估值：248.34", SortKey: "248.34"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decoded alert = %#v, want %#v", got, want)
	}

	card := formatGrafanaAlertForFeishu(got)
	elements := card["elements"].([]any)
	content := elements[0].(map[string]any)["content"]
	if content != "基础币：ES；头寸：-230911.4119；估值：248.34" {
		t.Fatalf("content = %q", content)
	}
}

func TestDecodeGrafanaAlertRequiresSummary(t *testing.T) {
	body := []byte(`{
		"receiver":"Lark - Test",
		"status":"firing",
		"commonLabels":{"alertname":"现货与合约头寸差异过大"},
		"commonAnnotations":{"notificationType":"alert"},
		"alerts":[{
			"status":"firing",
			"labels":{"alertname":"现货与合约头寸差异过大","currency":"ES"},
			"annotations":{},
			"values":{"R":248.34,"RB":-230911.4119}
		}]
	}`)

	if _, err := decodeGrafanaAlert(body); err == nil {
		t.Fatal("decodeGrafanaAlert() error = nil, want missing summary error")
	}
}

func TestDecodeGrafanaAlertDoesNotInferSummary(t *testing.T) {
	body := []byte(`{
		"receiver":"Lark - Test",
		"status":"firing",
		"commonLabels":{"alertname":"Runner error log count"},
		"commonAnnotations":{"notificationType":"alert"},
		"alerts":[{
			"status":"firing",
			"labels":{"alertname":"Runner error log count","instance_id":"221","service":"runner"},
			"annotations":{},
			"values":{"A":310,"B":1}
		}]
	}`)

	if _, err := decodeGrafanaAlert(body); err == nil {
		t.Fatal("decodeGrafanaAlert() error = nil, want missing summary error")
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
			"annotations":{"Error":"database connection failed"},
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

func TestDecodeGrafanaAlertRejectsMissingSummary(t *testing.T) {
	body := []byte(`{
		"receiver":"Lark - Test",
		"status":"firing",
		"commonLabels":{"alertname":"Query warning"},
		"commonAnnotations":{"notificationType":"alert"},
		"alerts":[{
			"status":"firing",
			"labels":{"alertname":"Query warning","rulename":"Position mismatch"},
			"annotations":{},
			"values":{}
		}]
	}`)

	if _, err := decodeGrafanaAlert(body); err == nil {
		t.Fatal("decodeGrafanaAlert() error = nil, want missing summary error")
	}
}

func TestFormatGrafanaAlertForFeishuPreservesCard(t *testing.T) {
	alert := grafanaNotification{
		State:            "alerting",
		RuleName:         "Position mismatch",
		NotificationType: grafanaNotificationTypeAlert,
		Message:          "Current position difference exceeds the threshold",
		Matches: []grafanaMatch{
			{Summary: "ROAM, position: 58816.2444, valuation: 623.39"},
			{Summary: "H, position: 6839.5352, valuation: 406.92"},
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

func TestFormatGrafanaResolvedAlertForFeishuUsesGreen(t *testing.T) {
	alert := grafanaNotification{
		State:    "ok",
		RuleName: "Position mismatch",
	}

	card := formatGrafanaAlertForFeishu(alert)
	header := card["header"].(map[string]any)
	if header["template"] != "Green" {
		t.Fatalf("header = %#v", header)
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
