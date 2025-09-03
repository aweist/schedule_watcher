package notifier

import (
	"github.com/aweist/schedule-watcher/models"
)

type Notifier interface {
	SendNotification(game models.Game) error
	GetType() string
}