package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/verbeux-ai/whatsmiau/server/dto"
)

func TestEditMessage_BindFailure(t *testing.T) {
	e := echo.New()
	body := bytes.NewBufferString("{invalid json")
	req := httptest.NewRequest(http.MethodPut, "/", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	c := &Chat{}
	if err := c.EditMessage(ctx); err != nil {
		t.Fatalf("expected nil error from Bind failure handling, got: %v", err)
	}
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422 for bind failure, got %d", rec.Code)
	}
}

func TestEditMessage_MissingFields(t *testing.T) {
	e := echo.New()

	tests := []struct {
		name    string
		payload dto.EditMessageRequest
	}{
		{
			name:    "missing id",
			payload: dto.EditMessageRequest{RemoteJid: "5511999999999", NewMessage: "edited"},
		},
		{
			name:    "missing remoteJid",
			payload: dto.EditMessageRequest{ID: "msg123", NewMessage: "edited"},
		},
		{
			name:    "missing newMessage",
			payload: dto.EditMessageRequest{ID: "msg123", RemoteJid: "5511999999999"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.payload)
			req := httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)

			c := &Chat{}
			if err := c.EditMessage(ctx); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status 400 for validation failure, got %d", rec.Code)
			}
		})
	}
}

func TestReplyMessage_BindFailure(t *testing.T) {
	e := echo.New()
	body := bytes.NewBufferString("{invalid json")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	c := &Chat{}
	if err := c.ReplyMessage(ctx); err != nil {
		t.Fatalf("expected nil error from Bind failure handling, got: %v", err)
	}
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422 for bind failure, got %d", rec.Code)
	}
}

func TestReplyMessage_MissingFields(t *testing.T) {
	e := echo.New()

	tests := []struct {
		name    string
		payload dto.ReplyMessageRequest
	}{
		{
			name:    "missing remoteJid",
			payload: dto.ReplyMessageRequest{MessageId: "msg123", Text: "reply"},
		},
		{
			name:    "missing messageId",
			payload: dto.ReplyMessageRequest{RemoteJid: "5511999999999", Text: "reply"},
		},
		{
			name:    "missing text",
			payload: dto.ReplyMessageRequest{RemoteJid: "5511999999999", MessageId: "msg123"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.payload)
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)

			c := &Chat{}
			if err := c.ReplyMessage(ctx); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected status 400 for validation failure, got %d", rec.Code)
			}
		})
	}
}
