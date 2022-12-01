package mailing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	file_reader "github.com/frozosea/file-reader/pkg"
	"io"
	"net/http"
	"net/url"
	"sync"
)

type Response struct {
	Result struct {
		Statuses []struct {
			Id     int64  `json:"id"`
			Status string `json:"status"`
		} `json:"statuses"`
	} `json:"result"`
}

//WithUniSender this struct using for send email with Unisender service (url: https://www.unisender.com/)
type WithUniSender struct {
	reader          *file_reader.FileReader
	senderName      string
	senderEmail     string
	UnisenderApiKey string
	signature       string
}

func NewWithUniSender(senderName string, senderEmail string, unisenderApiKey string, signature string) *WithUniSender {
	return &WithUniSender{reader: file_reader.New(), senderName: senderName, senderEmail: senderEmail, UnisenderApiKey: unisenderApiKey, signature: signature}
}

func (m *WithUniSender) getForm(toAddress, subject, body string) url.Values {
	query := url.Values{}
	query.Set("format", "json")
	query.Set("api_key", m.UnisenderApiKey)
	query.Set("sender_name", m.senderName)
	query.Set("email", toAddress)
	query.Set("sender_email", m.senderEmail)
	query.Set("subject", subject)
	query.Set("body", body)
	query.Set("wrap_type", "STRING")
	query.Set("list_id", "1")
	return query
}
func (m *WithUniSender) checkStatusOfEmail(id string) error {
	client := http.Client{}
	checkStatusUrl := fmt.Sprintf(`https://api.unisender.com/ru/api/checkEmail?format=json&api_key=%s&email_id=%s`, m.UnisenderApiKey, id)
	r, err := client.Get(checkStatusUrl)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	body, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		return readErr
	}
	var s Response
	if unmarshalErr := json.Unmarshal(body, &s); unmarshalErr != nil {
		return unmarshalErr
	}
	for _, v := range s.Result.Statuses {
		if v.Status != "ok_sent" {
			return errors.New("email was not sent successfully")
		}
	}
	return nil
}
func (m *WithUniSender) wrap(ctx context.Context, toAddresses []string, callBack func(toAddress string) error) error {
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
		}
	}
}
func (m *WithUniSender) sendEmail(form url.Values) (string, error) {
	client := http.Client{}
	r, err := client.PostForm("https://api.unisender.com/ru/api/sendEmail", form)
	if err != nil {
		return "", err
	}
	defer r.Body.Close()
	if r.StatusCode > 250 {
		return "", errors.New("bad status code")
	}
	_, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		return "", readErr
	}
	return "", nil
}
func (m *WithUniSender) SendWithFile(ctx context.Context, toAddresses []string, subject, filePath string) error {
	fileName, err := m.reader.GetFileName(filePath)
	if err != nil {
		return err
	}
	readFile, err := m.reader.ReadFile(filePath)
	if err != nil {
		return err
	}
	return m.wrap(ctx, toAddresses, func(toAddress string) error {
		form := m.getForm(toAddress, subject, m.signature)
		form.Set(fmt.Sprintf(`attachments[%s]`, fileName), string(readFile))
		id, sendMailErr := m.sendEmail(form)
		if sendMailErr != nil {
			return sendMailErr
		}
		if err := m.checkStatusOfEmail(id); err != nil {
			return err
		}
		return nil
	})
}
func (m *WithUniSender) SendSimple(ctx context.Context, toAddresses []string, subject, body, _ string) error {
	return m.wrap(ctx, toAddresses, func(toAddress string) error {
		form := m.getForm(toAddress, subject, body)
		id, sendMailErr := m.sendEmail(form)
		if sendMailErr != nil {
			return sendMailErr
		}
		if err := m.checkStatusOfEmail(id); err != nil {
			return err
		}
		return nil
	})
}
