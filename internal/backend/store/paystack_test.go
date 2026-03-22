package store

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func setupStoreService(t *testing.T) *Service {
	t.Helper()
	svc, err := NewService(Config{
		DBPath:     filepath.Join(t.TempDir(), "myduka-test.db"),
		Mode:       DeploymentModeStandalone,
		DeviceID:   "store-test-device",
		DeviceName: "store-test-device",
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("start service: %v", err)
	}
	t.Cleanup(func() {
		_ = svc.Close()
	})
	return svc
}

func TestStartAndVerifyMPesaCharge(t *testing.T) {
	svc := setupStoreService(t)

	origFactory := newPaystackHTTPClient
	t.Cleanup(func() { newPaystackHTTPClient = origFactory })

	newPaystackHTTPClient = func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				body := `{"status":true,"message":"ok","data":{"reference":"mdk_ref_123","status":"pending","display_text":"Approve on phone"}}`
				if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/transaction/verify/") {
					body = `{"status":true,"message":"verified","data":{"reference":"mdk_ref_123","status":"success","gateway_response":"Successful"}}`
				}
				if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/charge/") {
					body = `{"status":true,"message":"verified","data":{"reference":"mdk_ref_123","status":"success","gateway_response":"Successful"}}`
				}
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}, nil
			}),
		}
	}

	t.Setenv("PAYSTACK_SECRET_KEY", "test_secret")
	t.Setenv("PAYSTACK_BASE_URL", "https://api.paystack.test")
	t.Setenv("PAYSTACK_POS_EMAIL", "sales@myduka.co.ke")

	session, err := svc.StartMPesaCharge(StartMPesaChargeInput{
		Phone:       "0712345678",
		AmountCents: 5000,
	})
	if err != nil {
		t.Fatalf("start charge: %v", err)
	}
	if session.Reference != "mdk_ref_123" {
		t.Fatalf("unexpected reference: %s", session.Reference)
	}
	if session.Status != "pending" {
		t.Fatalf("expected pending, got %s", session.Status)
	}

	status, err := svc.VerifyMPesaCharge(session.Reference)
	if err != nil {
		t.Fatalf("verify charge: %v", err)
	}
	if !status.Paid {
		t.Fatalf("expected paid=true")
	}
	if status.Status != "success" {
		t.Fatalf("expected success, got %s", status.Status)
	}
}

func TestStartMPesaChargeMissingConfig(t *testing.T) {
	svc := setupStoreService(t)

	t.Setenv("PAYSTACK_SECRET_KEY", "")
	t.Setenv("PAYSTACK_BASE_URL", "")
	t.Setenv("PAYSTACK_POS_EMAIL", "sales@myduka.co.ke")

	_, err := svc.StartMPesaCharge(StartMPesaChargeInput{
		Phone:       "0712345678",
		AmountCents: 1000,
	})
	if err == nil {
		t.Fatalf("expected paystack config error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "not configured") {
		t.Fatalf("expected not configured error, got %v", err)
	}
}

func TestVerifyMPesaChargeFallsBackToChargeLookup(t *testing.T) {
	svc := setupStoreService(t)

	origFactory := newPaystackHTTPClient
	t.Cleanup(func() { newPaystackHTTPClient = origFactory })

	newPaystackHTTPClient = func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/transaction/verify/") {
					return &http.Response{
						StatusCode: 404,
						Body:       io.NopCloser(strings.NewReader(`{"status":false,"message":"not found"}`)),
						Header:     make(http.Header),
					}, nil
				}
				if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/charge/") {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(`{"status":true,"message":"ok","data":{"reference":"mdk_ref_456","status":"success"}}`)),
						Header:     make(http.Header),
					}, nil
				}
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"status":true,"message":"ok","data":{"reference":"mdk_ref_456","status":"pending"}}`)),
					Header:     make(http.Header),
				}, nil
			}),
		}
	}

	t.Setenv("PAYSTACK_SECRET_KEY", "test_secret")
	t.Setenv("PAYSTACK_BASE_URL", "https://api.paystack.test")
	t.Setenv("PAYSTACK_POS_EMAIL", "sales@myduka.co.ke")

	status, err := svc.VerifyMPesaCharge("mdk_ref_456")
	if err != nil {
		t.Fatalf("verify with fallback: %v", err)
	}
	if !status.Paid {
		t.Fatalf("expected paid=true via fallback")
	}
}

func TestStartMPesaChargeNormalizesPhoneAndUsesEnvEmail(t *testing.T) {
	svc := setupStoreService(t)

	origFactory := newPaystackHTTPClient
	t.Cleanup(func() { newPaystackHTTPClient = origFactory })

	var captured map[string]any
	newPaystackHTTPClient = func() *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodPost && r.URL.Path == "/charge" {
					raw, _ := io.ReadAll(r.Body)
					_ = json.Unmarshal(raw, &captured)
				}
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"status":true,"message":"ok","data":{"reference":"mdk_ref_999","status":"pending"}}`)),
					Header:     make(http.Header),
				}, nil
			}),
		}
	}

	t.Setenv("PAYSTACK_SECRET_KEY", "test_secret")
	t.Setenv("PAYSTACK_BASE_URL", "https://api.paystack.test")
	t.Setenv("PAYSTACK_POS_EMAIL", "sales@myduka.co.ke")

	_, err := svc.StartMPesaCharge(StartMPesaChargeInput{
		Phone:       "0712 345 678",
		AmountCents: 2500,
	})
	if err != nil {
		t.Fatalf("start charge: %v", err)
	}
	if captured == nil {
		t.Fatalf("expected request payload to be captured")
	}

	mm, ok := captured["mobile_money"].(map[string]any)
	if !ok {
		t.Fatalf("missing mobile_money payload")
	}
	if mm["phone"] != "+254712345678" {
		t.Fatalf("expected normalized phone +254712345678, got %v", mm["phone"])
	}
	email, _ := captured["email"].(string)
	if email != "sales@myduka.co.ke" {
		t.Fatalf("expected env email sales@myduka.co.ke, got %s", email)
	}
}

func TestStartMPesaChargeMissingPOSEmail(t *testing.T) {
	svc := setupStoreService(t)

	t.Setenv("PAYSTACK_SECRET_KEY", "test_secret")
	t.Setenv("PAYSTACK_BASE_URL", "https://api.paystack.test")
	t.Setenv("PAYSTACK_POS_EMAIL", "")

	_, err := svc.StartMPesaCharge(StartMPesaChargeInput{
		Phone:       "0712345678",
		AmountCents: 1000,
	})
	if err == nil {
		t.Fatalf("expected missing pos email error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "paystack email") {
		t.Fatalf("expected paystack email error, got %v", err)
	}
}
