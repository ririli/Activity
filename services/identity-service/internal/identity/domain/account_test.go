package domain

import "testing"

func TestValidateAccount(t *testing.T) {
	for _, tc := range []struct {
		name, password, nickname string
		valid                    bool
	}{
		{"User_1", "密码密码密码", "昵称", true},
		{"ab", "123456", "", false},
		{"用户名", "123456", "", false},
		{"User", "12345", "", false},
		{"User", "123456", string([]byte{0xff}), false},
	} {
		err := ValidateAccount(tc.name, tc.password, tc.nickname)
		if (err == nil) != tc.valid {
			t.Errorf("%q: got %v", tc.name, err)
		}
	}
}
