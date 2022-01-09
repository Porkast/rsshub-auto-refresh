package job

import (
	"github.com/gogf/gf/database/gdb"
	"github.com/gogf/gf/encoding/ghash"
	"github.com/gogf/gf/encoding/gjson"
	"github.com/gogf/gf/os/gtime"
	"github.com/mmcdole/gofeed"
	"github.com/olivere/elastic/v7"
	"rsshub-auto-refresh/component"
	"rsshub-auto-refresh/model"
	"strconv"
	"strings"
	"time"

	"github.com/gogf/gf/frame/g"
)

type RouterInfoData struct {
	Route string
	Port  string
}

func RegisterJob() {
	nonAsyncRefreshFeed()
}

func nonAsyncRefreshFeed() {
	go func() {
		var freshStartTime = time.Now()
		var refreshHoldTime = time.Minute * 40

		time.Sleep(time.Second * 5)
		for {
			freshStartTime = time.Now()
			doNonAsyncRefreshFeed()
			if time.Now().Sub(freshStartTime) < refreshHoldTime {
				time.Sleep(time.Minute * 60)
			}
		}
	}()
}

func doNonAsyncRefreshFeed() {
	var (
		routerLength int
		routers      []RouterInfoData
		rsshubHost   = g.Cfg().GetString("refresher.rsshub-host")
	)
	routers = getRouterArray()
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
			if resp = component.GetHttpClient().SetHeaderMap(getHeaders()).GetContent(apiUrl); resp == "" {
				g.Log().Error("Feed refresh cron job error ")
				continue
			}

			fp := gofeed.NewParser()
			feed, err = fp.ParseString(resp)
			if err != nil {
				g.Log().Error("Parse RSS response error : ", err)
				continue
			}
			err = AddFeedChannelAndItem(feed, router.Route)
			if err != nil {
				g.Log().Error("Add feed channel and item error : ", err)
				continue
			}
			g.Log().Infof("Processed %d/%d feed refresh\n", index, routerLength)
		}
	}
}

func getRouterArray() (routers []RouterInfoData) {
	var (
		rsshubHost = g.Cfg().GetString("refresher.rsshub-host")
		routersAPI = rsshubHost + "/rss/api/v1/routers"
		resp       string
		err        error
		jsonResp   *gjson.Json
	)

	resp = component.GetHttpClient().GetContent(routersAPI)
	if resp == "" {
		g.Log().Error("Get router list error ")
	}
	jsonResp = gjson.New(resp)
	err = jsonResp.GetStructs("data", &routers)
	if err != nil {
		g.Log().Error("Parse response error : ", err)
	}

	return
}

func AddFeedChannelAndItem(feed *gofeed.Feed, rsshubLink string) error {

	feedID := strconv.FormatUint(ghash.RSHash64([]byte(feed.Link+feed.Title)), 32)
	feedChannelModel := model.RssFeedChannel{
		Id:          feedID,
		Title:       feed.Title,
		ChannelDesc: feed.Description,
		Link:        feed.Link,
		RsshubLink:  rsshubLink,
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

		if len(item.Authors) > 0 {
			author = item.Authors[0].Name
		}

		feedItem := model.RssFeedItem{
			ChannelId:   feedID,
			Title:       item.Title,
			ChannelDesc: item.Description,
			Link:        item.Link,
			Date:        gtime.New(item.Published),
			Author:      author,
			InputDate:   gtime.Now(),
			Thumbnail:   thumbnail,
		}
		uniString := feedItem.Link + feedItem.Title
		feedItemID := strconv.FormatUint(ghash.RSHash64([]byte(uniString)), 32)
		feedItem.Id = feedItemID
		feedItemModeList = append(feedItemModeList, feedItem)
	}

	err := component.GetDatabase().Transaction(func(tx *gdb.TX) error {
		var err error

		_, _ = tx.Save("rss_feed_channel", feedChannelModel)
		_, err = tx.BatchInsertIgnore("rss_feed_item", feedItemModeList)

		return err
	})
	if err != nil {
		g.Log().Error("insert rss feed data failed : ", err)
	}

	bulkRequest := component.GetESClient().Bulk()
	for _, feedItem := range feedItemModeList {
		esFeedItem := model.RssFeedItemESData{
			Id:              feedItem.Id,
			ChannelId:       feedItem.ChannelId,
			Title:           feedItem.Title,
			ChannelDesc:     feedItem.ChannelDesc,
			Thumbnail:       feedItem.Thumbnail,
			Link:            feedItem.Link,
			Date:            feedItem.Date,
			Author:          feedItem.Author,
			InputDate:       feedItem.InputDate,
			ChannelImageUrl: feedChannelModel.ImageUrl,
			ChannelTitle:    feedChannelModel.Title,
			ChannelLink:     feedChannelModel.Link,
		}
		indexReq := elastic.NewBulkIndexRequest().Index("rss_item").Id(feedItem.Id).Doc(esFeedItem)
		bulkRequest.Add(indexReq)
	}
	resp, err := bulkRequest.Do(component.GetESContext())
	if err != nil || resp.Errors {
		respStr := gjson.New(resp)
		g.Log().Errorf("bulk index request failed\nError message : %s \nResponse : %s", err, respStr)
	}

	return err
}

func getHeaders() map[string]string {
	headers := make(map[string]string)
	headers["accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9"
	headers["user-agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.135 Safari/537.36 Edg/84.0.522.63"
	return headers
}
