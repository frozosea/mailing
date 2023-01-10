package mailing

import (
	"context"
	"crypto/tls"
	"gopkg.in/gomail.v2"
	"sync"
)

type Mailing struct {
	smtpHost  string
	smtpPort  int
	fromEmail string
	password  string
}

func NewMailing(smtpHost string, smtpPort int, fromEmail string, password string) *Mailing {
	return &Mailing{smtpHost: smtpHost, smtpPort: smtpPort, fromEmail: fromEmail, password: password}
}

func (w *Mailing) wrap(ctx context.Context, toAddresses []string, callBack func(toEmail string) error) error {
	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, toEmail := range toAddresses {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer mu.Unlock()
			mu.Lock()
			if err := callBack(toEmail); err != nil {
				errCh <- err
			}
		}()
		wg.Wait()
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errCh:
			return err
		default:
			return nil
		}
	}
}
func (w *Mailing) SendSimple(ctx context.Context, toAddress []string, subject, message, textType string) error {
	return w.wrap(ctx, toAddress, func(toEmail string) error {
		m := gomail.NewMessage()
		m.SetHeader("From", w.fromEmail)
		m.SetHeader("To", toEmail)
		m.SetHeader("Subject", subject)
		m.SetBody(textType, message)
		d := gomail.NewDialer(w.smtpHost, w.smtpPort, w.fromEmail, w.password)
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
		if err := d.DialAndSend(m); err != nil {
			return err
		}
		return nil
	})
}
func (w *Mailing) SendWithFile(ctx context.Context, toAddresses []string, subject, filePath string) error {
	return w.wrap(ctx, toAddresses, func(toEmail string) error {
		m := gomail.NewMessage()
		m.SetHeader("From", w.fromEmail)
		m.SetHeader("To", toEmail)
		m.SetHeader("Subject", subject)
		m.Attach(filePath)
		d := gomail.NewDialer(w.smtpHost, w.smtpPort, w.fromEmail, w.password)
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
		if err := d.DialAndSend(m); err != nil {
			return err
		}
		return nil
	})
}
