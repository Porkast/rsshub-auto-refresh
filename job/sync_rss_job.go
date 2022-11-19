package job

import (
	"context"
	"rsshub-auto-refresh/component/database"
	"rsshub-auto-refresh/component/http"
	"rsshub-auto-refresh/component/logger"
	"rsshub-auto-refresh/config"
	"rsshub-auto-refresh/model"
	"strconv"
	"strings"
	"time"

	"github.com/anaskhan96/soup"
	"github.com/gogf/gf/v2/encoding/ghash"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/mmcdole/gofeed"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func RegisterJob() {
	doSync(doNonAsyncRefreshRSSHub)
}

func doNonAsyncRefreshRSSHub() {
	var (
		ctx          context.Context = context.Background()
		routerLength int             = 0
		routers      []RouterInfoData
		rsshubHost   string = config.GetConfig().Get("rsshub.baseUrl").String()
	)

	routers = getRouterArray(ctx)
	if len(routers) > 0 {
		routerLength = len(routers)
		for index, router := range routers {
			if strings.Contains(router.Route, ":") || strings.Contains(router.Route, "rss/api/") {
				continue
			}
			var (
				apiUrl string
				resp   string
				err    error
				feed   *gofeed.Feed
			)
			apiUrl = rsshubHost + router.Route
			if resp = http.GetContent(apiUrl); resp == "" {
				logger.Log().Error(ctx, "Feed refresh cron job error ")
				continue
			}
			fp := gofeed.NewParser()
			feed, err = fp.ParseString(resp)
			if err != nil {
				logger.Log().Error(ctx, "Parse RSS response error : ", err)
				continue
			}

			err = AddFeedChannelAndItem(ctx, feed, router.Route, nil)
			if err != nil {
				logger.Log().Error(ctx, "Add feed channel and item error : ", err)
				continue
			}
			logger.Log().Infof(ctx, "Processed %d/%d feed refresh\n", index, routerLength)

		}
	}
}

func doSync(f func()) {
	go func() {
		var freshStartTime = time.Now()
		var refreshHoldTime = time.Minute * 40
		for {
			freshStartTime = time.Now()
			f()
			if time.Now().Sub(freshStartTime) < refreshHoldTime {
				time.Sleep(time.Minute * 60)
			}
		}
	}()
}

func getRouterArray(ctx context.Context) (routers []RouterInfoData) {
	var (
		rsshubHost = config.GetConfig().Get("rsshub.baseUrl").String()
		routersAPI = rsshubHost + "/rss/api/v1/routers"
		resp       string
		err        error
		jsonResp   *gjson.Json
	)

	resp = http.GetContent(routersAPI)
	if resp == "" {
		logger.Log().Error(ctx, "Get router list error ")
	}
	jsonResp = gjson.New(resp)
	err = jsonResp.Scan(routers)
	if err != nil {
		logger.Log().Error(ctx, "Parse response error : ", err)
	}

	return
}

func getDescriptionThumbnail(htmlStr string) (thumbnail string) {

	docs := soup.HTMLParse(htmlStr)
	firstImgDoc := docs.Find("img")
	if firstImgDoc.Pointer != nil {
		thumbnail = firstImgDoc.Attrs()["src"]
	}

	return
}

func AddFeedChannelAndItem(ctx context.Context,feed *gofeed.Feed, rsshubLink string, tagList []string) error {

	feedID := strconv.FormatUint(ghash.RS64([]byte(feed.Link+feed.Title)), 32)
	feedChannelModel := model.RssFeedChannel{
		Id:          feedID,
		Title:       feed.Title,
		ChannelDesc: feed.Description,
		Link:        feed.Link,
	}
	if feed.Image != nil {
		feedChannelModel.ImageUrl = feed.Image.URL
	}

	feedItemModeList := make([]model.RssFeedItem, 0)
	for _, item := range feed.Items {
		var (
			thumbnail string
			author    string
		)
		if len(item.Enclosures) > 0 {
			thumbnail = item.Enclosures[0].URL
		}

		if thumbnail == "" {
			thumbnail = getDescriptionThumbnail(item.Description)
		}

		if len(item.Authors) > 0 {
			author = item.Authors[0].Name
		}

		feedItem := model.RssFeedItem{
			ChannelId:   feedID,
			Title:       item.Title,
			Description: item.Description,
			Content:     item.Content,
			Link:        item.Link,
			Date:        gtime.New(item.Published),
			Author:      author,
			InputDate:   gtime.Now(),
			Thumbnail:   thumbnail,
		}
		uniString := feedItem.Link + feedItem.Title
		feedItemID := strconv.FormatUint(ghash.RS64([]byte(uniString)), 32)
		feedItem.Id = feedItemID
		feedItemModeList = append(feedItemModeList, feedItem)
	}

	err := database.GetDatabase().Transaction(func(tx *gorm.DB) error {
		var err error

		_ = tx.Create(clause.OnConflict{
			UpdateAll: true,
		}).Create(&feedChannelModel)

		err = tx.Create(&feedItemModeList).Error
		if err != nil {
			return err
		}

		return err
	})
	if err != nil {
		logger.Log().Error(ctx, "insert rss feed data failed : ", err)
	}

	return err
}
