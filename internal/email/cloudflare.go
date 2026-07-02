package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
)

type CloudflareSender struct {
	AccountID string
	APIToken  string
	From      string
	Client    *http.Client
}

func (s CloudflareSender) SendVerificationCode(ctx context.Context, to, code, locale string) error {
	if locale == "en" {
		text := fmt.Sprintf("Your Gaia Calendar verification code is %s. It expires in 10 minutes.", code)
		html := fmt.Sprintf("<p>Your Gaia Calendar verification code is <strong>%s</strong>.</p><p>It expires in 10 minutes.</p>", code)
		return s.send(ctx, to, "Gaia Calendar verification code", text, html)
	}
	text := fmt.Sprintf("你的 Gaia Calendar 驗證碼是 %s，10 分鐘後失效。", code)
	html := fmt.Sprintf("<p>你的 Gaia Calendar 驗證碼是 <strong>%s</strong>。</p><p>此驗證碼會在 10 分鐘後失效。</p>", code)
	return s.send(ctx, to, "Gaia Calendar 驗證碼", text, html)
}

func (s CloudflareSender) SendPasswordReset(ctx context.Context, to, resetURL, locale string) error {
	if locale == "en" {
		text := fmt.Sprintf("Open this link to reset your Gaia Calendar password. It expires in 30 minutes:\n%s", resetURL)
		html := fmt.Sprintf("<p>Open this link to reset your Gaia Calendar password. It expires in 30 minutes:</p><p><a href=%q>Reset password</a></p>", resetURL)
		return s.send(ctx, to, "Reset your Gaia Calendar password", text, html)
	}
	text := fmt.Sprintf("請打開以下連結重設你的 Gaia Calendar 密碼。連結會在 30 分鐘後失效：\n%s", resetURL)
	html := fmt.Sprintf("<p>請打開以下連結重設你的 Gaia Calendar 密碼。連結會在 30 分鐘後失效：</p><p><a href=%q>重設密碼</a></p>", resetURL)
	return s.send(ctx, to, "重設你的 Gaia Calendar 密碼", text, html)
}

func (s CloudflareSender) SendGaiaCredentialWarning(ctx context.Context, to, reason, locale string) error {
	escapedReason := html.EscapeString(reason)
	if locale == "en" {
		text := fmt.Sprintf("Gaia Calendar could not sync your roster several times in a row. Your Gaia password may have expired or changed.\n\nLast error: %s", reason)
		htmlBody := fmt.Sprintf("<p>Gaia Calendar could not sync your roster several times in a row.</p><p>Your Gaia password may have expired or changed.</p><p>Last error: %s</p>", escapedReason)
		return s.send(ctx, to, "Gaia Calendar sync needs attention", text, htmlBody)
	}
	text := fmt.Sprintf("Gaia Calendar 已連續多次無法同步你的排班。你的 Gaia 密碼可能已過期或已更改。\n\n最後錯誤：%s", reason)
	htmlBody := fmt.Sprintf("<p>Gaia Calendar 已連續多次無法同步你的排班。</p><p>你的 Gaia 密碼可能已過期或已更改。</p><p>最後錯誤：%s</p>", escapedReason)
	return s.send(ctx, to, "Gaia Calendar 同步需要處理", text, htmlBody)
}

func (s CloudflareSender) send(ctx context.Context, to, subject, text, html string) error {
	if s.AccountID == "" || s.APIToken == "" || s.From == "" {
		return fmt.Errorf("cloudflare email is not configured")
	}
	body := map[string]string{
		"to":      to,
		"from":    s.From,
		"subject": subject,
		"text":    text,
		"html":    html,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/email/sending/send", s.AccountID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.APIToken)
	req.Header.Set("Content-Type", "application/json")
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("cloudflare email returned %s", res.Status)
	}
	return nil
}
