package main

import (
	"log/slog"
	"golang.org/x/sys/windows/svc"
)

const serviceName = "rabbitmq_exporter"

func runService() chan bool {
	stopCh := make(chan bool)
	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		slog.Any("error", err)
		panic(err)
	}
	if !isInteractive {
		go svc.Run(serviceName, &rmqExporterService{stopCh: stopCh})
	}
	return stopCh
}

type rmqExporterService struct {
	stopCh chan<- bool
}

func (s *rmqExporterService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				s.stopCh <- true
				break loop
			default:
				slog.Error("unexpected control request ", c)
			}
		}
		changes <- svc.Status{State: svc.StopPending}
	}
	return
}
