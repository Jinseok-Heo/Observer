package main

import (
	"bytes"
	"fmt"
	"github.com/aymerick/douceur/inliner"
	mail "github.com/xhit/go-simple-mail/v2"
	"html/template"
	"jaytaylor.com/html2text"
	"server_monitor/internal/channeldata"
	"strconv"
	"time"
)

type TemplateData struct {
	Content       template.HTML
	From          string
	FromName      string
	PreferenceMap map[string]string
	IntMap        map[string]int
	StringMap     map[string]string
	FloatMap      map[string]float32
	RowSets       map[string]interface{}
}

func NewTemplateData(mailMessage channeldata.MailData) *TemplateData {
	return &TemplateData{
		Content:       mailMessage.Content,
		From:          mailMessage.FromAddress,
		FromName:      mailMessage.FromName,
		PreferenceMap: preferenceMap,
		IntMap:        mailMessage.IntMap,
		StringMap:     mailMessage.StringMap,
		FloatMap:      mailMessage.FloatMap,
		RowSets:       mailMessage.RowSets,
	}
}

func (td *TemplateData) toString(tmpl *template.Template) string {
	var buff bytes.Buffer
	if err := tmpl.Execute(&buff, td); err != nil {
		fmt.Println(err)
	}

	return buff.String()
}

type Worker struct {
	id         int
	jobQueue   chan channeldata.MailJob
	workerPool chan chan channeldata.MailJob
	quitChan   chan bool
}

func NewWorker(id int, workerPool chan chan channeldata.MailJob) Worker {
	return Worker{
		id:         id,
		jobQueue:   make(chan channeldata.MailJob),
		workerPool: workerPool,
		quitChan:   make(chan bool),
	}
}

func (w Worker) start() {
	go func() {
		for {
			w.workerPool <- w.jobQueue

			select {
			case job := <-w.jobQueue:
				w.processMailQueueJob(job.MailMessage)
			case <-w.quitChan:
				fmt.Printf("worker%d stopping\n", w.id)
			}
		}
	}()
}

func (w Worker) stop() {
	go func() {
		w.quitChan <- true
	}()
}

func (w Worker) processMailQueueJob(mailMessage channeldata.MailData) {
	tmpl, err := getTemplateString(mailMessage)
	if err != nil {
		fmt.Println(err)
		return
	}

	alternativeText := getAlternativeText(tmpl)
	formattedMessage := formatTemplateString(tmpl)

	smtpClient, err := newSMTPClient()
	if err != nil {
		fmt.Println(err)
		return
	}

	email := newSMTPEmail(mailMessage, formattedMessage, alternativeText)

	if err = email.Send(smtpClient); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Email sent!")
	}
}

func getTemplate(tmplString string) (*template.Template, error) {
	if tmplString == "" {
		tmplString = "bootstrap.mail.tmpl"
	}

	tmpl, ok := app.TemplateCache[tmplString]
	if !ok {
		return nil, fmt.Errorf("could not get mail template %s", tmplString)
	}
	return tmpl, nil
}

func getTemplateString(mailMessage channeldata.MailData) (string, error) {
	tmpl, err := getTemplate(mailMessage.Template)
	if err != nil {
		return "", err
	}
	tmplData := NewTemplateData(mailMessage)

	return tmplData.toString(tmpl), nil
}

func formatTemplateString(tmpl string) string {
	formattedString, err := inliner.Inline(tmpl)
	if err != nil {
		fmt.Println(err)
		formattedString = tmpl
	}
	return formattedString
}

func getAlternativeText(tmpl string) string {
	alternativeText, err := html2text.FromString(tmpl, html2text.Options{PrettyTables: true})

	if err != nil {
		return ""
	}
	return alternativeText
}

func newSMTPServer() *mail.SMTPServer {
	server := mail.NewSMTPClient()

	port, _ := strconv.Atoi(preferenceMap["smtp_port"])
	server.Host = preferenceMap["smtp_server"]
	server.Port = port
	server.Username = preferenceMap["smtp_user"]
	server.Password = preferenceMap["smtp_password"]

	if preferenceMap["smtp_server"] == "localhost" {
		server.Authentication = mail.AuthPlain
	} else {
		server.Authentication = mail.AuthLogin
	}
	server.Encryption = mail.EncryptionTLS
	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second

	return server
}

func newSMTPClient() (*mail.SMTPClient, error) {
	server := newSMTPServer()

	return server.Connect()
}

func newSMTPEmail(mailMessage channeldata.MailData, formattedMessage, alternativeText string) *mail.Email {
	email := mail.NewMSG()
	email.SetFrom(mailMessage.FromAddress).
		AddTo(mailMessage.ToAddress).
		SetSubject(mailMessage.Subject)

	if len(mailMessage.AdditionalTo) > 0 {
		for _, x := range mailMessage.AdditionalTo {
			email.AddTo(x)
		}
	}
	if len(mailMessage.CC) > 0 {
		for _, x := range mailMessage.CC {
			email.AddCc(x)
		}
	}
	if len(mailMessage.Attachments) > 0 {
		for _, x := range mailMessage.Attachments {
			email.AddAttachment(x)
		}
	}
	email.SetBody(mail.TextHTML, formattedMessage)
	email.AddAlternative(mail.TextPlain, alternativeText)

	return email
}

type Dispatcher struct {
	workerPool chan chan channeldata.MailJob
	maxWorkers int
	jobQueue   chan channeldata.MailJob
}

func NewDispatcher(jobQueue chan channeldata.MailJob, maxWorkers int) *Dispatcher {
	return &Dispatcher{
		workerPool: make(chan chan channeldata.MailJob, maxWorkers),
		maxWorkers: maxWorkers,
		jobQueue:   jobQueue,
	}
}

func (d *Dispatcher) run() {
	for i := 0; i < d.maxWorkers; i++ {
		worker := NewWorker(i+1, d.workerPool)
		worker.start()
	}
	go d.dispatch()
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case job := <-d.jobQueue:
			go func() {
				workerJobQueue := <-d.workerPool
				workerJobQueue <- job
			}()
		}
	}
}
