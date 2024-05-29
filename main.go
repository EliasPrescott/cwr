package main

import (
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/labstack/echo"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

func walk(s string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if ! d.IsDir() {
		println(s)
	}
	return nil
}

func validAudioExt(s string) bool {
	// todo: add more at some point
	return path.Ext(s) == ".mp3"
}

func getAudioFilePaths(startDir string) []string {
	var files []string
	filepath.WalkDir(startDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsDir() && validAudioExt(path) {
			files = append(files, path)
		}
		return nil
	})
	return files
}

func getSongTitleFromPath(s string) string {
	_, filename := path.Split(s)
	title := strings.TrimSuffix(filename, path.Ext(filename))
	return title
}

type LocalCacheResult struct {
	FilePath string
	Title string
	Extension string
}
func searchLocalSongCache(queryString string, musicDir string) []LocalCacheResult {
	var results []LocalCacheResult
	audioFilePaths := getAudioFilePaths(musicDir)
	matches := fuzzy.RankFindNormalizedFold(queryString, audioFilePaths) 
	sort.Sort(matches) 
	for _, elem := range matches {
		title := getSongTitleFromPath(elem.Target)
		results = append(results, LocalCacheResult{FilePath: elem.Target, Title: title, Extension: path.Ext(elem.Target)})
	}
	return results
}

func loadFromLocalCache(songTitle string, musicDir string) (LocalCacheResult, error) {
	audioFilePaths := getAudioFilePaths(musicDir)
	for _, elem := range audioFilePaths {
		elemTitle := getSongTitleFromPath(elem)
		if elemTitle == songTitle {
			return LocalCacheResult{
				Title: elemTitle,
				FilePath: elem,
				Extension: path.Ext(elem),
			}, nil
		}
	}
	return LocalCacheResult{}, nil
}

// returns the path to the created file or an error
func searchYtForSong(queryString string, musicDir string) (LocalCacheResult, error) {
	cmd := exec.Command("yt-dlp", "--quiet", "--extract-audio", "--audio-format", "mp3", "ytsearch:" + queryString, "-o", musicDir + queryString + ".%(ext)s", "--exec", "echo")
	output, err := cmd.Output()
	if err != nil {
		return LocalCacheResult{}, err
	}
	songPath := string(output)
	return LocalCacheResult{FilePath: songPath, Title: getSongTitleFromPath(songPath), Extension: path.Ext(songPath)}, nil
}

type Template struct {
    templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func startServer(addr string, musicDir string) error {
	t := &Template{
		templates: template.Must(template.ParseGlob("views.html")),
	}
	e := echo.New()
	e.Renderer = t
	e.Static("/songs", musicDir)
	e.Static("/fonts", "fonts")
	e.File("/styles.css", "styles.css")
	e.File("/htmx.min.js", "htmx.min.js")
	e.GET("/", func(c echo.Context) error {
		var emptyResults []LocalCacheResult
		return c.Render(200, "homePage", emptyResults)
	})
	e.GET("/search", func(c echo.Context) error {
		query := c.QueryParam("query")
		results := searchLocalSongCache(query, musicDir)
		return c.Render(200, "searchResults", results)
	})
	e.GET("/download-yt", func(c echo.Context) error {	
		query := c.QueryParam("query")
		songResult, err := searchYtForSong(query, musicDir)
		if err != nil {
			// depending on the error, this would arguably should be a 404 🤔
			return c.String(500, err.Error())
		}
		return c.Render(200, "audioSource", songResult)
	})
	e.GET("/audio-source/:title", func(c echo.Context) error {
		title := c.Param("title")
		cacheResult, err := loadFromLocalCache(title, musicDir)
		if err != nil {
			return c.String(400, err.Error()) 
		}
		return c.Render(200, "audioSource", cacheResult)
	})

	e.Logger.Fatal(e.Start(addr))
	return nil
}

func requireEnvVar(envName string) string {
	value := os.Getenv(envName)
	if value == "" {
		panic("Environment variable " + envName + " is not set")
	}
	return value
}

func main() {
	musicDir := requireEnvVar("MUSICDIR")
	if strings.HasSuffix(musicDir, "/") {
		musicDir += "/"
	}
	port := requireEnvVar("PORT")
	if _, err := strconv.Atoi(port); err != nil {
		panic("PORT env variable must be set to a valid integer")
	}
	startServer("localhost:" + port, musicDir)
}
