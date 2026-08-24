package logalert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"logalert/internal/config"
	"logalert/internal/handler"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	cfg := config.DefaultConfig()
	log := logger.NewLogger(logger.LogLevelDebug, logger.NewDiscardWriter())

	logStore := store.NewMemoryLogStore(10000, log)
	alertStore := store.NewMemoryAlertStore(10000, log)

	logService := service.NewLogService(logStore, cfg, log)
	alertService := service.NewAlertService(alertStore, cfg, log)

	logHandler := handler.NewLogHandler(logService, log)
	alertHandler := handler.NewAlertHandler(alertService, log)

	ctx := context.Background()

	t.Run("LogLevelConsistency", func(t *testing.T) {
		createBody := `{"level":"INFO","source":"test-service","message":"test message for level consistency","service":"test-service"}`
		req := httptest.NewRequest(http.MethodPost, "/api/logs", bytes.NewBufferString(createBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		logHandler.CreateLog(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var createResp struct {
			Code int `json:"code"`
			Data struct {
				Level model.LogLevel `json:"level"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
			t.Fatalf("Failed to parse create response: %v", err)
		}

		if createResp.Data.Level != model.LevelInfo {
			t.Errorf("Level should be %s, got %s", model.LevelInfo, createResp.Data.Level)
		}

		queryReq := httptest.NewRequest(http.MethodGet, "/api/logs?levels=INFO&limit=100", nil)
		qw := httptest.NewRecorder()
		logHandler.QueryLogs(qw, queryReq)

		if qw.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", qw.Code)
		}

		var queryResp struct {
			Code int `json:"code"`
			Data struct {
				Total int64 `json:"total"`
				Items []struct {
					Level model.LogLevel `json:"level"`
				} `json:"items"`
			} `json:"data"`
		}
		if err := json.Unmarshal(qw.Body.Bytes(), &queryResp); err != nil {
			t.Fatalf("Failed to parse query response: %v", err)
		}

		if queryResp.Data.Total != 1 {
			t.Errorf("Expected 1 result, got %d", queryResp.Data.Total)
		}

		if len(queryResp.Data.Items) != 1 {
			t.Errorf("Expected 1 item, got %d", len(queryResp.Data.Items))
		}

		if queryResp.Data.Total == 1 && len(queryResp.Data.Items) == 1 {
			fmt.Println("LogLevelConsistency: GREEN - level stored and queried with consistent case")
		} else {
			fmt.Println("LogLevelConsistency: RED - level case mismatch causes query failure")
		}
	})

	t.Run("AlertStatusConsistency", func(t *testing.T) {
		rule := model.NewAlertRule("test-rule", model.RuleCondition{
			Type:    model.ConditionLevel,
			Level:   model.LevelError,
			Source:  "test-service",
			Service: "test-service",
		})

		alert := model.NewAlertEvent(rule, "test alert message", "test-service")
		alert.Status = model.AlertOpen
		alert.Severity = model.SeverityMedium

		err := alertService.RecordAlert(ctx, alert)
		if err != nil {
			t.Fatalf("RecordAlert failed: %v", err)
		}

		time.Sleep(10 * time.Millisecond)

		queryReq := httptest.NewRequest(http.MethodGet, "/api/alerts?statuses=open&severities=medium&limit=50", nil)
		qw := httptest.NewRecorder()
		alertHandler.QueryAlerts(qw, queryReq)

		if qw.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", qw.Code)
		}

		var queryResp struct {
			Code int `json:"code"`
			Data struct {
				Total int64 `json:"total"`
				Items []struct {
					Status model.AlertStatus `json:"status"`
				} `json:"items"`
			} `json:"data"`
		}
		if err := json.Unmarshal(qw.Body.Bytes(), &queryResp); err != nil {
			t.Fatalf("Failed to parse query response: %v", err)
		}

		if queryResp.Data.Total != 1 {
			t.Errorf("Expected 1 result, got %d", queryResp.Data.Total)
		}

		if len(queryResp.Data.Items) != 1 {
			t.Errorf("Expected 1 item, got %d", len(queryResp.Data.Items))
		}

		if queryResp.Data.Total == 1 && len(queryResp.Data.Items) == 1 {
			fmt.Println("AlertStatusConsistency: GREEN - status stored and queried with consistent case")
		} else {
			fmt.Println("AlertStatusConsistency: RED - status case mismatch causes query failure")
		}
	})

	if t.Failed() {
		fmt.Println("\n=== RED (红灯，缺陷未修复) ===")
	} else {
		fmt.Println("\n=== GREEN (绿灯，缺陷已修复) ===")
	}
}
