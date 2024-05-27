# CWR (Caching Web Radio)

a self-hosted web music player that caches previously played songs locally

### Running

This is currently just a personal project with little visual polish, so I haven't put any effort into packaging it for deployment.

If you want to run it for some reason, just `git clone` the repo and run something like `MUSICDIR=~/Music/ PORT=8080 go run main.go`

### Architecture

- Web Search: yt-dlp
- Audio Extraction: ffmpeg + ffprobe (possibly, I haven't tested this)
- Glue Code: golang
- *Blazingly Fast* Interactivity: [htmx](https://htmx.org/)
