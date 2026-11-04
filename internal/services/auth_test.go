package services

import "testing"

func TestHashVerify(t *testing.T) {
	encoded, err := hashPassword("senha-secreta")
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	if encoded == "" {
		t.Fatal("hash vazio")
	}
	svc := &AuthService{}
	if !svc.verifyPassword(encoded, "senha-secreta") {
		t.Error("senha correta deveria ser valida")
	}
	if svc.verifyPassword(encoded, "senha-errada") {
		t.Error("senha errada nao deveria ser valida")
	}
	if svc.verifyPassword("nao-e-um-hash", "qualquer") {
		t.Error("hash invalido nao deveria ser valido")
	}
}
