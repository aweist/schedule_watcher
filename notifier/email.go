package notifier

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"strings"

	"github.com/aweist/schedule-watcher/models"
)

type EmailNotifier struct {
	smtpHost string
	smtpPort string
	username string
	password string
	from     string
}

type EmailConfig struct {
	SMTPHost string
	SMTPPort string
	Username string
	Password string
	From     string
}

func NewEmailNotifier(config EmailConfig) *EmailNotifier {
	return &EmailNotifier{
		smtpHost: config.SMTPHost,
		smtpPort: config.SMTPPort,
		username: config.Username,
		password: config.Password,
		from:     config.From,
	}
}

func (e *EmailNotifier) GetType() string {
	return "email"
}

func (e *EmailNotifier) SendNotification(game models.Game, recipients []string) error {
	if len(recipients) == 0 {
		return fmt.Errorf("no email recipients provided")
	}

	leagueName := strings.ToUpper(game.League)
	subject := fmt.Sprintf("[%s] New Volleyball Game Scheduled - %s", leagueName, game.Date.Format("Mon, Jan 2"))
	body, err := e.buildEmailBody(game)
	if err != nil {
		return fmt.Errorf("building email body: %w", err)
	}

	icsContent := GenerateICS(game)
	message := e.buildMessageWithAttachment(subject, body, recipients, icsContent, game.Date.Format("2006-01-02"), leagueName)

	auth := smtp.PlainAuth("", e.username, e.password, e.smtpHost)
	addr := fmt.Sprintf("%s:%s", e.smtpHost, e.smtpPort)

	err = smtp.SendMail(addr, auth, e.from, recipients, []byte(message))
	if err != nil {
		return fmt.Errorf("sending email: %w", err)
	}

	return nil
}

func (e *EmailNotifier) buildMessageWithAttachment(subject, body string, recipients []string, icsContent string, dateStr string, leagueName string) string {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s Game Alerts <%s>", leagueName, e.from)
	headers["To"] = strings.Join(recipients, ", ")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = fmt.Sprintf("multipart/mixed; boundary=%s", writer.Boundary())

	var message strings.Builder
	for k, v := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")

	htmlPart, _ := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type": []string{"text/html; charset=UTF-8"},
	})
	htmlPart.Write([]byte(body))

	icsPart, _ := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type":              []string{"text/calendar; charset=UTF-8; method=REQUEST"},
		"Content-Transfer-Encoding": []string{"base64"},
		"Content-Disposition":       []string{fmt.Sprintf("attachment; filename=\"volleyball-game-%s.ics\"", dateStr)},
	})

	encoded := base64.StdEncoding.EncodeToString([]byte(icsContent))
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		icsPart.Write([]byte(encoded[i:end] + "\r\n"))
	}

	writer.Close()

	message.Write(buf.Bytes())

	return message.String()
}

func (e *EmailNotifier) buildEmailBody(game models.Game) (string, error) {
	leagueName := strings.ToUpper(game.League)
	scheduleLink := getScheduleLink(game.League)

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
            <h1>New {{.LeagueName}} Game Alert!</h1>
        </div>
        <div class="content">
            <div class="game-details">
                <div class="detail-row">
                    <span class="label">League:</span> {{.LeagueName}}
                </div>
                <div class="detail-row">
                    <span class="label">Date:</span> {{.Date}}
                </div>
                <div class="detail-row">
                    <span class="label">Time:</span> {{.Time}}
                </div>
                <div class="detail-row">
                    <span class="label">Court:</span> {{.Court}}
                </div>
                {{if .Division}}
                <div class="detail-row">
                    <span class="label">Division:</span> {{.Division}}
                </div>
                {{end}}
                <div class="detail-row">
                    <span class="label">Team:</span> {{.TeamCaptain}}{{if .TeamNumber}} (#{{.TeamNumber}}){{end}}
                </div>
                {{if .Opponent}}
                <div class="detail-row">
                    <span class="label">Opponent:</span> {{.Opponent}}
                </div>
                {{end}}
            </div>

            <p style="text-align: center; margin: 20px 0;">
                <em>A calendar invite is attached to this email</em>
            </p>

            {{if .ScheduleLink}}
            <div class="schedule-link">
                <a href="{{.ScheduleLink}}" target="_blank">See the full schedule here</a>
            </div>
            {{end}}

            <div class="footer">
                <p>This is an automated notification from the {{.LeagueName}} Schedule Watcher</p>
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
		LeagueName   string
		Date         string
		Time         string
		Court        string
		Division     string
		TeamCaptain  string
		TeamNumber   int
		Opponent     string
		ScheduleLink string
	}{
		LeagueName:   leagueName,
		Date:         game.Date.Format("Monday, January 2, 2006"),
		Time:         game.Time,
		Court:        game.Court,
		Division:     game.Division,
		TeamCaptain:  game.TeamCaptain,
		TeamNumber:   game.TeamNumber,
		Opponent:     game.Opponent,
		ScheduleLink: scheduleLink,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func getScheduleLink(league string) string {
	switch strings.ToLower(league) {
	case "ivp":
		return "https://winlossdraw.com/ivp"
	case "pins":
		return "https://pins.killerworld.com/schedules.cgi"
	default:
		return ""
	}
}
