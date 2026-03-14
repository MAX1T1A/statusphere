package linux

import (
	"fmt"
	"io"
	"net/http"
	"statusphere-client/internal/models"
	"strings"
	"time"
)

func Weather() func(models.Snapshot) {
	var (
		cached    string
		fetchedAt time.Time
	)

	return func(snap models.Snapshot) {
		if cached != "" && time.Since(fetchedAt) < 30*time.Minute {
			snap["weather"] = cached
			return
		}

		go func() {
			resp, err := (&http.Client{Timeout: 5 * time.Second}).Get("https://wttr.in/?format=%l:+%c%t")
			if err != nil {
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return
			}

			text := strings.TrimSpace(string(body))
			if text == "" || strings.Contains(text, "Unknown") {
				return
			}

			parts := strings.SplitN(text, ": ", 2)
			if len(parts) == 2 {
				text = fmt.Sprintf("%s · %s", parts[0], strings.TrimSpace(parts[1]))
			}

			cached = text
			fetchedAt = time.Now()
		}()

		if cached != "" {
			snap["weather"] = cached
		}
	}
}
