package cls

import (
	"testing"
)

// 测试用公钥
const testPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzg447JmU3Cg1UavZfSoP
PexFA0sbCnTlzKBFNS4Uq3qvTp5+Lo/hfO6QBFuLLGgHdv1x3qLGrkg8ZDBgNQGn
FhIWko15xD5ICv/fxNRpKLJwmvCLfqEN5jNLDOpE6cnCFp/9O31ZETzDNJi+fdlD
kwrM+GBZ3E77bJnVJ5veyztEExUa7YnLgN3qWGArQBIOYc30MzAy3+zPYOqdfjWT
6sRHGxGPJEZXC6MJQ0BGqTBNsduv5nCWgKgoksNdhZFl0zvEqOZYqabhOlhIusei
fYAieE9AfFWMGBgosXLo+zRX2e1C9Nf72/GlP76XtU5/+dQ6HvSOsIbXGDTu8lhl
JQIDAQAB
-----END PUBLIC KEY-----`

func TestTokenAuth_Mock(t *testing.T) {
	cls := NewCLSService(testPublicKey, "MiniCatch", "https://cls.mazhangjing.com")
	claims, err := cls.JwtAuth("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImNvcmtpbmUiLCJyb2xlIjoiYWRtaW4iLCJwdXJwb3NlIjoiTWluaUNhdGNoIiwiZXhwIjoxNzUzOTczNzYwLCJuYmYiOjE3NTMxNjczNjcsImlhdCI6MTc1MzE2NzM2N30.n_sWBUPkTjBa8Uuu_gFd5rw7mOvo3ccINuymFamserBos9WPxiOT9FEf9sPvT-bA7GAQ9K5RFdvWhAmDXRHryraxtk1nzf-WQeV6REjVdwbqHZxBQT0e8DnvPLpLkAbFtK-KP_CAlX3bEj-l7Mq8MzsREKJ456qJIIeybIxCwpmyJ--s2iRwcRS-5Y5kTrXMZqxkiGsLjjlLwKglgG7sqBcZNvPiNCkhnr3Jlj6H74_xF3jAdbBflrns_D4SCm-3IhaPvqpKEEnq6Un_oD1ZbvSsHxZozQKYIR4lqmmBvFo0BKyWOVL5yL8nrooF8Cj1SKMihisL_DgqFoyU65lCIQ")
	if err != nil {
		t.Fatalf("JwtAuth failed: %v", err)
	}
	t.Logf("JwtAuth success: %+v", claims)
}

func TestTokenAuthFailed(t *testing.T) {
	cls := NewCLSService(testPublicKey, "MiniCatch2", "https://cls.mazhangjing.com")
	claims, err := cls.JwtAuth("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImNvcmtpbmUiLCJyb2xlIjoiYWRtaW4iLCJwdXJwb3NlIjoiTWluaUNhdGNoIiwiZXhwIjoxNzUzOTczNzYwLCJuYmYiOjE3NTMxNjczNjcsImlhdCI6MTc1MzE2NzM2N30.n_sWBUPkTjBa8Uuu_gFd5rw7mOvo3ccINuymFamserBos9WPxiOT9FEf9sPvT-bA7GAQ9K5RFdvWhAmDXRHryraxtk1nzf-WQeV6REjVdwbqHZxBQT0e8DnvPLpLkAbFtK-KP_CAlX3bEj-l7Mq8MzsREKJ456qJIIeybIxCwpmyJ--s2iRwcRS-5Y5kTrXMZqxkiGsLjjlLwKglgG7sqBcZNvPiNCkhnr3Jlj6H74_xF3jAdbBflrns_D4SCm-3IhaPvqpKEEnq6Un_oD1ZbvSsHxZozQKYIR4lqmmBvFo0BKyWOVL5yL8nrooF8Cj1SKMihisL_DgqFoyU65lCIQ")
	if err != nil {
		t.Logf("JwtAuth failed: %v", err)
	} else {
		t.Fatalf("JwtAuth success: %+v", claims)
	}
}

func TestTokenTotpAuth1(t *testing.T) {
	cls := NewCLSService(testPublicKey, "MiniCatch", "https://cls.mazhangjing.com")
	claims, err := cls.TokenAuth("638963")
	if err != nil {
		t.Logf("TokenAuth failed: %v", err)
	} else {
		t.Fatalf("TokenAuth success: %+v", claims)
	}
}

func TestTokenTotpAuth2(t *testing.T) {
	cls := NewCLSService(testPublicKey, "MiniCatch2", "https://cls.mazhangjing.com")
	claims, err := cls.TokenAuth("638963")
	if err != nil {
		t.Logf("TokenAuth failed: %v", err)
	} else {
		t.Fatalf("TokenAuth success: %+v", claims)
	}
}
