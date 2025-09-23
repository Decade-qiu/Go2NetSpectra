package alerter

import (
	v1 "Go2NetSpectra/api/gen/v1"
	"Go2NetSpectra/internal/config"
	"Go2NetSpectra/internal/model"
	"context"
	"fmt"
	"log"
	"strings"
	sync "sync"
	time "time"

	"github.com/gomarkdown/markdown"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Alerter is responsible for evaluating task snapshots against predefined rules
// and triggering notifications if rules are violated.
type Alerter struct {
	tasks         []model.Task
	rules         []config.AlerterRule
	notifier      model.Notifier
	checkInterval time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup

	// AI analysis components
	aiEnabled bool
	aiClient  v1.AIServiceClient
}

// NewAlerter creates a new Alerter instance.
func NewAlerter(cfg *config.AlerterConfig, tasks []model.Task, notifier model.Notifier) (*Alerter, error) {
	interval, err := time.ParseDuration(cfg.CheckInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid check_interval for alerter: %w", err)
	}

	a := &Alerter{
		tasks:         tasks,
		rules:         cfg.Rules,
		notifier:      notifier,
		checkInterval: interval,
		stopChan:      make(chan struct{}),
		aiEnabled:     cfg.AIAnalysis.Enabled,
	}

	if a.aiEnabled {
		log.Printf("AI analysis is enabled, connecting to AI service at %s", cfg.AIAnalysis.ServiceAddr)
		conn, err := grpc.NewClient(cfg.AIAnalysis.ServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, fmt.Errorf("failed to connect to AI service: %w", err)
		}
		a.aiClient = v1.NewAIServiceClient(conn)
	}

	return a, nil
}

// Start begins the periodic evaluation of alert rules.
func (a *Alerter) Start() {
	log.Println("Alerter started")
	
	a.wg.Add(1)
	defer a.wg.Done()

	ticker := time.NewTicker(a.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.evaluateAllTasks()
		case <-a.stopChan:
			return
		}
	}
}

// Stop gracefully stops the alerter's evaluation loop.
func (a *Alerter) Stop() {
	log.Println("Stopping Alerter...")
	close(a.stopChan)
	a.wg.Wait()
	a.evaluateAllTasks()
}

// evaluateAllTasks orchestrates the concurrent evaluation of all tasks against the rules.
func (a *Alerter) evaluateAllTasks() {
	var wg sync.WaitGroup
	resultsChan := make(chan string, len(a.tasks)) // Buffered channel

	for _, task := range a.tasks {
		wg.Add(1)
		go func(t model.Task) {
			defer wg.Done()
			// Find rules relevant to this task
			var relevantRules []config.AlerterRule
			for _, rule := range a.rules {
				if rule.TaskName == t.Name() {
					relevantRules = append(relevantRules, rule)
				}
			}

			// If there are relevant rules, ask the task to evaluate itself
			if len(relevantRules) > 0 {
				if msg := t.AlerterMsg(relevantRules); msg != "" {
					resultsChan <- msg
				}
			}
		}(task)
	}

	wg.Wait()
	close(resultsChan)

	// Collect all triggered alert messages
	var allMessages []string
	for msg := range resultsChan {
		allMessages = append(allMessages, msg)
	}

	if len(allMessages) == 0 {
		return // No alerts triggered, do nothing
	}

	log.Printf("Alerter evaluation completed. %d alert(s) triggered.", len(allMessages))

	// Prepare the consolidated notification body
	body := "<h1>Go2NetSpectra Alert Summary</h1>" +
		"<p>The following alerts were triggered during the last check:</p><hr>" +
		strings.Join(allMessages, "<hr>")

	// Get AI analysis for the summary if enabled
	aiAnalysis, err := a.getAIAnalysis(strings.Join(allMessages, "\n"))
	if err != nil {
		log.Printf("Failed to get AI analysis: %v", err)
	} else if aiAnalysis != "" {
		// Convert AI's markdown response to HTML
		md := []byte(aiAnalysis)
		html := markdown.ToHTML(md, nil, nil)
		body += "<hr><h2>AI-Powered Analysis</h2>" + string(html)
	}

	// Send the final notification
	if a.notifier != nil {
		subject := fmt.Sprintf("Go2NetSpectra Alert Summary (%d Triggered)", len(allMessages))
		if err := a.notifier.Send(subject, body); err != nil {
			log.Printf("ERROR: Failed to send consolidated alert notification: %v", err)
		} else {
			log.Printf("INFO: Consolidated alert notification sent successfully.")
		}
	}
}

// getAIAnalysis calls the AI service to get an analysis of the alert summary.
func (a *Alerter) getAIAnalysis(alertContent string) (string, error) {
	if !a.aiEnabled || a.aiClient == nil {
		return "", nil // AI analysis is not enabled, do nothing.
	}

	log.Println("Requesting AI analysis for alert summary...")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // 30-second timeout for AI call
	defer cancel()

	resp, err := a.aiClient.AnalyzeTraffic(ctx, &v1.AnalyzeTrafficRequest{TextInput: alertContent})
	if err != nil {
		return "", fmt.Errorf("AI service call failed: %w", err)
	}

	return resp.GetTextOutput(), nil
}