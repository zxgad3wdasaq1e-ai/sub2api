package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func TestExtractEasyPayQRCode(t *testing.T) {
	t.Parallel()

	body := []byte(`<html><body>
		<img src="https://pay.example/logo.png">
		<img alt="二维码" src="data:image/jpg;base64,ZmFrZS1xci1pbWFnZQ==">
	</body></html>`)

	got := extractEasyPayQRCode(body)
	if got != "data:image/jpg;base64,ZmFrZS1xci1pbWFnZQ==" {
		t.Fatalf("QR image = %q", got)
	}
}

func TestExtractEasyPayQRCodeRejectsSVGDataURI(t *testing.T) {
	t.Parallel()

	body := []byte(`<img src="data:image/svg+xml;base64,PHN2Zz48L3N2Zz4=">`)
	if got := extractEasyPayQRCode(body); got != "" {
		t.Fatalf("QR image = %q, want empty", got)
	}
}

func TestEasyPayCreateAPIPaymentExtractsHostedWechatQRCode(t *testing.T) {
	t.Parallel()

	const qrImage = "data:image/png;base64,ZmFrZS13ZWNoYXQtcXI="
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mapi.php":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"code":1,"trade_no":"gateway-42","payurl":%q,"qrcode":""}`, server.URL+"/submit.php?mapiid=42")
		case "/submit.php":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<html><body><img alt="二维码" src="%s"></body></html>`, qrImage)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider, err := NewEasyPay("test", map[string]string{
		"pid":       "1001",
		"pkey":      "secret",
		"apiBase":   server.URL,
		"notifyUrl": server.URL + "/notify",
		"returnUrl": server.URL + "/return",
	})
	if err != nil {
		t.Fatalf("NewEasyPay: %v", err)
	}

	result, err := provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{
		OrderID:     "sub2_42",
		Amount:      "10.00",
		PaymentType: payment.TypeWxpay,
		Subject:     "Recharge",
		ClientIP:    "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	if result.QRCode != qrImage {
		t.Fatalf("QR code = %q, want hosted image", result.QRCode)
	}
	if result.PayURL != server.URL+"/submit.php?mapiid=42" {
		t.Fatalf("pay URL = %q", result.PayURL)
	}
}

func TestEasyPayCreateAPIPaymentKeepsMobileWechatRedirect(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mapi.php" {
			t.Errorf("unexpected hosted QR fetch: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":1,"trade_no":"gateway-mobile","payurl":"https://pay.example/desktop","payurl2":"https://pay.example/mobile","qrcode":""}`)
	}))
	defer server.Close()

	provider, err := NewEasyPay("test", map[string]string{
		"pid":       "1001",
		"pkey":      "secret",
		"apiBase":   server.URL,
		"notifyUrl": server.URL + "/notify",
		"returnUrl": server.URL + "/return",
	})
	if err != nil {
		t.Fatalf("NewEasyPay: %v", err)
	}

	result, err := provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{
		OrderID:     "sub2_mobile",
		Amount:      "10.00",
		PaymentType: payment.TypeWxpay,
		Subject:     "Recharge",
		IsMobile:    true,
	})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	if result.QRCode != "" || result.PayURL != "https://pay.example/mobile" {
		t.Fatalf("unexpected mobile result: qr=%q payURL=%q", result.QRCode, result.PayURL)
	}
}
