package mailing

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/gomail.v2"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

type WithElasticEmail struct {
	smtpHost  string
	smtpPort  int
	fromEmail string
	password  string
	authKey   string
	listName  string
}

func NewWithElasticEmail(smtpHost string, smtpPort int, fromEmail string, password string, authKey, listName string) (*WithElasticEmail, error) {
	m := &WithElasticEmail{smtpHost: smtpHost, smtpPort: smtpPort, fromEmail: fromEmail, password: password, authKey: authKey, listName: listName}
	if err := m.CreateList(); err != nil {
		return nil, err
	}
	if err := m.AddContactToList(context.Background(), m.fromEmail); err != nil {
		return nil, err
	}
	return m, nil
}
func (w *WithElasticEmail) wrap(ctx context.Context, toAddresses []string, callBack func(toEmail string) error) error {
	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, toEmail := range toAddresses {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer mu.Unlock()
			w.AddContactToList(ctx, toEmail)
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
func (w *WithElasticEmail) SendSimple(ctx context.Context, toAddress []string, subject, message, textType string) error {
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
func (w *WithElasticEmail) SendWithFile(ctx context.Context, toAddresses []string, subject, filePath string) error {
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
func (w *WithElasticEmail) AddContactToList(ctx context.Context, email string) error {
	url := fmt.Sprintf(`https://api.elasticemail.com/v4/lists/%s/contacts`, w.listName)
	strReprOfBody := fmt.Sprintf(`{
		"Emails": ["%s"]
	}`, email)
	cli := &http.Client{}
	var data = strings.NewReader(strReprOfBody)
	req, err := http.NewRequestWithContext(ctx, "POST", url, data)
	if err != nil {
		return err
	}
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7,zh-TW;q=0.6,zh-CN;q=0.5,zh;q=0.4")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ElasticEmail-ApiKey", w.authKey)
	response, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode > 300 {
		b, _ := ioutil.ReadAll(response.Body)
		var resp struct {
			Error string `json:"Error"`
		}
		if err := json.Unmarshal(b, &resp); err != nil {
			return errors.New("incorrect status code")
		}
		return errors.New(resp.Error)
	}
	return nil
}

func (w *WithElasticEmail) CreateList() error {
	cli := &http.Client{}
	strReprOfBody := fmt.Sprintf(`{
  		"ListName": "%s"
	}`, w.listName)
	var data = strings.NewReader(strReprOfBody)
	req, err := http.NewRequest("POST", "https://api.elasticemail.com/v4/lists", data)
	if err != nil {
		return err
	}
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7,zh-TW;q=0.6,zh-CN;q=0.5,zh;q=0.4")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ElasticEmail-ApiKey", w.authKey)
	response, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 201 {
		b, _ := ioutil.ReadAll(response.Body)
		var resp struct {
			Error string `json:"Error"`
		}
		if err := json.Unmarshal(b, &resp); err != nil {
			return errors.New("incorrect status code")
		}
		if strings.ToLower(resp.Error) == strings.ToLower("A list with the given name already exists.") {
			return nil
		} else {
			return errors.New(resp.Error)
		}
	}
	return nil
}
