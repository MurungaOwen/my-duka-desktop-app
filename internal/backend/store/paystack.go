package store

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

type paystackClient struct {
	baseURL    string
	secretKey  string
	httpClient *http.Client
}

var newPaystackHTTPClient = func() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}

type paystackEnvelope struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type paystackChargeData struct {
	Reference       string `json:"reference"`
	Status          string `json:"status"`
	Amount          int64  `json:"amount"`
	Currency        string `json:"currency"`
	Channel         string `json:"channel"`
	DisplayText     string `json:"display_text"`
	GatewayResponse string `json:"gateway_response"`
}

type paystackTransactionList struct {
	Data []paystackTransaction `json:"data"`
}

type paystackTransaction struct {
	Reference       string `json:"reference"`
	Amount          int64  `json:"amount"`
	Currency        string `json:"currency"`
	Channel         string `json:"channel"`
	GatewayResponse string `json:"gateway_response"`
	PaidAt          string `json:"paid_at"`
	CreatedAt       string `json:"created_at"`
	Customer        struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	} `json:"customer"`
	Authorization struct {
		MobileMoneyNumber string `json:"mobile_money_number"`
	} `json:"authorization"`
}

func (s *Service) StartMPesaCharge(input StartMPesaChargeInput) (MPesaChargeSession, error) {
	db, err := s.getDB()
	if err != nil {
		return MPesaChargeSession{}, err
	}

	phone := strings.TrimSpace(input.Phone)
	if phone == "" {
		return MPesaChargeSession{}, errors.New("phone is required")
	}
	normalizedPhone, err := normalizeKEPhone(phone)
	if err != nil {
		return MPesaChargeSession{}, err
	}
	if input.AmountCents <= 0 {
		return MPesaChargeSession{}, errors.New("amount must be greater than zero")
	}

	client, err := s.newPaystackClient(db)
	if err != nil {
		return MPesaChargeSession{}, err
	}

	email := strings.TrimSpace(input.Email)
	if email == "" {
		email = strings.TrimSpace(os.Getenv("PAYSTACK_POS_EMAIL"))
		if email == "" {
			var configured string
			err := db.QueryRow(`SELECT value FROM settings WHERE key = 'paystack_pos_email' AND deleted_at IS NULL LIMIT 1`).Scan(&configured)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return MPesaChargeSession{}, fmt.Errorf("read paystack pos email: %w", err)
			}
			email = strings.TrimSpace(configured)
		}
	}
	if !looksLikeEmail(email) {
		return MPesaChargeSession{}, errors.New("invalid paystack email: set PAYSTACK_POS_EMAIL in .env")
	}

	reference := strings.TrimSpace(input.Reference)
	if reference == "" {
		reference = "mdk_" + uuid.NewString()
	}

	payload := map[string]any{
		"email":     email,
		"amount":    strconv.FormatInt(input.AmountCents, 10),
		"currency":  "KES",
		"reference": reference,
		"mobile_money": map[string]string{
			"phone":    normalizedPhone,
			"provider": "mpesa",
		},
		"metadata": map[string]any{
			"source": "myduka_pos",
			"phone":  normalizedPhone,
		},
	}

	var env paystackEnvelope
	if err := client.doJSON(http.MethodPost, "/charge", payload, &env); err != nil {
		return MPesaChargeSession{}, err
	}
	if !env.Status {
		return MPesaChargeSession{}, fmt.Errorf("paystack charge failed: %s", strings.TrimSpace(env.Message))
	}

	var data paystackChargeData
	if len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, &data); err != nil {
			return MPesaChargeSession{}, fmt.Errorf("decode paystack charge response: %w", err)
		}
	}
	if strings.TrimSpace(data.Reference) == "" {
		data.Reference = reference
	}

	return MPesaChargeSession{
		Reference:   data.Reference,
		Status:      strings.ToLower(strings.TrimSpace(data.Status)),
		DisplayText: strings.TrimSpace(data.DisplayText),
		Message:     strings.TrimSpace(env.Message),
	}, nil
}

func normalizeKEPhone(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", errors.New("phone is required")
	}
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) || r == '+' {
			b.WriteRune(r)
		}
	}
	s = b.String()

	if strings.HasPrefix(s, "+") {
		s = strings.TrimPrefix(s, "+")
	}

	switch {
	case len(s) == 10 && strings.HasPrefix(s, "0"):
		s = "254" + s[1:]
	case len(s) == 9 && (strings.HasPrefix(s, "7") || strings.HasPrefix(s, "1")):
		s = "254" + s
	}

	if len(s) != 12 || !strings.HasPrefix(s, "254") {
		return "", errors.New("invalid phone number format: use +2547XXXXXXXX")
	}
	local := s[3:]
	if !(strings.HasPrefix(local, "7") || strings.HasPrefix(local, "1")) {
		return "", errors.New("invalid phone number format: use +2547XXXXXXXX")
	}
	return "+" + s, nil
}

func looksLikeEmail(v string) bool {
	at := strings.Index(v, "@")
	if at <= 0 || at >= len(v)-1 {
		return false
	}
	dot := strings.LastIndex(v, ".")
	return dot > at+1 && dot < len(v)-1
}

func (s *Service) VerifyMPesaCharge(reference string) (MPesaChargeStatus, error) {
	db, err := s.getDB()
	if err != nil {
		return MPesaChargeStatus{}, err
	}

	ref := strings.TrimSpace(reference)
	if ref == "" {
		return MPesaChargeStatus{}, errors.New("reference is required")
	}

	client, err := s.newPaystackClient(db)
	if err != nil {
		return MPesaChargeStatus{}, err
	}

	var (
		env paystackEnvelope
	)
	err = client.doJSON(http.MethodGet, "/transaction/verify/"+ref, nil, &env)
	if err != nil || !env.Status {
		// Fallback to charge lookup for compatibility across account configurations.
		if fallbackErr := client.doJSON(http.MethodGet, "/charge/"+ref, nil, &env); fallbackErr != nil {
			if err != nil {
				return MPesaChargeStatus{}, err
			}
			return MPesaChargeStatus{}, fallbackErr
		}
		if !env.Status {
			return MPesaChargeStatus{}, fmt.Errorf("paystack verify failed: %s", strings.TrimSpace(env.Message))
		}
	}

	var data paystackChargeData
	if len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, &data); err != nil {
			return MPesaChargeStatus{}, fmt.Errorf("decode paystack verify response: %w", err)
		}
	}

	status := strings.ToLower(strings.TrimSpace(data.Status))
	return MPesaChargeStatus{
		Reference:       ref,
		Status:          status,
		Paid:            status == "success",
		AmountCents:     data.Amount,
		Currency:        strings.ToUpper(strings.TrimSpace(data.Currency)),
		Channel:         strings.ToLower(strings.TrimSpace(data.Channel)),
		GatewayResponse: strings.TrimSpace(data.GatewayResponse),
		DisplayText:     strings.TrimSpace(data.DisplayText),
		Message:         strings.TrimSpace(env.Message),
	}, nil
}

func (s *Service) ListRecentMPesaPayments(input ListRecentMPesaPaymentsInput) ([]RecentMPesaPayment, error) {
	db, err := s.getDB()
	if err != nil {
		return nil, err
	}
	client, err := s.newPaystackClient(db)
	if err != nil {
		return nil, err
	}

	windowMinutes := input.WindowMinutes
	if windowMinutes <= 0 {
		windowMinutes = 15
	}
	limit := input.Limit
	if limit <= 0 || limit > 100 {
		limit = 30
	}

	path := fmt.Sprintf("/transaction?status=success&perPage=%d&page=1", limit)
	var env paystackEnvelope
	if err := client.doJSON(http.MethodGet, path, nil, &env); err != nil {
		return nil, err
	}
	if !env.Status {
		return nil, fmt.Errorf("paystack list transactions failed: %s", strings.TrimSpace(env.Message))
	}

	var list paystackTransactionList
	if len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, &list); err != nil {
			var flat []paystackTransaction
			if err2 := json.Unmarshal(env.Data, &flat); err2 != nil {
				return nil, fmt.Errorf("decode paystack transaction list: %w", err)
			}
			list.Data = flat
		}
	}

	usedRefs, err := s.usedPaymentReferences(db)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().UTC().Add(-time.Duration(windowMinutes) * time.Minute)
	out := make([]RecentMPesaPayment, 0, len(list.Data))
	for _, tx := range list.Data {
		ref := strings.TrimSpace(tx.Reference)
		if ref == "" || usedRefs[ref] {
			continue
		}
		channel := strings.ToLower(strings.TrimSpace(tx.Channel))
		if input.AmountCents > 0 && tx.Amount != input.AmountCents {
			continue
		}
		paidAt := strings.TrimSpace(tx.PaidAt)
		if paidAt == "" {
			paidAt = strings.TrimSpace(tx.CreatedAt)
		}
		if paidAt != "" {
			if t, parseErr := time.Parse(time.RFC3339, paidAt); parseErr == nil {
				if t.UTC().Before(cutoff) {
					continue
				}
			}
		}

		name := strings.TrimSpace(strings.TrimSpace(tx.Customer.FirstName) + " " + strings.TrimSpace(tx.Customer.LastName))
		out = append(out, RecentMPesaPayment{
			Reference:        ref,
			AmountCents:      tx.Amount,
			Currency:         strings.ToUpper(strings.TrimSpace(tx.Currency)),
			Channel:          channel,
			PaidAt:           paidAt,
			GatewayResponse:  strings.TrimSpace(tx.GatewayResponse),
			CustomerEmail:    strings.TrimSpace(tx.Customer.Email),
			CustomerName:     name,
			AuthorizationKey: strings.TrimSpace(tx.Authorization.MobileMoneyNumber),
		})
	}
	return out, nil
}

func (s *Service) usedPaymentReferences(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query(`SELECT reference FROM sale_payments`)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return map[string]bool{}, nil
		}
		return nil, fmt.Errorf("query used payment references: %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var ref string
		if err := rows.Scan(&ref); err != nil {
			return nil, fmt.Errorf("scan used payment reference: %w", err)
		}
		out[strings.TrimSpace(ref)] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate used payment references: %w", err)
	}
	return out, nil
}

func (s *Service) newPaystackClient(db *sql.DB) (*paystackClient, error) {
	secret := strings.TrimSpace(os.Getenv("PAYSTACK_SECRET_KEY"))
	if secret == "" {
		var configured string
		err := db.QueryRow(`SELECT value FROM settings WHERE key = 'paystack_secret_key' AND deleted_at IS NULL LIMIT 1`).Scan(&configured)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("read paystack configuration: %w", err)
		}
		secret = strings.TrimSpace(configured)
	}
	if secret == "" {
		return nil, errors.New("paystack is not configured: set PAYSTACK_SECRET_KEY or settings.paystack_secret_key")
	}

	baseURL := strings.TrimSpace(os.Getenv("PAYSTACK_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.paystack.co"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &paystackClient{
		baseURL:    baseURL,
		secretKey:  secret,
		httpClient: newPaystackHTTPClient(),
	}, nil
}

func (c *paystackClient) doJSON(method, path string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal paystack request: %w", err)
		}
		body = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("create paystack request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call paystack %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read paystack response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("paystack %s %s returned %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(rawBody)))
	}

	if out == nil || len(rawBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(rawBody, out); err != nil {
		return fmt.Errorf("decode paystack response: %w", err)
	}
	return nil
}
