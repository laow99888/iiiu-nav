package auth

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

var testPasswordParams = PasswordParams{Memory: 7 * 1024, Time: 1, Threads: 1}

func TestPasswordHashRoundTrip(t *testing.T) {
	t.Parallel()

	encoded, err := hashPassword("correct horse battery staple", testPasswordParams, bytes.NewReader(bytes.Repeat([]byte{1}, 32)))
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$v=19$m=7168,t=1,p=1$") {
		t.Fatalf("unexpected hash format: %s", encoded)
	}
	valid, err := verifyPassword("correct horse battery staple", encoded)
	if err != nil || !valid {
		t.Fatalf("verify correct password: valid=%v err=%v", valid, err)
	}
	valid, err = verifyPassword("incorrect password value", encoded)
	if err != nil || valid {
		t.Fatalf("verify incorrect password: valid=%v err=%v", valid, err)
	}
}

func TestPasswordValidationAndHashBounds(t *testing.T) {
	t.Parallel()

	for _, password := range []string{"short", strings.Repeat("a", maxPasswordBytes+1), "密码足够长但字符不够"} {
		if !errors.Is(ValidatePassword(password), ErrInvalidPassword) {
			t.Fatalf("expected password %q to be rejected", password)
		}
	}
	if err := ValidatePassword("这是一个足够长的管理员密码值"); err != nil {
		t.Fatalf("expected Unicode password to pass: %v", err)
	}

	malformed := []string{
		"plain text",
		"$argon2id$v=19$m=999999,t=1,p=1$YWJjZGVmZ2g$YWJjZGVmZ2hpamtsbW5vcA",
		"$argon2id$v=18$m=7168,t=1,p=1$YWJjZGVmZ2g$YWJjZGVmZ2hpamtsbW5vcA",
	}
	for _, encoded := range malformed {
		if _, err := verifyPassword("a password long enough", encoded); err == nil {
			t.Fatalf("expected malformed hash to fail: %s", encoded)
		}
	}
}
