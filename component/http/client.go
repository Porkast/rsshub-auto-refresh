package http

import (
	"context"
	"rsshub-auto-refresh/component/logger"
	"time"

	"github.com/gogf/gf/v2/net/gclient"
	"github.com/gogf/gf/v2/os/gfile"
)

var client *gclient.Client

func InitHttpClient(ctx context.Context) {
	client = gclient.New()
	client.SetTimeout(20 * time.Second)
}

func GetHttpClient(ctx context.Context) *gclient.Client {
	return client
}

func GetContent(link string) (resp string) {
	var (
		client *gclient.Client
	)
	ctx := context.Background()
	client = GetHttpClient(ctx)
	resp = client.SetHeaderMap(getHeaders()).GetContent(ctx, link)

	return
}

func DownloadFile(ctx context.Context, url string, dir string) error {
	var (
		resp *gclient.Response
		err  error
	)
	if resp, err = client.Get(ctx, url); err != nil {
		logger.Log().Error(ctx, err)
		return err
	}
	defer resp.Close()
	gfile.PutBytes(dir, resp.ReadAll())
	return nil
}

func getHeaders() map[string]string {
	headers := make(map[string]string)
	headers["accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9"
	headers["user-agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.135 Safari/537.36 Edg/84.0.522.63"
	return headers
}

