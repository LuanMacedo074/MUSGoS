package factory_test

import (
	"testing"

	"fsos-server/_tests/testutil"
	"fsos-server/internal/adapters/inbound/mus"
	"fsos-server/internal/factory"
)

func TestNewHandler_SMUS(t *testing.T) {
	logger := &testutil.MockLogger{}
	cipher := &testutil.MockCipher{}
	sessionStore := testutil.NewMockSessionStore()
	connWriter := &testutil.MockConnectionWriter{}
	sender := mus.NewSender(connWriter, sessionStore, logger, nil, false)

	handler, err := factory.NewHandler("smus", logger, cipher, nil, nil, sessionStore, nil, connWriter, sender, "open", 40, false, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handler == nil {
		t.Fatal("handler should not be nil")
	}
}

func TestNewHandler_Unknown(t *testing.T) {
	logger := &testutil.MockLogger{}
	cipher := &testutil.MockCipher{}

	_, err := factory.NewHandler("http", logger, cipher, nil, nil, nil, nil, nil, nil, "open", 40, false, nil, nil, nil)
	if err == nil {
		t.Error("expected error for unknown protocol")
	}
}
