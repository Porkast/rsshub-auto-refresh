package job

import (
	_ "embed"
	"github.com/gogf/gf/os/grpool"
	"rsshub-auto-refresh/component"
	"rsshub-auto-refresh/model"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/database/gdb"
	"github.com/gogf/gf/encoding/ghash"
	"github.com/gogf/gf/encoding/gjson"
	"github.com/gogf/gf/os/gtime"
	"github.com/mmcdole/gofeed"
	"github.com/olivere/elastic/v7"

	"github.com/anaskhan96/soup"
	"github.com/gogf/gf/frame/g"
)

//go:embed rss_source.json
var rssResource string

type RssResourceItem struct {
	Link string
	Tags []string
}

type RouterInfoData struct {
	Route string
	Port  string
}

func RegisterJob() {
	doSync(doNonAsyncRefreshFeed)
	doSync(doSyncRSSSource)
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
			if resp = component.GetContent(apiUrl); resp == "" {
				g.Log().Error("Feed refresh cron job error ")
				continue
			}

			fp := gofeed.NewParser()
			feed, err = fp.ParseString(resp)
			if err != nil {
				g.Log().Error("Parse RSS response error : ", err)
				continue
			}
			err = AddFeedChannelAndItem(feed, router.Route, nil)
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

	resp = component.GetContent(routersAPI)
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

func doSyncRSSSource() {
	rssResourceJson := gjson.New(rssResource)
	rssResourceList := make([]RssResourceItem, 0)
	_ = rssResourceJson.GetStructs("data", &rssResourceList)
	rssResourceListLength := len(rssResourceList)

	syncGRPool := grpool.New(3)
	wg := sync.WaitGroup{}
	for index, item := range rssResourceList {
		link := item.Link
		processIndex := index
		wg.Add(1)
		_ = syncGRPool.Add(func() {
			if resp := component.GetContent(link); resp != "" {
				fp := gofeed.NewParser()
				feed, err := fp.ParseString(resp)
				if err != nil {
					g.Log().Errorf("Parse RSS %s response error : %s\n", link, err)
					wg.Done()
					return
				}

				if feed.Title == "" {
					wg.Done()
					return
				}

				err = AddFeedChannelAndItem(feed, link, item.Tags)
				if err != nil {
					g.Log().Errorf("Add feed %s channel and item error : %s\n", link, err)
					wg.Done()
					return
				}
				g.Log().Infof("Processed RSS Resource %d/%d feed refresh %s\n", processIndex, rssResourceListLength, link)
			}
			wg.Done()
		})
	}
	wg.Wait()
}

func getDescriptionThumbnail(htmlStr string) (thumbnail string) {

	docs := soup.HTMLParse(htmlStr)
	firstImgDoc := docs.Find("img")
	if firstImgDoc.Pointer != nil {
		thumbnail = firstImgDoc.Attrs()["src"]
	}

	return
}

func AddFeedChannelAndItem(feed *gofeed.Feed, rsshubLink string, tagList []string) error {

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
		feedItemID := strconv.FormatUint(ghash.RSHash64([]byte(uniString)), 32)
		feedItem.Id = feedItemID
		feedItemModeList = append(feedItemModeList, feedItem)
	}

	tagModeList := make([]model.RssFeedTag, 0)
	if tagList != nil {
		for _, tagStr := range tagList {
			if tagStr == "" {
				continue
			}
			tagModel := model.RssFeedTag{
				Name:      tagStr,
				ChannelId: feedID,
				Title:     feed.Title,
				Date:      gtime.Now(),
			}

			tagModeList = append(tagModeList, tagModel)
		}
	}

	err := component.GetDatabase().Transaction(func(tx *gdb.TX) error {
		var err error

		_, _ = tx.Save("rss_feed_channel", feedChannelModel)
		if len(tagModeList) > 0 {
			_, _ = tx.BatchInsertIgnore("rss_feed_tag", tagModeList)
		}
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
			Description:     feedItem.Description,
			Content:         feedItem.Content,
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
