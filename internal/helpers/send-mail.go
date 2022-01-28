package helpers

import "server_monitor/internal/channeldata"

// SendEmail sends an email
func SendEmail(mailMessage channeldata.MailData) {
	if mailMessage.FromAddress == "" {
		mailMessage.FromAddress = app.PreferenceMap["smtp_from_email"]
		mailMessage.FromName = app.PreferenceMap["smtp_from_name"]
	}

	job := channeldata.MailJob{MailMessage: mailMessage}

	app.MailQueue <- job
}