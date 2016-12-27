package app

import (
	"fmt"
	"time"

	"github.com/u8slvn/raspigosms/database"
	"github.com/u8slvn/raspigosms/gsm"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// NewSenderWorker creates, and returns a new SenderWorker object.
func NewSenderWorker(modem *gsm.Modem) SenderWorker {
	worker := SenderWorker{
		WorkerQueue: make(chan SmsRequest),
		Modem:       modem,
		QuitChan:    make(chan bool),
	}

	return worker
}

// SenderWorker struct
type SenderWorker struct {
	WorkerQueue chan SmsRequest
	Modem       *gsm.Modem
	QuitChan    chan bool
}

// Start function "starts" an infinite loop which consume the SmsQueue.
func (w SenderWorker) Start() {
	fmt.Printf("SenderWorker starting...\n")
	go func() {
		for {
			select {
			case smsr := <-w.WorkerQueue:
				fmt.Printf("worker: => to : %s, message : %s\n", smsr.Sms.Phone, smsr.Sms.Message)
				go func() {
					time.Sleep(1000 * time.Millisecond)
					status := gsm.SmsStatusFailed
					err := w.Modem.SendSms(smsr.Sms)
					if err == nil {
						smsr.RemainingAttempts = 0
						status = gsm.SmsStatusSent
						fmt.Printf("Success\n")
					} else {
						smsr.RemainingAttempts--
						fmt.Printf("Failed\n")
					}

					if smsr.RemainingAttempts > 0 {
						SmsRequestQueue <- smsr
						return
					}

					change := mgo.Change{
						Update:    bson.M{"$set": bson.M{"status": status}},
						ReturnNew: true,
					}
					database.DBConnection.C("sms").FindId(smsr.Sms.UUID).Apply(change, &smsr.Sms)
				}()
			case <-w.QuitChan:
				fmt.Printf("worker stopping\n")
				return
			}
		}
	}()
}

// Stop the worker to stop listening for Sms requests.
// The worker will only stop *after* it has finished its work.
func (w SenderWorker) Stop() {
	go func() {
		w.QuitChan <- true
	}()
}
