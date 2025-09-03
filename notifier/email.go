package notifier

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	"github.com/aweist/schedule-watcher/models"
)

type EmailNotifier struct {
	smtpHost string
	smtpPort string
	username string
	password string
	from     string
	teamName string
	storage  EmailStorage
}

type EmailStorage interface {
	GetActiveEmailRecipients() ([]models.EmailRecipient, error)
}

type EmailConfig struct {
	SMTPHost string
	SMTPPort string
	Username string
	Password string
	From     string
	TeamName string
	Storage  EmailStorage
}

func NewEmailNotifier(config EmailConfig) *EmailNotifier {
	return &EmailNotifier{
		smtpHost: config.SMTPHost,
		smtpPort: config.SMTPPort,
		username: config.Username,
		password: config.Password,
		from:     config.From,
		teamName: config.TeamName,
		storage:  config.Storage,
	}
}

func (e *EmailNotifier) GetType() string {
	return "email"
}

func (e *EmailNotifier) SendNotification(game models.Game) error {
	// Get recipients from database
	recipients, err := e.getRecipients()
	if err != nil {
		return fmt.Errorf("getting recipients: %w", err)
	}
	
	if len(recipients) == 0 {
		return fmt.Errorf("no active email recipients configured")
	}

	subject := fmt.Sprintf("New Volleyball Game Scheduled - %s", game.Date.Format("Mon, Jan 2"))
	body, err := e.buildEmailBody(game)
	if err != nil {
		return fmt.Errorf("building email body: %w", err)
	}

	message := e.buildMessage(subject, body, recipients)

	auth := smtp.PlainAuth("", e.username, e.password, e.smtpHost)
	addr := fmt.Sprintf("%s:%s", e.smtpHost, e.smtpPort)

	err = smtp.SendMail(addr, auth, e.from, recipients, []byte(message))
	if err != nil {
		return fmt.Errorf("sending email: %w", err)
	}

	return nil
}

func (e *EmailNotifier) getRecipients() ([]string, error) {
	// Storage is required for getting recipients
	if e.storage == nil {
		return nil, fmt.Errorf("no storage configured for email recipients")
	}
	
	dbRecipients, err := e.storage.GetActiveEmailRecipients()
	if err != nil {
		return nil, fmt.Errorf("failed to get recipients from database: %w", err)
	}
	
	var emails []string
	for _, r := range dbRecipients {
		emails = append(emails, r.Email)
	}
	
	if len(emails) == 0 {
		return nil, fmt.Errorf("no active email recipients in database")
	}
	
	return emails, nil
}

func (e *EmailNotifier) buildMessage(subject, body string, recipients []string) string {
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("IVP Game Alerts <%s>", e.from)
	headers["To"] = strings.Join(recipients, ", ")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	var message strings.Builder
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")
	message.WriteString(body)

	return message.String()
}

func (e *EmailNotifier) buildEmailBody(game models.Game) (string, error) {
	tmplStr := `
<!DOCTYPE html>
<html>
<head>
    <style>
        body {
            font-family: Arial, sans-serif;
            line-height: 1.6;
            color: #333;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f4f4f4;
        }
        .header {
            background-color: #2c3e50;
            color: white;
            padding: 20px;
            text-align: center;
            border-radius: 5px 5px 0 0;
        }
        .content {
            background-color: white;
            padding: 30px;
            border-radius: 0 0 5px 5px;
        }
        .game-details {
            background-color: #ecf0f1;
            padding: 15px;
            border-radius: 5px;
            margin: 20px 0;
        }
        .detail-row {
            margin: 10px 0;
            font-size: 16px;
        }
        .label {
            font-weight: bold;
            display: inline-block;
            width: 100px;
        }
        .footer {
            text-align: center;
            margin-top: 20px;
            font-size: 12px;
            color: #7f8c8d;
        }
        .schedule-link {
            display: block;
            text-align: center;
            margin: 25px 0;
        }
        .schedule-link a {
            display: inline-block;
            padding: 12px 30px;
            background-color: #2c3e50;
            color: white;
            text-decoration: none;
            border-radius: 5px;
            font-weight: bold;
            transition: background-color 0.3s;
        }
        .schedule-link a:hover {
            background-color: #34495e;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🏐 New Game Alert!</h1>
        </div>
        <div class="content">            
            <div class="game-details">
                <div class="detail-row">
                    <span class="label">Date:</span> {{.Date}}
                </div>
                <div class="detail-row">
                    <span class="label">Time:</span> {{.Time}}
                </div>
                <div class="detail-row">
                    <span class="label">Court:</span> {{.Court}}
                </div>
                <div class="detail-row">
                    <span class="label">Division:</span> {{.Division}}
                </div>
                <div class="detail-row">
                    <span class="label">Team:</span> {{.TeamCaptain}} (#{{.TeamNumber}})
                </div>
            </div>
            
            <div class="schedule-link">
                <a href="https://winlossdraw.com/ivp" target="_blank">See the full schedule here</a>
            </div>
            
            <p>Remember to suck the head!</p>
            
            <div class="footer">
                <p>This is an automated notification from the IVP Schedule Watcher</p>
            </div>
        </div>
    </div>
</body>
</html>
`

	tmpl, err := template.New("email").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	data := struct {
		TeamName    string
		Date        string
		Time        string
		Court       string
		Division    string
		TeamCaptain string
		TeamNumber  int
	}{
		TeamName:    e.teamName,
		Date:        game.Date.Format("Monday, January 2, 2006"),
		Time:        game.Time,
		Court:       game.Court,
		Division:    game.Division,
		TeamCaptain: game.TeamCaptain,
		TeamNumber:  game.TeamNumber,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
