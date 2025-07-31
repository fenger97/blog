package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordHash(t *testing.T) {
	plain := "123456" // 用你实际的明文密码
	hash := adminPasswordHash
	// hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// t.Logf("hash: %s", hash)
	// 正确密码
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) != nil {
		t.Error("密码校验失败，应该通过")
	}

	// 错误密码
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte("wrongpass")) == nil {
		t.Error("密码校验错误，应该失败")
	}
}

func TestLoginHandler(t *testing.T) {
	// 正确用户名密码
	body := []byte(`{"username":"admin","password":"123456"}`) // 用你实际的明文密码
	req := httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	LoginHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("登录失败，期望200，实际%d", rr.Code)
	}

	// 错误密码
	body = []byte(`{"username":"admin","password":"wrongpass"}`)
	req = httptest.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()

	LoginHandler(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("错误密码登录，期望401，实际%d", rr.Code)
	}
}
