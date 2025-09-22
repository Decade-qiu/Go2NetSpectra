package alerter

import (
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/factory"
	"Go2NetSpectra/internal/model"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

const TaskNum = 255

// Alerter evaluates data snapshots against a set of rules and triggers notifications.
type Alerter struct {
	cfg        config.AlerterConfig
	notifiers  []model.Notifier
	taskGroups []factory.TaskGroup
	done       chan struct{}
}

// NewAlerter creates a new Alerter.
func NewAlerter(cfg config.AlerterConfig, notifiers []model.Notifier, taskGroups []factory.TaskGroup) *Alerter {
	return &Alerter{
		cfg:        cfg,
		notifiers:  notifiers,
		taskGroups: taskGroups,
		done:       make(chan struct{}),
	}
}

// Run starts the alerter's main loop for periodic evaluation.
func (a *Alerter) Run() {
	if !a.cfg.Enabled {
		log.Println("Alerter is disabled.")
		return
	}

	interval, err := time.ParseDuration(a.cfg.CheckInterval)
	if err != nil {
		log.Printf("ERROR: Invalid alerter check_interval '%s', alerter will not run.", a.cfg.CheckInterval)
		return
	}

	log.Printf("Alerter started with check interval %s", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.evaluateAllTasks()
		case <-a.done:
			log.Println("Alerter shutting down.")
			return
		}
	}
}

// Stop gracefully shuts down the alerter.
func (a *Alerter) Stop() {
	if a.cfg.Enabled {
		close(a.done)
	}
	a.evaluateAllTasks()
}

// evaluateAllTasks concurrently calls AlerterMsg on all tasks and sends a consolidated report.
func (a *Alerter) evaluateAllTasks() {
	var wg sync.WaitGroup
	resultsChan := make(chan string, TaskNum) // Buffered channel to collect results

	for _, group := range a.taskGroups {
		for _, task := range group.Tasks {
			wg.Add(1)
			go func(t model.Task) {
				defer wg.Done() 
				// Find rules relevant to this task
				var relevantRules []config.AlerterRule
				for _, rule := range a.cfg.Rules {
					if rule.TaskName == t.Name() {
						relevantRules = append(relevantRules, rule)
					}
				}

				if len(relevantRules) > 0 {
					if msg := t.AlerterMsg(relevantRules); msg != "" {
						resultsChan <- msg
					}
				}
			}(task)
		}
	}

	wg.Wait()
	close(resultsChan)

	var allMessages []string
	for msg := range resultsChan {
		allMessages = append(allMessages, msg)
	}

	log.Printf("Alerter evaluation completed. %d alert(s) triggered, will send %d notifications", len(allMessages), len(a.notifiers))

	if len(allMessages) > 0 {
		log.Printf("INFO: Triggered %d alert(s). Sending notification...", len(allMessages))
		subject := fmt.Sprintf("Go2NetSpectra Alert Summary (%d Triggered)", len(allMessages))
		body := "<h1>Go2NetSpectra Alert Summary</h1>" +
			"<p>The following alerts were triggered during the last check:</p><hr>" +
			strings.Join(allMessages, "<hr>")

		for _, notifier := range a.notifiers {
			if err := notifier.Send(subject, body); err != nil {
				log.Printf("ERROR: Failed to send consolidated alert notification: %v", err)
			} else {
				log.Printf("INFO: Consolidated alert notification sent via %T", notifier)
			}
		}
	}
}