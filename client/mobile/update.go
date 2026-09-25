package mobile

import (
	"context"
	"time"

	"statusphere-client/internal/selfupdate"
)

const (
	updateCheckTimeout    = 20 * time.Second
	updateDownloadTimeout = 5 * time.Minute
)

type Update struct {
	Version string
	rel     *selfupdate.Release
}

func CheckUpdate(current string) (*Update, error) {
	ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
	defer cancel()
	rel, err := selfupdate.LatestAndroid(ctx)
	if err != nil {
		return nil, err
	}
	if !selfupdate.Newer(rel.Version, current) {
		return nil, nil
	}
	return &Update{Version: rel.Version, rel: rel}, nil
}

func (u *Update) Download(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), updateDownloadTimeout)
	defer cancel()
	return selfupdate.Download(ctx, u.rel, path)
}
