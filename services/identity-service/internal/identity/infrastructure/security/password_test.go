package security

import "testing"

func TestPasswordHash(t *testing.T) {
	hash, err := HashPassword("密码 secret")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := VerifyPassword(hash, "密码 secret"); err != nil || !ok {
		t.Fatalf("valid password: %v %v", ok, err)
	}
	if ok, err := VerifyPassword(hash, "wrong"); err != nil || ok {
		t.Fatalf("wrong password: %v %v", ok, err)
	}
	if _, err := VerifyPassword("$argon2id$v=19$m=999999999,t=1,p=1$aa$aa", "x"); err == nil {
		t.Fatal("accepted unbounded hash")
	}
}
