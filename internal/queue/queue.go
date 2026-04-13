package queue

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"

	"notify/internal/config"
	"notify/internal/service"
)

type Task struct {
	ID        string
	Channel   service.Channel
	Target    string
	Message   any
	Attempts  int
	CreatedAt time.Time
	LastError string
}

type targetQueue struct {
	key     string
	tasks   chan *Task
	limiter *rate.Limiter
	svc     service.NotifyService
}

type Manager struct {
	queues  map[string]*targetQueue
	mu      sync.Mutex
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	cfg     config.QueueConfig
	taskSeq atomic.Uint64
}

var manager *Manager

func Init(cfg config.QueueConfig) {
	ctx, cancel := context.WithCancel(context.Background())
	manager = &Manager{
		queues: make(map[string]*targetQueue),
		ctx:    ctx,
		cancel: cancel,
		cfg:    cfg,
	}
}

func GetManager() *Manager {
	return manager
}

func (m *Manager) Enqueue(channel service.Channel, target string, message any) {
	seq := m.taskSeq.Add(1)
	taskID := fmt.Sprintf("task_%d_%d", time.Now().UnixNano(), seq)

	task := &Task{
		ID:        taskID,
		Channel:   channel,
		Target:    target,
		Message:   message,
		Attempts:  0,
		CreatedAt: time.Now(),
	}

	key := fmt.Sprintf("%s:%s", channel, target)

	m.mu.Lock()
	tq, exists := m.queues[key]
	if !exists {
		svc, err := service.GetService(channel)
		if err != nil {
			m.mu.Unlock()
			slog.Error("Failed to get service for queue", "channel", channel, "error", err)
			return
		}

		tq = &targetQueue{
			key:     key,
			tasks:   make(chan *Task, m.cfg.BufferSize),
			limiter: rate.NewLimiter(rate.Limit(m.cfg.RatePerSecond), 1),
			svc:     svc,
		}
		m.queues[key] = tq

		m.wg.Add(1)
		go m.runWorker(tq)
	}

	// Send to channel while holding the lock to prevent race with worker shutdown
	select {
	case tq.tasks <- task:
		slog.Info("Task enqueued", "taskId", taskID, "channel", channel, "target", target)
	default:
		slog.Warn("Queue full, task dropped", "taskId", taskID, "channel", channel, "target", target)
	}
	m.mu.Unlock()
}

func (m *Manager) runWorker(tq *targetQueue) {
	defer m.wg.Done()

	idleTimer := time.NewTimer(m.cfg.IdleTimeout)
	defer idleTimer.Stop()

	for {
		select {
		case <-m.ctx.Done():
			m.drainQueue(tq)
			return

		case task := <-tq.tasks:
			idleTimer.Reset(m.cfg.IdleTimeout)
			m.processTask(m.ctx, tq, task)

		case <-idleTimer.C:
			m.mu.Lock()
			// Check if queue is empty while holding lock
			if len(tq.tasks) == 0 {
				delete(m.queues, tq.key)
				m.mu.Unlock()
				slog.Info("Queue idle, closing", "key", tq.key)
				return
			}
			m.mu.Unlock()
			idleTimer.Reset(m.cfg.IdleTimeout)
		}
	}
}

func (m *Manager) processTask(ctx context.Context, tq *targetQueue, task *Task) {
	if err := tq.limiter.Wait(ctx); err != nil {
		return
	}

	for task.Attempts < m.cfg.MaxAttempts {
		task.Attempts++
		_, err := tq.svc.SendRawMessage(task.Target, task.Message)
		if err == nil {
			slog.Info("Message sent", "attempt", task.Attempts)
			return
		}
		task.LastError = err.Error()
		slog.Warn("Send failed", "taskId", task.ID, "attempt", task.Attempts, "error", err)

		if task.Attempts < m.cfg.MaxAttempts {
			// Exponential backoff: delay = RetryDelay * 2^(attempts-1)
			backoffFactor := 1 << (task.Attempts - 1)
			delay := m.cfg.RetryDelay * time.Duration(backoffFactor)

			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
		}
	}
	slog.Error("Send failed after retries", "taskId", task.ID, "attempts", m.cfg.MaxAttempts, "lastError", task.LastError)
}

func (m *Manager) drainQueue(tq *targetQueue) {
	// Skip rate limiter during shutdown drain — send remaining tasks as fast as possible.
	for {
		select {
		case task := <-tq.tasks:
			for task.Attempts < m.cfg.MaxAttempts {
				task.Attempts++
				_, err := tq.svc.SendRawMessage(task.Target, task.Message)
				if err == nil {
					slog.Info("Message sent during drain", "attempt", task.Attempts)
					break
				}
				task.LastError = err.Error()
				slog.Warn("Send failed during drain", "taskId", task.ID, "attempt", task.Attempts, "error", err)
			}
			if task.LastError != "" && task.Attempts >= m.cfg.MaxAttempts {
				slog.Error("Send failed after retries during drain", "taskId", task.ID, "attempts", task.Attempts, "lastError", task.LastError)
			}
		default:
			return
		}
	}
}

func (m *Manager) Shutdown() {
	slog.Info("Shutting down queue manager...")
	m.cancel()
	m.wg.Wait()
	slog.Info("Queue manager shutdown complete")
}
