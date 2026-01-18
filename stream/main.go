package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/yaml.v3"

	"stream/logger"
)

type Config struct {
	RTSPURL          string `yaml:"rtsp_url"`
	IntervalMS       int    `yaml:"interval_ms"`
	PostURL          string `yaml:"post_url"`
	FFmpegPath       string `yaml:"ffmpeg_path"`
	RequestTimeoutMS int    `yaml:"request_timeout_ms"`
	OutputDir        string `yaml:"output_dir"`
	LogDir           string `yaml:"log_dir"`
}

func loadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.FFmpegPath == "" {
		if runtime.GOOS == "windows" {
			cfg.FFmpegPath = "ffmpeg.exe"
		} else {
			cfg.FFmpegPath = "ffmpeg"
		}
	}
	if cfg.IntervalMS <= 0 {
		cfg.IntervalMS = 1000
	}
	if cfg.RequestTimeoutMS <= 0 {
		cfg.RequestTimeoutMS = 5000
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "frames"
	}
	if cfg.LogDir == "" {
		cfg.LogDir = "logs"
	}
	return cfg, nil
}

func captureFrame(ctx context.Context, cfg Config) ([]byte, error) {
	cmd := exec.CommandContext(ctx, cfg.FFmpegPath,
		"-rtsp_transport", "tcp",
		"-i", cfg.RTSPURL,
		"-frames:v", "1",
		"-f", "image2pipe",
		"-vcodec", "mjpeg",
		"pipe:1",
	)
	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg error: %w: %s", err, errBuf.String())
	}
	return out.Bytes(), nil
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "path to config file")
	flag.Parse()

	logger.Setup("")

	cfg, err := loadConfig(configPath)
	if err != nil {
		logger.Error.Fatalf("load config: %v", err)
	}
	logger.Setup(cfg.LogDir)
	if cfg.RTSPURL == "" {
		logger.Error.Fatal("rtsp_url must be set")
	}

	ticker := time.NewTicker(time.Duration(cfg.IntervalMS) * time.Millisecond)
	defer ticker.Stop()

	logger.Info.Printf("starting capture loop, interval=%dms", cfg.IntervalMS)
	for {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.RequestTimeoutMS)*time.Millisecond)
		frame, err := captureFrame(ctx, cfg)
		cancel()
		if err != nil {
			logger.Error.Printf("capture failed: %v", err)
		} else {
			if cfg.PostURL == "" {
				if err := saveImage(cfg, frame); err != nil {
					logger.Error.Printf("save failed: %v", err)
				} else {
					logger.Info.Printf("saved frame, size=%d bytes, cost=%s", len(frame), time.Since(start))
				}
			} else {
				if err := sendImage(cfg, frame); err != nil {
					logger.Error.Printf("post failed: %v", err)
				} else {
					logger.Info.Printf("posted frame, size=%d bytes, cost=%s", len(frame), time.Since(start))
				}
			}
		}

		<-ticker.C
	}
}

func saveImage(cfg Config, data []byte) error {
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return err
	}
	name := time.Now().Format("20060102_150405.000") + ".jpg"
	path := filepath.Join(cfg.OutputDir, name)
	return os.WriteFile(path, data, 0o644)
}

func sendImage(cfg Config, data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.RequestTimeoutMS)*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.PostURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "image/jpeg")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %s: %s", resp.Status, string(body))
	}
	return nil
}
