package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Config - конфигурация загрузчика
type Config struct {
	StartURL   string
	Depth      int
	OutputDir  string
	MaxWorkers int
	SameDomain bool
	TimeoutSec int
}

// DownloadTask - одна задача для очереди (URL + глубина)
type DownloadTask struct {
	URL   string
	Depth int
}

// Downloader - основной объект загрузчика
type Downloader struct {
	cfg      Config
	client   *http.Client
	queue    chan DownloadTask
	workersW sync.WaitGroup
	taskW    sync.WaitGroup
	visited  *URLSet
	baseHost string
}

// NewDownloader - создаёт Downloader с http.Client и каналами очереди
func NewDownloader(cfg Config) (*Downloader, error) {
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = 4
	}
	if cfg.Depth < 0 {
		cfg.Depth = 0
	}
	parsed, err := url.Parse(cfg.StartURL)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout: time.Duration(cfg.TimeoutSec) * time.Second,
	}
	d := &Downloader{
		cfg:      cfg,
		client:   client,
		queue:    make(chan DownloadTask, 500),
		visited:  NewURLSet(),
		baseHost: parsed.Host,
	}
	return d, nil
}

// Start - запускает пул воркеров и очередь задач
func (d *Downloader) Start() error {
	for i := 0; i < d.cfg.MaxWorkers; i++ {
		d.workersW.Add(1)
		go d.worker(i)
	}
	start := normalizeURL(d.cfg.StartURL)
	if start == "" {
		return errors.New("invalid start url")
	}
	if d.visited.Add(start) {
		d.taskW.Add(1)
		d.queue <- DownloadTask{URL: start, Depth: 0}
	}
	go func() {
		d.taskW.Wait()
		close(d.queue)
	}()
	d.workersW.Wait()
	return nil
}

// worker - функция воркера, обрабатывает задачи из канала
func (d *Downloader) worker(id int) {
	defer d.workersW.Done()
	for task := range d.queue {
		d.processTask(id, task)
		d.taskW.Done()
	}
}

// processTask - скачивает страницу/ресурс, сохраняет локально, парсит HTML ссылки
func (d *Downloader) processTask(workerID int, task DownloadTask) {
	log.Printf("[%d] GET %s (depth=%d)\n", workerID, task.URL, task.Depth)
	body, contentType, err := d.fetch(task.URL)
	if err != nil {
		log.Printf("[%d] error fetch %s: %v\n", workerID, task.URL, err)
		return
	}

	isHTML := isHTMLByType(contentType, body)
	resType := getResourceType(task.URL)
	localPath, err := urlToLocalPath(task.URL, d.cfg.OutputDir, resType)
	if err != nil {
		log.Printf("[%d] urlToLocalPath error: %v\n", workerID, err)
		return
	}

	if err := ensureDir(filepath.Dir(localPath)); err != nil {
		log.Printf("[%d] ensureDir error: %v\n", workerID, err)
		return
	}

	if err := writeFileAtomic(localPath, body); err != nil {
		log.Printf("[%d] save file error: %v\n", workerID, err)
		return
	}

	if isHTML && task.Depth < d.cfg.Depth {
		links, updated, err := ExtractAndRewriteLinks(body, task.URL, d.cfg.OutputDir)
		if err != nil {
			log.Printf("[%d] parse/update links error: %v\n", workerID, err)
		} else {
			if err := writeFileAtomic(localPath, updated); err != nil {
				log.Printf("[%d] write updated html error: %v\n", workerID, err)
			}
		}
		for _, l := range links {
			n := normalizeURL(l)
			if n == "" {
				continue
			}
			if d.cfg.SameDomain {
				u, err := url.Parse(n)
				if err != nil || u.Host != d.baseHost {
					continue
				}
			}
			if !(strings.HasPrefix(n, "http://") || strings.HasPrefix(n, "https://")) {
				continue
			}
			if d.visited.Add(n) {
				d.taskW.Add(1)
				d.queue <- DownloadTask{URL: n, Depth: task.Depth + 1}
			}
		}
	}
}

// fetch - выполняет GET-запрос, возвращает тело и Content-Type
func (d *Downloader) fetch(u string) ([]byte, string, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "WgetMirror/1.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", &httpStatusError{Code: resp.StatusCode}
	}

	body, err := readAllLimit(resp.Body, 50*1024*1024)
	if err != nil {
		return nil, "", err
	}
	return body, resp.Header.Get("Content-Type"), nil
}

// httpStatusError - ошибка HTTP кода
type httpStatusError struct{ Code int }

func (h *httpStatusError) Error() string { return "http status " + strconvItoa(h.Code) }

// strconvItoa - маленькая замена strconv.Itoa
func strconvItoa(i int) string { return fmt.Sprintf("%d", i) }
