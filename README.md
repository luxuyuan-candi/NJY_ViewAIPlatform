# Stream Capture

Capture frames from an RTSP stream via ffmpeg. If `post_url` is empty, frames are saved locally; otherwise they are POSTed to the remote endpoint.

## Requirements
- Go 1.20+ (or compatible)
- ffmpeg installed and reachable by path

## Run
```powershell
cd c:\Users\Administrator\Desktop\AI视频检测平台\github\stream
go run . -config config.yaml
```

## Configuration
Edit `stream/config.yaml`:
- `rtsp_url` (required): RTSP source URL
- `interval_ms`: capture interval in milliseconds
- `post_url`: remote endpoint; leave empty to save locally
- `ffmpeg_path`: full path to ffmpeg (use `/` or escaped `\\` on Windows)
- `request_timeout_ms`: request timeout for ffmpeg and POST
- `output_dir`: local output directory for frames (when `post_url` is empty)
- `log_dir`: directory for app/error logs

## Logs and Output
- Logs are written under `log_dir` (default `logs/`).
- Captured frames are written under `output_dir` (default `frames/`) when `post_url` is empty.

