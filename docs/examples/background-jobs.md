# Background Jobs Example

This example shows how to implement logging for background job processing systems using zlog's signal-based approach.

## Job Processing System

```go
package main

import (
    "context"
    "fmt"
    "math/rand"
    "sync"
    "time"

    "github.com/zoobzio/zlog"
)

// Job processing signals
const (
    // Job lifecycle
    JOB_QUEUED     = "JOB_QUEUED"
    JOB_STARTED    = "JOB_STARTED" 
    JOB_COMPLETED  = "JOB_COMPLETED"
    JOB_FAILED     = "JOB_FAILED"
    JOB_RETRYING   = "JOB_RETRYING"
    JOB_ABANDONED  = "JOB_ABANDONED"
    
    // Worker events
    WORKER_STARTED   = "WORKER_STARTED"
    WORKER_STOPPED   = "WORKER_STOPPED"
    WORKER_IDLE      = "WORKER_IDLE"
    WORKER_BUSY      = "WORKER_BUSY"
    
    // Queue events
    QUEUE_HEALTH_CHECK = "QUEUE_HEALTH_CHECK"
    QUEUE_BACKLOG      = "QUEUE_BACKLOG"
    
    // Business domain jobs
    EMAIL_SENT         = "EMAIL_SENT"
    REPORT_GENERATED   = "REPORT_GENERATED"
    DATA_PROCESSED     = "DATA_PROCESSED"
    PAYMENT_PROCESSED  = "PAYMENT_PROCESSED"
)

// Job represents a unit of work
type Job struct {
    ID          string                 `json:"id"`
    Type        string                 `json:"type"`
    Priority    int                    `json:"priority"`
    Payload     map[string]interface{} `json:"payload"`
    Attempts    int                    `json:"attempts"`
    MaxAttempts int                    `json:"max_attempts"`
    CreatedAt   time.Time              `json:"created_at"`
    StartedAt   *time.Time             `json:"started_at,omitempty"`
    CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// JobProcessor defines the interface for job handlers
type JobProcessor interface {
    Process(ctx context.Context, job *Job) error
    JobType() string
}

// Worker processes jobs from the queue
type Worker struct {
    ID         string
    queue      chan *Job
    processors map[string]JobProcessor
    stopped    chan bool
    wg         sync.WaitGroup
}

// JobQueue manages job queuing and distribution
type JobQueue struct {
    jobs    chan *Job
    workers []*Worker
    metrics *QueueMetrics
}

type QueueMetrics struct {
    TotalJobs     int64
    CompletedJobs int64
    FailedJobs    int64
    mutex         sync.RWMutex
}

func main() {
    // Setup logging
    setupLogging()
    
    // Initialize job system
    queue := NewJobQueue(10, 3) // 10 buffer, 3 workers
    
    // Register job processors
    queue.RegisterProcessor(&EmailProcessor{})
    queue.RegisterProcessor(&ReportProcessor{})
    queue.RegisterProcessor(&DataProcessor{})
    
    // Start workers
    queue.Start()
    
    // Simulate job creation
    go simulateJobs(queue)
    
    // Health monitoring
    go monitorHealth(queue)
    
    // Run for demonstration
    time.Sleep(30 * time.Second)
    
    // Graceful shutdown
    queue.Stop()
    
    zlog.Info("Job system shutdown complete")
}

func setupLogging() {
    zlog.EnableStandardLogging(zlog.INFO)
    
    // Setup metrics for job processing
    setupJobMetrics()
    
    // Setup alerting for job failures
    setupJobAlerting()
    
    // Setup audit logging for sensitive jobs
    setupJobAuditLogging()
}

func NewJobQueue(bufferSize, workerCount int) *JobQueue {
    queue := &JobQueue{
        jobs:    make(chan *Job, bufferSize),
        workers: make([]*Worker, workerCount),
        metrics: &QueueMetrics{},
    }
    
    // Create workers
    for i := 0; i < workerCount; i++ {
        worker := &Worker{
            ID:         fmt.Sprintf("worker-%d", i+1),
            queue:      queue.jobs,
            processors: make(map[string]JobProcessor),
            stopped:    make(chan bool),
        }
        queue.workers[i] = worker
    }
    
    return queue
}

func (jq *JobQueue) RegisterProcessor(processor JobProcessor) {
    for _, worker := range jq.workers {
        worker.processors[processor.JobType()] = processor
    }
    
    zlog.Info("Job processor registered",
        zlog.String("job_type", processor.JobType()))
}

func (jq *JobQueue) Start() {
    for _, worker := range jq.workers {
        worker.Start()
    }
    
    zlog.Info("Job queue started",
        zlog.Int("worker_count", len(jq.workers)),
        zlog.Int("buffer_size", cap(jq.jobs)))
}

func (jq *JobQueue) Stop() {
    // Stop accepting new jobs
    close(jq.jobs)
    
    // Stop all workers
    for _, worker := range jq.workers {
        worker.Stop()
    }
    
    zlog.Info("Job queue stopped")
}

func (jq *JobQueue) Enqueue(job *Job) error {
    job.CreatedAt = time.Now()
    
    // Log job queuing
    zlog.Emit(JOB_QUEUED, "Job queued for processing",
        zlog.String("job_id", job.ID),
        zlog.String("job_type", job.Type),
        zlog.Int("priority", job.Priority),
        zlog.Int("queue_size", len(jq.jobs)))
    
    select {
    case jq.jobs <- job:
        jq.metrics.IncrementTotal()
        return nil
    default:
        zlog.Emit(QUEUE_BACKLOG, "Job queue full, rejecting job",
            zlog.String("job_id", job.ID),
            zlog.String("job_type", job.Type),
            zlog.Int("queue_size", len(jq.jobs)))
        return fmt.Errorf("queue full")
    }
}

func (w *Worker) Start() {
    zlog.Emit(WORKER_STARTED, "Worker started",
        zlog.String("worker_id", w.ID))
    
    w.wg.Add(1)
    go w.run()
}

func (w *Worker) Stop() {
    close(w.stopped)
    w.wg.Wait()
    
    zlog.Emit(WORKER_STOPPED, "Worker stopped",
        zlog.String("worker_id", w.ID))
}

func (w *Worker) run() {
    defer w.wg.Done()
    
    for {
        select {
        case job, ok := <-w.queue:
            if !ok {
                return // Queue closed
            }
            
            w.processJob(job)
            
        case <-w.stopped:
            return
            
        default:
            // Worker is idle
            time.Sleep(1 * time.Second)
        }
    }
}

func (w *Worker) processJob(job *Job) {
    ctx := context.Background()
    startTime := time.Now()
    job.StartedAt = &startTime
    
    zlog.Emit(JOB_STARTED, "Job processing started",
        zlog.String("worker_id", w.ID),
        zlog.String("job_id", job.ID),
        zlog.String("job_type", job.Type),
        zlog.Int("attempt", job.Attempts+1),
        zlog.Int("max_attempts", job.MaxAttempts))
    
    // Find processor for job type
    processor, exists := w.processors[job.Type]
    if !exists {
        zlog.Emit(JOB_FAILED, "No processor found for job type",
            zlog.String("worker_id", w.ID),
            zlog.String("job_id", job.ID),
            zlog.String("job_type", job.Type))
        return
    }
    
    // Process the job
    err := processor.Process(ctx, job)
    
    job.Attempts++
    duration := time.Since(startTime)
    
    if err != nil {
        // Job failed
        zlog.Emit(JOB_FAILED, "Job processing failed",
            zlog.String("worker_id", w.ID),
            zlog.String("job_id", job.ID),
            zlog.String("job_type", job.Type),
            zlog.Int("attempt", job.Attempts),
            zlog.Duration("duration", duration),
            zlog.Err(err))
        
        // Retry logic
        if job.Attempts < job.MaxAttempts {
            delay := time.Duration(job.Attempts*job.Attempts) * time.Second
            
            zlog.Emit(JOB_RETRYING, "Job will be retried",
                zlog.String("job_id", job.ID),
                zlog.String("job_type", job.Type),
                zlog.Int("attempt", job.Attempts),
                zlog.Duration("retry_delay", delay))
            
            // Re-queue after delay
            time.Sleep(delay)
            select {
            case w.queue <- job:
                // Re-queued successfully
            default:
                zlog.Emit(JOB_ABANDONED, "Job abandoned - queue full during retry",
                    zlog.String("job_id", job.ID),
                    zlog.String("job_type", job.Type))
            }
        } else {
            zlog.Emit(JOB_ABANDONED, "Job abandoned after max attempts",
                zlog.String("job_id", job.ID),
                zlog.String("job_type", job.Type),
                zlog.Int("attempts", job.Attempts))
        }
    } else {
        // Job succeeded
        completedAt := time.Now()
        job.CompletedAt = &completedAt
        
        zlog.Emit(JOB_COMPLETED, "Job processing completed",
            zlog.String("worker_id", w.ID),
            zlog.String("job_id", job.ID),
            zlog.String("job_type", job.Type),
            zlog.Duration("duration", duration),
            zlog.Int("attempts", job.Attempts))
    }
}

// Job Processors

type EmailProcessor struct{}

func (e *EmailProcessor) JobType() string { return "email" }

func (e *EmailProcessor) Process(ctx context.Context, job *Job) error {
    to, _ := job.Payload["to"].(string)
    subject, _ := job.Payload["subject"].(string)
    template, _ := job.Payload["template"].(string)
    
    // Simulate email sending
    time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
    
    // Simulate occasional failures
    if rand.Float32() < 0.1 {
        return fmt.Errorf("SMTP server error")
    }
    
    zlog.Emit(EMAIL_SENT, "Email sent successfully",
        zlog.String("job_id", job.ID),
        zlog.String("to", to),
        zlog.String("subject", subject),
        zlog.String("template", template))
    
    return nil
}

type ReportProcessor struct{}

func (r *ReportProcessor) JobType() string { return "report" }

func (r *ReportProcessor) Process(ctx context.Context, job *Job) error {
    reportType, _ := job.Payload["report_type"].(string)
    userID, _ := job.Payload["user_id"].(string)
    
    // Simulate report generation
    time.Sleep(time.Duration(rand.Intn(5000)) * time.Millisecond)
    
    if rand.Float32() < 0.05 {
        return fmt.Errorf("database connection timeout")
    }
    
    reportID := fmt.Sprintf("report_%d", time.Now().UnixNano())
    
    zlog.Emit(REPORT_GENERATED, "Report generated successfully",
        zlog.String("job_id", job.ID),
        zlog.String("report_id", reportID),
        zlog.String("report_type", reportType),
        zlog.String("user_id", userID))
    
    return nil
}

type DataProcessor struct{}

func (d *DataProcessor) JobType() string { return "data_processing" }

func (d *DataProcessor) Process(ctx context.Context, job *Job) error {
    dataSource, _ := job.Payload["data_source"].(string)
    operation, _ := job.Payload["operation"].(string)
    
    // Simulate data processing
    time.Sleep(time.Duration(rand.Intn(3000)) * time.Millisecond)
    
    if rand.Float32() < 0.08 {
        return fmt.Errorf("data validation error")
    }
    
    zlog.Emit(DATA_PROCESSED, "Data processing completed",
        zlog.String("job_id", job.ID),
        zlog.String("data_source", dataSource),
        zlog.String("operation", operation))
    
    return nil
}

// Queue Metrics

func (qm *QueueMetrics) IncrementTotal() {
    qm.mutex.Lock()
    defer qm.mutex.Unlock()
    qm.TotalJobs++
}

func (qm *QueueMetrics) GetStats() (int64, int64, int64) {
    qm.mutex.RLock()
    defer qm.mutex.RUnlock()
    return qm.TotalJobs, qm.CompletedJobs, qm.FailedJobs
}

// Simulation and Monitoring

func simulateJobs(queue *JobQueue) {
    jobTypes := []string{"email", "report", "data_processing"}
    
    for i := 0; i < 20; i++ {
        jobType := jobTypes[rand.Intn(len(jobTypes))]
        
        var payload map[string]interface{}
        
        switch jobType {
        case "email":
            payload = map[string]interface{}{
                "to":       "user@example.com",
                "subject":  "Welcome to our service",
                "template": "welcome_email",
            }
        case "report":
            payload = map[string]interface{}{
                "report_type": "sales",
                "user_id":     "user_123",
            }
        case "data_processing":
            payload = map[string]interface{}{
                "data_source": "user_events",
                "operation":   "aggregate",
            }
        }
        
        job := &Job{
            ID:          fmt.Sprintf("job_%d", time.Now().UnixNano()),
            Type:        jobType,
            Priority:    rand.Intn(10),
            Payload:     payload,
            MaxAttempts: 3,
        }
        
        if err := queue.Enqueue(job); err != nil {
            zlog.Error("Failed to enqueue job",
                zlog.String("job_id", job.ID),
                zlog.Err(err))
        }
        
        time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
    }
}

func monitorHealth(queue *JobQueue) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        total, completed, failed := queue.metrics.GetStats()
        queueSize := len(queue.jobs)
        
        successRate := float64(0)
        if total > 0 {
            successRate = float64(completed) / float64(total) * 100
        }
        
        zlog.Emit(QUEUE_HEALTH_CHECK, "Queue health check",
            zlog.Int64("total_jobs", total),
            zlog.Int64("completed_jobs", completed),
            zlog.Int64("failed_jobs", failed),
            zlog.Int("queue_size", queueSize),
            zlog.Float64("success_rate", successRate))
        
        // Alert on high failure rate
        if total > 10 {
            failureRate := float64(failed) / float64(total) * 100
            if failureRate > 20 {
                zlog.Error("High job failure rate detected",
                    zlog.Float64("failure_rate", failureRate),
                    zlog.Int64("failed_jobs", failed),
                    zlog.Int64("total_jobs", total))
            }
        }
        
        // Alert on queue backlog
        if queueSize > 8 {
            zlog.Emit(QUEUE_BACKLOG, "Queue backlog detected",
                zlog.Int("queue_size", queueSize),
                zlog.Int("queue_capacity", cap(queue.jobs)))
        }
    }
}

// Logging Setup

func setupJobMetrics() {
    metricsSink := zlog.NewSink("job-metrics", func(ctx context.Context, event zlog.Event) error {
        // Send metrics to monitoring system
        switch event.Signal {
        case JOB_COMPLETED:
            // jobCompletionCounter.Inc()
        case JOB_FAILED:
            // jobFailureCounter.Inc()
        case JOB_ABANDONED:
            // jobAbandonedCounter.Inc()
        }
        return nil
    })
    
    zlog.RouteSignal(JOB_COMPLETED, metricsSink)
    zlog.RouteSignal(JOB_FAILED, metricsSink)
    zlog.RouteSignal(JOB_ABANDONED, metricsSink)
    zlog.RouteSignal(QUEUE_HEALTH_CHECK, metricsSink)
}

func setupJobAlerting() {
    alertSink := zlog.NewSink("job-alerts", func(ctx context.Context, event zlog.Event) error {
        // Send alerts for critical job events
        if event.Signal == JOB_ABANDONED || event.Signal == QUEUE_BACKLOG {
            fmt.Printf("ALERT: %s - %s\n", event.Signal, event.Message)
        }
        return nil
    })
    
    zlog.RouteSignal(JOB_ABANDONED, alertSink)
    zlog.RouteSignal(QUEUE_BACKLOG, alertSink)
    zlog.RouteSignal(zlog.ERROR, alertSink)
}

func setupJobAuditLogging() {
    auditSink := zlog.NewSink("job-audit", func(ctx context.Context, event zlog.Event) error {
        // Log sensitive job operations to audit trail
        return nil
    })
    
    zlog.RouteSignal(PAYMENT_PROCESSED, auditSink)
    zlog.RouteSignal(REPORT_GENERATED, auditSink)
}
```

## Example Output

```json
{"time":"2023-10-20T14:30:15Z","signal":"WORKER_STARTED","message":"Worker started","worker_id":"worker-1"}
{"time":"2023-10-20T14:30:15Z","signal":"JOB_QUEUED","message":"Job queued for processing","job_id":"job_1640123420123456789","job_type":"email","priority":5,"queue_size":1}
{"time":"2023-10-20T14:30:16Z","signal":"JOB_STARTED","message":"Job processing started","worker_id":"worker-1","job_id":"job_1640123420123456789","job_type":"email","attempt":1,"max_attempts":3}
{"time":"2023-10-20T14:30:17Z","signal":"EMAIL_SENT","message":"Email sent successfully","job_id":"job_1640123420123456789","to":"user@example.com","subject":"Welcome to our service","template":"welcome_email"}
{"time":"2023-10-20T14:30:17Z","signal":"JOB_COMPLETED","message":"Job processing completed","worker_id":"worker-1","job_id":"job_1640123420123456789","job_type":"email","duration":"1.2s","attempts":1}
```

This example demonstrates:

- **Job lifecycle tracking**: Complete visibility into job processing stages
- **Worker management**: Monitoring worker status and load distribution
- **Retry logic**: Automatic retries with exponential backoff and proper logging
- **Health monitoring**: Queue metrics and alerting on performance issues
- **Business domain events**: Specific signals for different job types
- **Error handling**: Structured error information with retry context
- **Audit logging**: Secure audit trail for sensitive operations

The signal-based approach makes it easy to monitor job processing health, debug failures, and route different types of events to appropriate monitoring systems.